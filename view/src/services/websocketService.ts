// src/services/websocketService.ts
import useWebSocket from "react-use-websocket";
import React from "react";
import {CommandType} from "@app/model/command";
import {store} from "@app/store/store";
import {setServers} from "@app/store/serverSlice";


// WebSocket URL (adjust to your backend endpoint)
const WS_URL = 'ws://localhost:3000/ws/';

export const useServerWebSocket = () => {
  const {sendMessage, lastMessage, readyState} = useWebSocket(WS_URL);

  // Whenever a message is received from WebSocket
  React.useEffect(() => {
    if (lastMessage !== null) {
      try {
        const messageJSON = JSON.parse(lastMessage.data);

        switch (messageJSON.type) {
          case CommandType.CONTAINERLIST:
            store.dispatch(setServers(messageJSON.data));
            break
          case CommandType.CONTAINERDETAIL:
            break
          default:
            console.error("Could not parse reply message")
        }

      } catch (error) {
        console.error("Error parsing WebSocket message:", error);
      }
    }
  }, [lastMessage]);

  return {sendMessage, readyState};
};
