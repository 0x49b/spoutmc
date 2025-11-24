import React from 'react';
import { Modal, ModalVariant, Button } from '@patternfly/react-core';

interface ConfigurePluginModalProps {
  isOpen: boolean;
  onClose: () => void;
  pluginId: string;
}

const ConfigurePluginModal: React.FC<ConfigurePluginModalProps> = ({ isOpen, onClose }) => {
  return (
    <Modal
      variant={ModalVariant.medium}
      title="Configure Plugin"
      isOpen={isOpen}
      onClose={onClose}
      actions={[
        <Button key="save" variant="primary" onClick={onClose}>Save</Button>,
        <Button key="cancel" variant="link" onClick={onClose}>Cancel</Button>
      ]}
    >
      <p>Plugin configuration form goes here</p>
    </Modal>
  );
};

export default ConfigurePluginModal;
