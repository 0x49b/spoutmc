import React, {useCallback, useEffect, useMemo, useState} from 'react';
import {Alert, Banner, Button, Modal, ModalVariant, Spinner} from '@patternfly/react-core';
import {ExclamationTriangleIcon} from '@patternfly/react-icons';
import {checkForUpdates, getUpdateStatus, startSelfUpdate, UpdateStatusDTO} from '../../service/apiService';
import {useAuthStore} from '../../store/authStore';
import {useNotificationStore} from '../../store/notificationStore';

const isProgressState = (state?: string): boolean =>
  state === 'checking' || state === 'downloading' || state === 'installing' || state === 'restarting';

const UpdateAvailableBanner: React.FC = () => {
  const hasRole = useAuthStore((s) => s.hasRole);
  const pushToast = useNotificationStore((s) => s.pushToast);
  const isAdmin = hasRole('admin');

  const [status, setStatus] = useState<UpdateStatusDTO | null>(null);
  const [loading, setLoading] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [starting, setStarting] = useState(false);

  const refreshStatus = useCallback(async () => {
    if (!isAdmin) return;
    try {
      const response = await getUpdateStatus();
      setStatus(response.data);
    } catch (error) {
      console.error('Failed to fetch update status:', error);
    }
  }, [isAdmin]);

  useEffect(() => {
    if (!isAdmin) {
      setStatus(null);
      return;
    }

    refreshStatus();
    const intervalId = window.setInterval(refreshStatus, 30000);
    return () => window.clearInterval(intervalId);
  }, [isAdmin, refreshStatus]);

  const onManualCheck = async () => {
    if (!isAdmin || loading) return;
    setLoading(true);
    try {
      const response = await checkForUpdates();
      setStatus(response.data);
      if (response.data.updateAvailable) {
        pushToast({
          variant: 'info',
          title: `SpoutMC ${response.data.latestVersion} is available`
        });
      } else {
        pushToast({
          variant: 'success',
          title: 'No new SpoutMC version found'
        });
      }
    } catch (error: any) {
      const message = error?.response?.data?.error || 'Failed to check for updates';
      pushToast({
        variant: 'danger',
        title: 'Update check failed',
        description: message
      });
    } finally {
      setLoading(false);
    }
  };

  const onStartUpdate = async () => {
    setStarting(true);
    try {
      await startSelfUpdate();
      pushToast({
        variant: 'warning',
        title: 'Update started',
        description: 'SpoutMC is downloading and installing the release. The server network will restart when complete.'
      });
      setConfirmOpen(false);
      await refreshStatus();
    } catch (error: any) {
      const message = error?.response?.data?.error || 'Failed to start update';
      pushToast({
        variant: 'danger',
        title: 'Could not start update',
        description: message
      });
    } finally {
      setStarting(false);
    }
  };

  const shouldShow = useMemo(() => {
    if (!isAdmin || !status?.configured) return false;
    if (status.updateAvailable) return true;
    if (status.state === 'error') return true;
    return isProgressState(status.state);
  }, [isAdmin, status]);

  if (!shouldShow || !status) return null;

  return (
    <>
      <Banner status={status.state === 'error' ? 'danger' : 'warning'} screenReaderText="Update status">
        <div style={{display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.75rem', flexWrap: 'wrap'}}>
          <div style={{display: 'flex', alignItems: 'center', gap: '0.5rem'}}>
            {isProgressState(status.state) ? <Spinner size="sm" aria-label="Update in progress" /> : <ExclamationTriangleIcon aria-hidden="true" />}
            {status.state === 'error' ? (
              <span>Update check failed: {status.lastError || 'unknown error'}.</span>
            ) : isProgressState(status.state) ? (
              <span>SpoutMC update in progress ({status.state}).</span>
            ) : (
              <span>
                New SpoutMC version available: {status.currentVersion} -&gt; {status.latestVersion}
              </span>
            )}
          </div>
          <div style={{display: 'flex', alignItems: 'center', gap: '0.5rem'}}>
            {status.releaseUrl && (
              <Button
                variant="link"
                component="a"
                href={status.releaseUrl}
                target="_blank"
                rel="noreferrer"
                isInline
              >
                Release notes
              </Button>
            )}
            {!isProgressState(status.state) && status.updateAvailable && (
              <Button variant="primary" onClick={() => setConfirmOpen(true)}>
                Start update
              </Button>
            )}
            <Button variant="secondary" onClick={onManualCheck} isDisabled={loading || isProgressState(status.state)}>
              {loading ? 'Checking...' : 'Check now'}
            </Button>
          </div>
        </div>
      </Banner>

      <Modal
        variant={ModalVariant.small}
        title="Start SpoutMC update?"
        isOpen={confirmOpen}
        onClose={() => !starting && setConfirmOpen(false)}
        titleIconVariant={ExclamationTriangleIcon}
        actions={[
          <Button key="confirm" variant="danger" onClick={onStartUpdate} isLoading={starting} isDisabled={starting}>
            {starting ? 'Starting update...' : 'Start update'}
          </Button>,
          <Button key="cancel" variant="link" onClick={() => setConfirmOpen(false)} isDisabled={starting}>
            Cancel
          </Button>
        ]}
      >
        <Alert
          variant="warning"
          isInline
          title="This update will restart the server network"
          className="pf-v6-u-mb-md"
        >
          All connected players will be disconnected while the update installs and SpoutMC restarts.
        </Alert>
        <p>
          You are about to install SpoutMC {status.latestVersion}. Continue only if this restart window is acceptable.
        </p>
      </Modal>
    </>
  );
};

export default UpdateAvailableBanner;
