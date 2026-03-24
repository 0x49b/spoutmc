import React from 'react';
import {Button, Modal, ModalVariant} from '@patternfly/react-core';

interface ViewBanDialogProps {
  isOpen: boolean;
  onClose: () => void;
  banDetails?: any;
}

const ViewBanDialog: React.FC<ViewBanDialogProps> = ({ isOpen, onClose, banDetails }) => {
  return (
    <Modal
      variant={ModalVariant.small}
      title="Ban Details"
      isOpen={isOpen}
      onClose={onClose}
      actions={[
        <Button key="close" variant="primary" onClick={onClose}>Close</Button>
      ]}
    >
      <p>Ban details: {JSON.stringify(banDetails)}</p>
    </Modal>
  );
};

export default ViewBanDialog;
