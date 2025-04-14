export enum WsCommandType {
  START = 'start',
  STOP = 'stop',
  RESTART = 'restart',
  CREATE = 'create',
  REMOVE = 'remove',
  CONTAINERLIST = 'containerlist',
  HEARTBEAT = 'heartbeat',
  LOGS = 'logs',
  CONTAINERDETAIL = 'containerdetail',
  CONTAINERSTATS = 'containerstats',
  CONTAINERSTATSLIST = 'containerstatslist',
  SUBSCRIBE_CONTAINER_STATS = 'subscribe_container_stats',
  UNSUBSCRIBE_CONTAINER_STATS = 'unsubscribe_container_stats',
  REGISTER_SUBSCRIPTIONS = 'register_subscription',
  UNREGISTER_SUBSCRIPTIONS = 'unregister_subscriptions',
  EXEC_REQUEST = 'exec_request',
  EXEC_RESPONSE = 'exec_response'

}

export enum Subscription {
  SUB_DETAIL = 'CONTAINERDETAIL',
  SUB_LOGS = 'CONTAINERLOGS',
  SUB_STATS = 'CONTAINERSTATS',
  SUB_LIST = 'CONTAINERLIST',
}

export interface WsCommand {
  type: WsCommandType;
  message?: string;
  containerId?: string;
  subscriptions?: string[];
}

export interface WsReply {
  type: WsCommandType;
  data?: string | string[];
  ts: number;
  containerId?: string;
}
