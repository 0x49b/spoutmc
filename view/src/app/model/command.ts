export enum CommandType {
  START = "start",
  STOP = "stop",
  RESTART = "restart",
  CREATE = "create",
  REMOVE = "remove",
  CONTAINERLIST = "containerlist",
}

export interface Command {
  type: CommandType;
  message?: string;
  containerId?: string;
}
