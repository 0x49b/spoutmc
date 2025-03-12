export enum CommandType {
  START = "start",
  STOP = "stop",
  RESTART = "restart",
  CREATE = "create",
  REMOVE = "remove",
  CONTAINERLIST = "containerlist",
  HEARTBEAT = "heartbeat",
}

export interface Command {
  type: CommandType;
  message?: string;
  containerId?: string;
}

export interface Reply {
  type: CommandType;
  data?: any
}
