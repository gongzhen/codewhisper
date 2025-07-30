// src/components/CodeViewer.tsx

import React, { useState, useEffect } from 'react';
import { useFolderContext } from '../context/FolderContext';
import { getFileContent } from '../apis/folderApi';
import { Spin, Typography, Alert } from 'antd';
import { useTheme } from '../context/ThemeContext';

// Lazy load the MarkdownRenderer for syntax highlighting
const MarkdownRenderer = React.lazy(() => import("./MarkdownRenderer"));

const CodeViewer: React.FC = () => {
  const { selectedKeys } = useFolderContext();
  const [content, setContent] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { isDarkMode } = useTheme();

  // This value is needed by the MarkdownRenderer
  const enableCodeApply = (window as any).enableCodeApply === 'true';

  console.log("CodeViewer selectedKeys in App:", selectedKeys);

    // in src/components/CodeViewer.tsx

  useEffect(() => {
    if (selectedKeys && selectedKeys.length === 1) {
      const filePath = selectedKeys[0] as string;
      setIsLoading(true);
      setError(null);
      setContent(null);
      
      getFileContent(filePath)
        .then(data => {
          // Check if the returned data is empty
          if (data.trim() === '') {
            setError('File is empty or could not be loaded.');
          } else {
            setContent(data);
          }
          setIsLoading(false);
        })
        .catch(() => {
          setError('Failed to load file content.');
          setIsLoading(false);
        });
    } else {
      setContent(null);
      setError(null);
    }
  }, [selectedKeys]);

  if (!selectedKeys || selectedKeys.length !== 1) {
    return null; // Don't render if not exactly one file is selected
  }

  const filePath = selectedKeys[0] as string;
  const fileName = filePath.split('/').pop() || '';
  const language = fileName.split('.').pop() || 'plaintext';

  return (
    <div style={{ padding: '16px', height: '100%', overflow: 'auto', background: isDarkMode ? '#1f1f1f' : '#fff' }}>
      <Typography.Title level={4} style={{ color: isDarkMode ? 'rgba(255, 255, 255, 0.85)' : 'rgba(0, 0, 0, 0.85)', marginBottom: '16px', borderBottom: `1px solid ${isDarkMode ? '#303030' : '#f0f0f0'}`, paddingBottom: '8px' }}>
          {fileName}
      </Typography.Title>
      {isLoading && <div style={{ display: 'flex', justifyContent: 'center', paddingTop: '20px' }}><Spin size="large" /></div>}
      {error && <Alert message={error} type="error" showIcon />}
      {content !== null && (
        <React.Suspense fallback={<Spin />}>
           {/* Add the enableCodeApply prop here */}
           <MarkdownRenderer
                markdown={`\`\`\`${language}\n${content}\n\`\`\``}
                enableCodeApply={enableCodeApply}
            />
        </React.Suspense>
      )}
    </div>
  );
};

export default CodeViewer;