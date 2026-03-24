import React from 'react';
import {Alert, AlertActionCloseButton, AlertGroup} from '@patternfly/react-core';
import {useNotificationStore} from '../../store/notificationStore';

const TOAST_TIMEOUT_MS = 5000;

/**
 * Top-right toast stack (PatternFly AlertGroup + Alert timeout).
 * @see https://www.patternfly.org/components/alert#alert-timeout
 */
const ToastHost: React.FC = () => {
  const toasts = useNotificationStore((s) => s.toasts);
  const dismissToast = useNotificationStore((s) => s.dismissToast);

  return (
    <AlertGroup isToast isLiveRegion aria-label="Notifications">
      {toasts.map((t) => (
        <Alert
          key={t.id}
          variant={t.variant}
          title={t.title}
          timeout={TOAST_TIMEOUT_MS}
          onTimeout={() => dismissToast(t.id)}
          actionClose={
            <AlertActionCloseButton
              aria-label="Close notification"
              onClose={() => dismissToast(t.id)}
            />
          }
        >
          {t.description}
        </Alert>
      ))}
    </AlertGroup>
  );
};

export default ToastHost;
