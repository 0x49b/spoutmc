import React from 'react';
import { Modal, ModalVariant, Button } from '@patternfly/react-core';

interface EditUserModalProps {
  isOpen: boolean;
  onClose: () => void;
}

const EditUserModal: React.FC<EditUserModalProps> = ({ isOpen, onClose }) => {
  return (
    <Modal
      variant={ModalVariant.small}
      title="Edit User"
      isOpen={isOpen}
      onClose={onClose}
      actions={[
        <Button key="save" variant="primary" onClick={onClose}>Save</Button>,
        <Button key="cancel" variant="link" onClick={onClose}>Cancel</Button>
      ]}
    >
      <p>Edit user form goes here</p>
    </Modal>
  );
};

export default EditUserModal;
