export const fetchDefaultIncludedFolders = async (): Promise<string[]> => {
  const response = await fetch("/api/default-included-folders");
  const data = await response.json();
  return data.defaultIncludedFolders;
};

export const getFileContent = async (path: string): Promise<string> => {
  const response = await fetch(
    `/api/file-content?path=${encodeURIComponent(path)}`
  );
  if (!response.ok) {
    throw new Error(`Failed to fetch file content: ${response.statusText}`);
  }
  return response.text();
};
