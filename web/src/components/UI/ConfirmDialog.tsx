import React from 'react';
import {Button, Modal, ModalVariant} from '@patternfly/react-core';
import {ExclamationTriangleIcon} from '@patternfly/react-icons';

interface ConfirmDialogProps {
  isOpen: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  onConfirm: () => void;
  onCancel: () => void;
  variant?: 'danger' | 'warning' | 'info';
}

const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
  isOpen,
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  onConfirm,
  onCancel,
  variant = 'warning'
}) => {
  const getButtonVariant = () => {
    switch (variant) {
      case 'danger':
        return 'danger';
      case 'warning':
        return 'warning';
      case 'info':
        return 'primary';
      default:
        return 'primary';
    }
  };

  return (
    <Modal
      variant={ModalVariant.small}
      title={title}
      titleIconVariant={variant === 'danger' || variant === 'warning' ? ExclamationTriangleIcon : undefined}
      isOpen={isOpen}
      onClose={onCancel}
      actions={[
        <Button key="confirm" variant={getButtonVariant()} onClick={onConfirm}>
          {confirmLabel}
        </Button>,
        <Button key="cancel" variant="link" onClick={onCancel}>
          {cancelLabel}
        </Button>
      ]}
    >
      {message}
    </Modal>
  );
};

export default ConfirmDialog;
