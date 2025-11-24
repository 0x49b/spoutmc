import React, { useState } from 'react';
import { Modal, ModalVariant, Button, Form, FormGroup, TextInput } from '@patternfly/react-core';

interface AddUserModalProps {
  isOpen: boolean;
  onClose: () => void;
}

const AddUserModal: React.FC<AddUserModalProps> = ({ isOpen, onClose }) => {
  const [email, setEmail] = useState('');

  return (
    <Modal
      variant={ModalVariant.small}
      title="Add User"
      isOpen={isOpen}
      onClose={onClose}
      actions={[
        <Button key="add" variant="primary" onClick={onClose}>Add</Button>,
        <Button key="cancel" variant="link" onClick={onClose}>Cancel</Button>
      ]}
    >
      <Form>
        <FormGroup label="Email" fieldId="email">
          <TextInput id="email" value={email} onChange={(_event, value) => setEmail(value)} />
        </FormGroup>
      </Form>
    </Modal>
  );
};

export default AddUserModal;
