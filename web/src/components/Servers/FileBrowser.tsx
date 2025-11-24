import React, { useState } from 'react';
import { TreeView, TreeViewDataItem } from '@patternfly/react-core';
import { FolderIcon, FolderOpenIcon, FileIcon } from '@patternfly/react-icons';
import { FileNode } from '../../service/apiService';

interface FileBrowserProps {
  files: FileNode;
  onFileClick: (filePath: string) => void;
  containerPath: string;
}

const FileBrowser: React.FC<FileBrowserProps> = ({ files, onFileClick, containerPath }) => {
  const formatSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const convertToTreeViewData = (node: FileNode): TreeViewDataItem => {
    const item: TreeViewDataItem = {
      name: node.name,
      id: node.path,
      icon: node.isDir ? <FolderIcon /> : <FileIcon />,
      defaultExpanded: false,
    };

    if (node.isDir && node.children) {
      item.children = node.children.map(convertToTreeViewData);
    }

    if (!node.isDir && node.size !== undefined) {
      item.customBadgeContent = formatSize(node.size);
    }

    return item;
  };

  const treeData: TreeViewDataItem[] = files.children
    ? files.children.map(convertToTreeViewData)
    : [];

  const handleSelect = (_event: React.MouseEvent, item: TreeViewDataItem) => {
    // Only handle file clicks, not directories
    const findNode = (nodes: FileNode[], path: string): FileNode | null => {
      for (const node of nodes) {
        if (node.path === path) return node;
        if (node.children) {
          const found = findNode(node.children, path);
          if (found) return found;
        }
      }
      return null;
    };

    const node = files.children ? findNode(files.children, item.id as string) : null;
    if (node && !node.isDir) {
      onFileClick(item.id as string);
    }
  };

  return (
    <div style={{
      backgroundColor: 'var(--pf-v6-global--BackgroundColor--100)',
      border: '1px solid var(--pf-v6-global--BorderColor--100)',
      borderRadius: 'var(--pf-v6-global--BorderRadius--sm)',
      overflow: 'hidden'
    }}>
      {/* Header */}
      <div style={{
        padding: 'var(--pf-v6-global--spacer--md)',
        borderBottom: '1px solid var(--pf-v6-global--BorderColor--100)',
        backgroundColor: 'var(--pf-v6-global--BackgroundColor--200)'
      }}>
        <h3 className="pf-v6-u-font-size-sm pf-v6-u-font-weight-bold">
          {containerPath}
        </h3>
      </div>

      {/* File tree */}
      <div style={{ maxHeight: '500px', overflowY: 'auto', padding: 'var(--pf-v6-global--spacer--sm)' }}>
        {treeData.length > 0 ? (
          <TreeView
            data={treeData}
            onSelect={handleSelect}
            hasSelectableNodes
          />
        ) : (
          <div style={{ textAlign: 'center', padding: '3rem', color: 'var(--pf-v6-global--Color--200)' }}>
            No files found in this volume
          </div>
        )}
      </div>
    </div>
  );
};

export default FileBrowser;
