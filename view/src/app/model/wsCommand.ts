export enum WsCommandType {
  START = "start",
  STOP = "stop",
  RESTART = "restart",
  CREATE = "create",
  REMOVE = "remove",
  CONTAINERLIST = "containerlist",
  HEARTBEAT = "heartbeat",
  LOGS = "log",
  CONTAINERDETAIL = "containerdetail",
  CONTAINERSTATS = "containerstats",
  CONTAINERSTATSLIST = "containerstatslist"
}

export interface WsCommand {
  type: WsCommandType;
  message?: string;
  containerId?: string;
}

export interface WsReply {
  type: WsCommandType;
  data?: string;
  ts: number;
}

export interface CreateServer {
  servername: string
  proxy?: boolean
  lobby?: boolean
}
