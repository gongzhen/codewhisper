<div align="center">
  <h1>CodeWhisper</h1>
  <p>
    An AI-powered assistant to help developers generate, modify, and understand code.
  </p>
  <p>
    <a href="https://github.com/gongzhen/codewhisper/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
    <a href="https://github.com/gongzhen/codewhisper/actions"><img src="https://img.shields.io/github/actions/workflow/status/gongzhen/codewhisper/main.yml?branch=main" alt="Build Status"></a>
  </p>
</div>

---

## 📖 Table of Contents

- [Overview](#-overview)
- [Features](#-features)
- [Screenshot](#-screenshot)
- [Tech Stack](#-tech-stack)
- [Installation](#-installation)
- [Usage](#-usage)
- [Contributing](#-contributing)
- [License](#-license)
- [Contact](#-contact)

---

##  Overview

CodeWhisper is an AI-powered project designed to assist developers in generating, modifying, and understanding code. It leverages advanced language models to streamline software development tasks, reduce boilerplate, and improve overall productivity.

---

## ✨ Features

- **Code-Aware Chat**: Engage in a conversation with an AI that has context of your selected code files. Ask questions, get explanations, and request documentation for your codebase.
- **Context-Aware File Selection**: Use the built-in file explorer to browse your project and select specific files or directories to include in the conversation, giving the AI focused context for its answers.
- **Apply AI-Suggested Code Changes**: The application can generate `diff` patches, and you can apply these suggested changes directly to your code with the click of a button.
- **Rich Content and Data Visualization**: The AI can respond with more than just text. It can generate and display complex content, including Markdown, tables, and even D3.js and Graphviz diagrams for data visualization.
- **Multi-Language Syntax Highlighting**: View code in the chat with proper syntax highlighting for a wide variety of popular programming languages.
- **Persistent Chat History**: All your conversations are saved locally in your browser's IndexedDB, allowing you to resume them at any time. You can also import and export your chat history as a JSON file.
- **Real-Time Streaming Responses**: Get instant feedback as the AI generates its response in real-time, thanks to server-sent events (SSE).

---

## 📸 Screenshot

Here's a look at the CodeWhisper interface in action:

![A screenshot of the CodeWhisper application interface](https://i.imgur.com/8V1iA8h.png)

---

## 🛠️ Tech Stack

- **Backend:** Go (Golang)
- **Frontend:** React, TypeScript, Ant Design
- **AI/ML:** OpenAI API (GPT-4 Turbo)
- **Database:** IndexedDB (in-browser)

---

## 🚀 Installation

### Prerequisites

Make sure you have the following installed on your system:

- [Go](https://golang.org/doc/install) (v1.18 or later)
- [Node.js](https://nodejs.org/) (v18.x or later)
- [npm](https://www.npmjs.com/) or [yarn](https://yarnpkg.com/)

### Steps

1.  **Clone the repository:**
    ```bash
    git clone [https://github.com/gongzhen/codewhisper.git](https://github.com/gongzhen/codewhisper.git)
    ```
2.  **Navigate to the project directory:**
    ```bash
    cd codewhisper
    ```
3.  **Install front-end dependencies:**
    ```bash
    npm install
    ```
4.  **Install back-end dependencies:**
    ```bash
    go mod tidy
    ```

---

## 💡 Usage

You will need to run the back-end server and the front-end development server in separate terminals.

1.  **Start the back-end server:**
    ```bash
    go run cmd/codewhisper/main.go --endpoint openai --model gpt-4-turbo
    ```
2.  **In a new terminal, start the front-end development server:**
    ```bash
    npm start
    ```
3.  Open your browser and navigate to `http://localhost:3000`.

---

## 🤝 Contributing

We welcome contributions! Please feel free to fork the repository, make changes, and submit a pull request.

1.  Fork the repository.
2.  Create a new branch for your feature or bug fix (`git checkout -b feature/your-feature-name`).
3.  Commit your changes (`git commit -m 'feat: add some amazing feature'`).
4.  Push to the branch (`git push origin feature/your-feature-name`).
5.  Open a pull request.

Please read our `CONTRIBUTING.md` for more details on our code of conduct and the process for submitting pull requests.

---

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

## 📬 Contact

For questions or support, please use the GitHub Issues tracker.

- **GitHub Issues:** [CodeWhisper Issues](https://github.com/gongzhen/codewhisper/issues)
- **Project Lead:** [gongzhen](https://github.com/gongzhen)