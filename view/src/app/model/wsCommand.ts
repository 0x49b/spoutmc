export enum WsCommandType {
  START = "start",
  STOP = "stop",
  RESTART = "restart",
  CREATE = "create",
  REMOVE = "remove",
  CONTAINERLIST = "containerlist",
  HEARTBEAT = "heartbeat",
  LOGS = "logs",
  CONTAINERDETAIL = "containerdetail",
  CONTAINERSTATS = "containerstats",
  CONTAINERSTATSLIST = "containerstatslist",
  SUBSCRIBE_CONTAINER_STATS = "subscribe_container_stats",
  UNSUBSCRIBE_CONTAINER_STATS = "unsubscribe_container_stats"
}

export interface WsCommand {
  type: WsCommandType;
  message?: string;
  containerId?: string;
}

export interface WsReply {
  type: WsCommandType;
  data?: string | string[];
  ts: number;
  containerId?: string;
}
