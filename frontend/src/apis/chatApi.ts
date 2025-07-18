import { SetStateAction, Dispatch } from "react";
import { createParser } from "eventsource-parser";
import { message } from "antd";
import { Message } from "../utils/types";
import { db } from "../utils/db";

interface Operation {
  op: string;
  path: string;
  value?: string;
}

interface ErrorResponse {
  error: string;
  detail: string;
  event?: string;
}

const isValidMessage = (message: any) => {
  if (!message || typeof message !== "object") return false;
  if (!message.content || typeof message.content !== "string") return false;
  return message.content.trim().length > 0;
};

const cleanMessages = (messages: any[]) => {
  return messages
    .filter(isValidMessage)
    .map((msg) => ({ ...msg, content: msg.content.trim() }));
};

const isStreamingError = (error: any): boolean => {
  return (
    error instanceof Error &&
    (error.message.includes("validationException") ||
      error.message.includes("Input is too long"))
  );
};

const handleStreamError = async (response: Response): Promise<never> => {
  console.log("Handling stream error:", response.status);
  let errorMessage: string = "An unknown error occurred";

  // Try to get detailed error message from response
  try {
    const errorData = await response.json();
    errorMessage = errorData.detail || errorMessage;
  } catch (e) {
    // If we can't parse the response, keep the default error message
    console.warn("Could not parse error response:", e);
  }

  // Handle different response status codes
  switch (response.status) {
    case 413:
      console.log("Content too large error detected");
      errorMessage =
        "Selected content is too large for the model. Please reduce the number of files.";
      break;
    case 401:
      console.log("Authentication error");
      errorMessage = "Authentication failed. Please check your credentials.";
      break;
    case 503:
      console.log("Service unavailable");
      errorMessage =
        "Service is temporarily unavailable. Please try again in a moment.";
      break;
    default:
      errorMessage = "An unexpected error occurred. Please try again.";
  }
  // Always throw error to stop the streaming process
  throw new Error(errorMessage);
};

const createEventSource = (url: string, body: any): EventSource => {
  try {
    const params = new URLSearchParams();
    params.append("data", JSON.stringify(body));
    const eventSource = new EventSource(`${url}?${params}`);
    eventSource.onerror = (error) => {
      console.error("EventSource error:", error);
      eventSource.close();
    };
    eventSource.close();
    message.error({
      content: "Connection to server lost. Please try again.",
      duration: 5,
    });
    return eventSource;
  } catch (error) {
    console.error("Error creating EventSource:", error);
    throw error;
  }
};

// Fix for chatApi.ts - handling the ops format correctly
export const sendPayload = async (
  conversationId: string,
  question: string,
  isStreamingToCurrentConversation: boolean,
  messages: Message[],
  setStreamedContentMap: Dispatch<SetStateAction<Map<string, string>>>,
  setIsStreaming: (streaming: boolean) => void,
  checkedItems: string[],
  addMessageToConversation: (
    message: Message,
    targetConversationId: string,
    isNonCurrentConversation?: boolean
  ) => void,
  removeStreamingConversation: (id: string) => void,
  onStreamComplete?: (content: string) => void
) => {
  try {
    let currentContent = "";
    setIsStreaming(true);
    let response = await getApiResponse(messages, question, checkedItems);

    if (!response.ok) {
      await handleStreamError(response);
    }

    if (!response.body) {
      throw new Error("No body in response");
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder("utf-8");

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      const chunk = decoder.decode(value, { stream: true });
      const lines = chunk.split("\n");

      for (const line of lines) {
        // fix: Check for 'data: ' prefix (SSE format)
        if (line.startsWith("data: ")) {
          try {
            const jsonStr = line.slice(6).trim(); // fix: Remove 'data: ' prefix
            if (!jsonStr) continue;

            const data = JSON.parse(jsonStr);

            // fix: Handle error response
            if (data.error) {
              message.error({
                content: data.detail || "An error occurred",
                duration: 10,
                key: "stream-error",
              });
              removeStreamingConversation(conversationId);
              return "";
            }

            // fix: Process ops array to extract content
            if (data.ops && Array.isArray(data.ops)) {
              for (const op of data.ops) {
                if (
                  op.op === "add" &&
                  op.path === "/streamed_output_str/-" &&
                  op.value
                ) {
                  currentContent += op.value;
                  // fix: Update streamed content map with accumulated content
                  setStreamedContentMap((prev: Map<string, string>) => {
                    const next = new Map(prev);
                    next.set(conversationId, currentContent);
                    return next;
                  });
                }
              }
            }
          } catch (e) {
            // fix: Only log actual parse errors, not empty lines
            if (line.trim()) {
              console.error("Error parsing SSE data:", e, "Line:", line);
            }
          }
        }
      }
    }

    // fix: Finalize the stream
    if (currentContent) {
      onStreamComplete?.(currentContent);
      const aiMessage: Message = {
        role: "assistant",
        content: currentContent,
      };
      addMessageToConversation(
        aiMessage,
        conversationId,
        !isStreamingToCurrentConversation
      );
    }

    removeStreamingConversation(conversationId);
    return currentContent || "";
  } catch (error) {
    console.error("Error in sendPayload:", error);
    setIsStreaming(false);
    removeStreamingConversation(conversationId);

    if (error instanceof Error) {
      message.error({
        content: error.message,
        key: "stream-error",
        duration: 10,
      });
    }
    return "";
  }
};

async function getApiResponse(
  messages: any[],
  question: string,
  checkedItems: string[]
) {
  const messageTuples: string[][] = [];

  // Validate that we have files selected
  console.log("API Request File Selection:", {
    endpoint: "/codewhisper/stream_log",
    checkedItemsCount: checkedItems.length,
    checkedItems,
    sampleFile: checkedItems[0],
    hasD3Renderer: checkedItems.includes(
      "frontend/src/components/D3Renderer.tsx"
    ),
  });

  // Log specific file paths we're interested in
  console.log("Looking for specific files:", {
    d3Path: "frontend/src/components/D3Renderer.tsx",
    checkedItems,
    sampleFile: checkedItems[0],
  });

  console.log(
    "Messages received in getApiResponse:",
    messages.map((m) => ({
      role: m.role,
      content: m.content.substring(0, 50),
    }))
  );

  // Build pairs of human messages and AI responses
  try {
    // If this is the first message, we won't have any pairs yet
    if (messages.length === 1 && messages[0].role === "human") {
      console.log("First message in conversation, no history to send");
      console.log("Selected files being sent to server:", checkedItems);
      const response = await fetch("/codewhisper/stream_log", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          input: {
            chat_history: [],
            question: question,
            config: {
              files: checkedItems,
              // Add explicit file list for debugging
              fileList: checkedItems.join(", "),
            },
          },
        }),
      });
      if (!response.ok) {
        throw await handleStreamError(response);
      }
      return response;
    }

    // For subsequent messages, build the history pairs
    const validMessages = messages.filter((msg) => msg?.content?.trim());

    console.log(
      "Valid messages:",
      validMessages.map((m) => ({
        role: m.role,
        content: m.content.substring(0, 50),
      }))
    );

    // Build pairs from completed exchanges
    for (let i = 0; i < validMessages.length; i++) {
      const current = validMessages[i];
      const next = validMessages[i + 1];

      // Only add complete human-assistant pairs
      if (current?.role === "human" && next?.role === "assistant") {
        messageTuples.push([current.content, next.content]);
        console.log("Added pair:", {
          human: current.content.substring(0, 50),
          ai: next.content.substring(0, 50),
          humanRole: current.role,
          aiRole: next.role,
        });
        i++; // Skip the next message since we've used it
      }
    }

    console.log("Chat history pairs:", messageTuples.length);
    console.log("Current question:", question);
    console.log("Full chat history:", messageTuples);

    const payload = {
      input: {
        chat_history: messageTuples,
        question: question,
        config: {
          files: checkedItems,
          // Add explicit file list for debugging
          fileList: checkedItems.join(", "),
        },
      },
    };

    console.log("Sending payload to server:", JSON.stringify(payload, null, 2));
    const response = await fetch("/codewhisper/stream_log", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!response.ok) {
      throw await handleStreamError(response);
    }
    return response;
  } catch (error) {
    console.error("API request failed:", error);
    throw error;
  }
}
