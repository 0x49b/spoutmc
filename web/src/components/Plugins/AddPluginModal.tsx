import React from 'react';
import { Modal, ModalVariant, Button } from '@patternfly/react-core';

interface AddPluginModalProps {
  isOpen: boolean;
  onClose: () => void;
}

const AddPluginModal: React.FC<AddPluginModalProps> = ({ isOpen, onClose }) => {
  return (
    <Modal
      variant={ModalVariant.small}
      title="Add Plugin"
      isOpen={isOpen}
      onClose={onClose}
      actions={[
        <Button key="add" variant="primary" onClick={onClose}>Add</Button>,
        <Button key="cancel" variant="link" onClick={onClose}>Cancel</Button>
      ]}
    >
      <p>Plugin installation form goes here</p>
    </Modal>
  );
};

export default AddPluginModal;
