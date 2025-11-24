import React, { useState } from 'react';
import { Modal, ModalVariant, Button, Form, FormGroup, TextInput } from '@patternfly/react-core';

interface BanPlayerModalProps {
  isOpen: boolean;
  onClose: () => void;
  playerId: string;
}

const BanPlayerModal: React.FC<BanPlayerModalProps> = ({ isOpen, onClose }) => {
  const [reason, setReason] = useState('');

  return (
    <Modal
      variant={ModalVariant.small}
      title="Ban Player"
      isOpen={isOpen}
      onClose={onClose}
      actions={[
        <Button key="ban" variant="danger" onClick={onClose}>Ban</Button>,
        <Button key="cancel" variant="link" onClick={onClose}>Cancel</Button>
      ]}
    >
      <Form>
        <FormGroup label="Ban Reason" fieldId="reason">
          <TextInput id="reason" value={reason} onChange={(_event, value) => setReason(value)} />
        </FormGroup>
      </Form>
    </Modal>
  );
};

export default BanPlayerModal;
