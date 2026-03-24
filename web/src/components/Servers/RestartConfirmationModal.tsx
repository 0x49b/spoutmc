import React from 'react';
import {Alert, Button, Modal, ModalVariant} from '@patternfly/react-core';
import {CheckCircleIcon, SyncAltIcon} from '@patternfly/react-icons';

interface RestartConfirmationModalProps {
  isOpen: boolean;
  onClose: () => void;
  onRestartNow: () => void;
  serverName: string;
  isRestarting?: boolean;
}

const RestartConfirmationModal: React.FC<RestartConfirmationModalProps> = ({
  isOpen,
  onClose,
  onRestartNow,
  serverName,
  isRestarting = false
}) => {
  return (
    <Modal
      variant={ModalVariant.small}
      title="Configuration Saved"
      titleIconVariant={CheckCircleIcon}
      isOpen={isOpen}
      onClose={onClose}
      actions={[
        <Button
          key="restart"
          variant="primary"
          onClick={onRestartNow}
          isLoading={isRestarting}
          isDisabled={isRestarting}
          icon={<SyncAltIcon />}
        >
          {isRestarting ? 'Restarting...' : 'Restart Now'}
        </Button>,
        <Button key="later" variant="link" onClick={onClose} isDisabled={isRestarting}>
          Restart Later
        </Button>
      ]}
    >
      <p>
        Configuration file has been saved successfully for <strong>{serverName}</strong>.
      </p>

      <Alert
        variant="info"
        isInline
        title="Restart required"
        className="pf-v6-u-mt-md"
      >
        The configuration changes will only take effect after the server restarts. Would you like to restart the server now?
      </Alert>
    </Modal>
  );
};

export default RestartConfirmationModal;
