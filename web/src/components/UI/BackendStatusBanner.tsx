import React, {useEffect, useMemo, useRef, useState} from 'react';
import {Banner} from '@patternfly/react-core';
import {ExclamationTriangleIcon} from '@patternfly/react-icons';
import {useAuthStore} from '../../store/authStore';
import {withSSEAuth} from '../../service/apiService';

type ConnectionState = 'unknown' | 'connected' | 'disconnected';

const API_BASE_URL = 'http://localhost:3000/api/v1';

/**
 * Global backend indicator.
 *
 * Uses SSE streams to detect disconnects and relies on EventSource's internal reconnection to
 * auto-hide the banner as soon as the connection is reestablished.
 */
const BackendStatusBanner: React.FC = () => {
  const hasPermission = useAuthStore((s) => s.hasPermission);
  const canMonitorServer = hasPermission('server.list.read');
  const canMonitorPlayer = hasPermission('player.list.read');

  const [serverState, setServerState] = useState<ConnectionState>('unknown');
  const [playerState, setPlayerState] = useState<ConnectionState>('unknown');

  const serverSourceRef = useRef<EventSource | null>(null);
  const playerSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    // Permission-gated monitoring to avoid false "backend down" for users who can't access the streams.
    if (!canMonitorServer) {
      serverSourceRef.current?.close();
      serverSourceRef.current = null;
      setServerState('unknown');
      return;
    }

    const source = new EventSource(withSSEAuth(`${API_BASE_URL}/server/stream`));
    serverSourceRef.current = source;

    source.onopen = () => {
      if (serverSourceRef.current !== source) return;
      setServerState('connected');
    };

    source.onerror = () => {
      if (serverSourceRef.current !== source) return;
      setServerState('disconnected');
    };

    return () => {
      if (serverSourceRef.current !== source) return;
      source.close();
      serverSourceRef.current = null;
      setServerState('unknown');
    };
  }, [canMonitorServer]);

  useEffect(() => {
    if (!canMonitorPlayer) {
      playerSourceRef.current?.close();
      playerSourceRef.current = null;
      setPlayerState('unknown');
      return;
    }

    const source = new EventSource(withSSEAuth(`${API_BASE_URL}/player/stream`));
    playerSourceRef.current = source;

    source.onopen = () => {
      if (playerSourceRef.current !== source) return;
      setPlayerState('connected');
    };

    source.onerror = () => {
      if (playerSourceRef.current !== source) return;
      setPlayerState('disconnected');
    };

    return () => {
      if (playerSourceRef.current !== source) return;
      source.close();
      playerSourceRef.current = null;
      setPlayerState('unknown');
    };
  }, [canMonitorPlayer]);

  const shouldShow = useMemo(() => {
    const monitoredCount = (canMonitorServer ? 1 : 0) + (canMonitorPlayer ? 1 : 0);
    if (monitoredCount === 0) return false;

    // "either-sse" mode: show down only when all monitored streams are disconnected.
    const serverOk = !canMonitorServer || serverState === 'disconnected';
    const playerOk = !canMonitorPlayer || playerState === 'disconnected';
    return serverOk && playerOk;
  }, [canMonitorPlayer, canMonitorServer, playerState, serverState]);

  if (!shouldShow) return null;

  return (
    <Banner status="danger" screenReaderText="Backend disconnected">
      <div style={{display: 'flex', alignItems: 'center', gap: '0.5rem'}}>
        <ExclamationTriangleIcon aria-hidden="true" />
        <span>Backend connection lost. Attempting to reconnect...</span>
      </div>
    </Banner>
  );
};

export default BackendStatusBanner;

