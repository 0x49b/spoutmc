// src/app/connection/MqttContext.tsx

import React, { createContext, useContext, useEffect, useRef, useState } from 'react';
import mqtt, { MqttClient } from 'mqtt';
import { useDispatch } from 'react-redux';
import { setIsConnectedStore } from '@app/store/socketSlice';
import { setMessage } from '@app/store/messageSlice';
import { WsReply } from '@app/model/wsCommand';

interface IMqttContext {
  client: MqttClient | null;
  isConnected: boolean;
  publish: (topic: string, message: string) => void;
  subscribe: (topic: string, callback: (message: string) => void) => void;
  unsubscribe: (topic: string) => void;
}

const MqttContext = createContext<IMqttContext | null>(null);


export const MqttProvider = ({ children }: { children: React.ReactNode }) => {
  const [isConnected, setIsConnected] = useState(false);
  const clientRef = useRef<MqttClient | null>(null);
  const subscribersRef = useRef<Record<string, ((msg: string) => void)[]>>({});
  const clientId = useRef<string>(`client-${Math.random().toString(36).substring(2, 10)}`);
  const dispatch = useDispatch();

  useEffect(() => {
    const client = mqtt.connect('ws://localhost:9001'); // using MQTT over WebSocket
    clientRef.current = client;

    client.on('connect', () => {
      setIsConnected(true);
      dispatch(setIsConnectedStore(true));
    });

    client.on('message', (topic, message) => {
      const msgStr = message.toString();
      const lastMessage:WsReply = JSON.parse(msgStr);
      dispatch(setMessage(lastMessage));
      if (subscribersRef.current[topic]) {
        subscribersRef.current[topic].forEach((callback) => callback(msgStr));
      }
    });

    client.on('error', (err) => {
      console.error('[MQTT] Error:', err);
    });

    return () => {
      client.end();
    };
  }, []);

  const publish = (topic: string, message: string) => {
    const payload = JSON.stringify({
      ...JSON.parse(message),
      clientId: clientId.current
    });
    clientRef.current?.publish(topic, payload);
  };

  const subscribe = (topic: string, callback: (msg: string) => void) => {
    const client = clientRef.current;
    if (!client) return;

    if (!subscribersRef.current[topic]) {
      subscribersRef.current[topic] = [];
      client.subscribe(topic);
    }

    subscribersRef.current[topic].push(callback);
  };

  const unsubscribe = (topic: string) => {
    const client = clientRef.current;
    if (!client) return;

    client.unsubscribe(topic);
    delete subscribersRef.current[topic];
  };

  return (
    <MqttContext.Provider
      value={{ client: clientRef.current, isConnected, publish, subscribe, unsubscribe }}
    >
      {children}
    </MqttContext.Provider>
  );
};

export const useMqtt = () => {
  const context = useContext(MqttContext);
  if (!context) throw new Error('useMqtt must be used within an MqttProvider');
  return context;
};
