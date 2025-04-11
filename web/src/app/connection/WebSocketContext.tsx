import React, { createContext, useContext, useEffect, useState } from 'react';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import { useDispatch } from 'react-redux';
import { setSocketState } from '@app/store/socketSlice';
import { setMessage } from '@app/store/messageSlice';
import { setServer, setServers, setServersLogs, setServerStats } from '@app/store/serverSlice';
import { Subscription, WsCommand, WsCommandType, WsReply } from '@app/model/wsCommand';

const WS_URL = 'ws://localhost:3000/ws/';
const RECONNECT_INTERVAL = 1000;

const WebSocketContext = createContext<any>(null);

export const WebSocketProvider = ({ children }: { children: React.ReactNode }) => {
  const dispatch = useDispatch();
  const [shouldReconnect] = useState(true);

  const { sendMessage, lastMessage, readyState } = useWebSocket(WS_URL, {
    shouldReconnect: () => shouldReconnect,
    reconnectAttempts: 50,
    reconnectInterval: RECONNECT_INTERVAL
  });

  const connectionStatus = {
    [ReadyState.CONNECTING]: 'Connecting',
    [ReadyState.OPEN]: 'Open',
    [ReadyState.CLOSING]: 'Closing',
    [ReadyState.CLOSED]: 'Closed',
    [ReadyState.UNINSTANTIATED]: 'Uninstantiated'
  }[readyState];

  // Update Redux store with connection state
  useEffect(() => {
    dispatch(setSocketState({ readyState, readyStateString: connectionStatus }));
  }, [readyState]);

  // Handle incoming WebSocket messages globally
  useEffect(() => {
    if (lastMessage !== null) {
      try {
        const messageJSON: WsReply = JSON.parse(lastMessage.data);
        dispatch(setMessage(messageJSON));

        switch (messageJSON.type) {
          case WsCommandType.CONTAINERLIST:
            // @ts-ignore
            dispatch(setServers(messageJSON.data));
            break;
          case WsCommandType.CONTAINERDETAIL:
            // @ts-ignore
            dispatch(setServer(messageJSON.data));
            break;
          case WsCommandType.CONTAINERSTATS:
            // @ts-ignore
            dispatch(setServerStats(messageJSON.data));
            break;
          case WsCommandType.LOGS:
            dispatch(setServersLogs(messageJSON));
            break;
          default:
            console.error('Unhandled WebSocket message:', messageJSON);
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    }
  }, [lastMessage]);

  return (
    <WebSocketContext.Provider value={{ sendMessage, readyState }}>
      {children}
    </WebSocketContext.Provider>
  );
};

export const useSharedWebSocket = () => useContext(WebSocketContext);

export const registerSubscriptions = (sendMessage: (msg: string) => void, sub: Subscription[], cId?: string) => {
  const commandMessage: WsCommand = {
    type: WsCommandType.REGISTER_SUBSCRIPTIONS,
    ...(cId && { containerId: cId }),
    subscriptions: sub
  };
  sendMessage(JSON.stringify(commandMessage));
};
