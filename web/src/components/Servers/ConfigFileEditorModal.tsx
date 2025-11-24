import React, { useState, useEffect } from 'react';
import { Modal, ModalVariant, Button, Alert, Spinner } from '@patternfly/react-core';
import { SaveIcon } from '@patternfly/react-icons';
import { CodeEditor, Language } from '@patternfly/react-code-editor';
import * as api from '../../service/apiService';

interface ConfigFileEditorModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSaveSuccess: () => void;
  serverId: string;
  filename: string;
}

const ConfigFileEditorModal: React.FC<ConfigFileEditorModalProps> = ({
  isOpen, onClose, onSaveSuccess, serverId, filename
}) => {
  const [content, setContent] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen && serverId && filename) {
      loadFile();
    }
  }, [isOpen, serverId, filename]);

  const loadFile = async () => {
    setIsLoading(true);
    try {
      const response = await api.getConfigFile(serverId, filename);
      setContent(response.data.content);
    } catch (err) {
      setError('Failed to load file');
    } finally {
      setIsLoading(false);
    }
  };

  const handleSave = async () => {
    setIsSaving(true);
    try {
      await api.updateConfigFile(serverId, filename, content);
      onSaveSuccess();
      onClose();
    } catch (err) {
      setError('Failed to save file');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Modal
      variant={ModalVariant.large}
      title={`Edit ${filename}`}
      isOpen={isOpen}
      onClose={onClose}
      actions={[
        <Button key="save" variant="primary" onClick={handleSave} isLoading={isSaving} icon={<SaveIcon />}>
          Save
        </Button>,
        <Button key="cancel" variant="link" onClick={onClose}>Cancel</Button>
      ]}
    >
      {error && <Alert variant="danger" isInline title="Error" className="pf-v6-u-mb-md">{error}</Alert>}
      {isLoading ? (
        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '400px' }}>
          <Spinner size="xl" />
        </div>
      ) : (
        <CodeEditor
          code={content}
          onChange={(value) => setContent(value || '')}
          language={Language.yaml}
          height="500px"
          isLineNumbersVisible
          isReadOnly={false}
          options={{
            fontSize: 14,
            scrollBeyondLastLine: false,
            tabSize: 2,
            wordWrap: 'on',
          }}
        />
      )}
    </Modal>
  );
};

export default ConfigFileEditorModal;
