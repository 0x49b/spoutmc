import useWebSocket, {ReadyState} from "react-use-websocket";
import {useEffect, useState} from "react";
import {CommandType, Reply} from "@app/model/command";
import {store} from "@app/store/store";
import {setServer, setServers} from "@app/store/serverSlice";
import {setMessage} from "@app/store/messageSlice";
import {setSocketState} from "@app/store/socketSlice";

const WS_URL = "ws://localhost:3000/ws/";
const RECONNECT_INTERVAL = 5000; // 5 seconds

export const useServerWebSocket = () => {
  const [shouldReconnect, setShouldReconnect] = useState(true);

  const {sendMessage, lastMessage, readyState} = useWebSocket(WS_URL, {
    shouldReconnect: () => shouldReconnect,
    reconnectAttempts: 10,
    reconnectInterval: RECONNECT_INTERVAL,
  });

  const connectionStatus = {
    [ReadyState.CONNECTING]: "Connecting",
    [ReadyState.OPEN]: "Open",
    [ReadyState.CLOSING]: "Closing",
    [ReadyState.CLOSED]: "Closed",
    [ReadyState.UNINSTANTIATED]: "Uninstantiated",
  }[readyState];

  // Update Redux store when readyState changes
  useEffect(() => {
    store.dispatch(setSocketState({readyState, readyStateString: connectionStatus}));
  }, [readyState]);

  // Handle reconnection logic
  useEffect(() => {
    if (readyState === ReadyState.CLOSED) {
      console.warn("WebSocket closed. Attempting to reconnect...");
      setTimeout(() => {
        setShouldReconnect(true);
      }, RECONNECT_INTERVAL);
    }
  }, [readyState]);

  // Process incoming WebSocket messages
  useEffect(() => {
    if (lastMessage !== null) {
      try {
        const messageJSON: Reply = JSON.parse(lastMessage.data);
        store.dispatch(setMessage(messageJSON));

        switch (messageJSON.type) {
          case CommandType.CONTAINERLIST:
            // @ts-ignore
            store.dispatch(setServers(messageJSON.data));
            break;
          case CommandType.CONTAINERDETAIL:
            // Handle additional cases if needed
            // @ts-ignore
            store.dispatch(setServer(messageJSON.data));
            break;
          default:
            console.error("Could not parse reply message");
        }
      } catch (error) {
        console.error("Error parsing WebSocket message:", error);
      }
    }
  }, [lastMessage]);

  return {sendMessage, readyState};
};
