import React, {useEffect} from 'react';
import {useParams} from "react-router";
import {useServerWebSocket} from "../../../services/websocketService";
import {WsCommand, WsCommandType} from "@app/model/wsCommand";
import {ServerStats} from "@app/model/serverstats";
import {useSelector} from "react-redux";
import {RootState} from "@app/store/store";
import {Skeleton} from "@patternfly/react-core";


export const ServerDetailsStats: React.FC = () => {

  const {serverId} = useParams<{ serverId: string }>();
  const {sendMessage} = useServerWebSocket();

  const serverStats: ServerStats | undefined = useSelector((state: RootState) => state.server.serverStats);


  useEffect(() => {
    loadServerStats(); // initial call
  }, []);

  const loadServerStats = () => {
    const commandMessage: WsCommand = {
      type: WsCommandType.CONTAINERSTATS,
      containerId: serverId,
    }
    sendMessage(JSON.stringify(commandMessage));
  }


  return (
    <pre>
        {serverStats ? JSON.stringify(serverStats, null, 2) : <Skeleton/>}
      </pre>
  );
};
