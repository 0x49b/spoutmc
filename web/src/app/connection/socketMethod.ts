/**
 * Get a List of all running Network Containers
 */
import {WsCommandType, WsCommand} from "@app/model/wsCommand";
import {socket} from "@app/connection/socketConfig";

const sendMessage = (commandMessage: WsCommand) => {
  socket.send(JSON.stringify(commandMessage));
}

export const getContainerList = () => {

  const commandMessage: WsCommand = {
    type: WsCommandType.CONTAINERLIST
  };
  sendMessage(commandMessage)
  return {}
}

export const startServer = (id: string) => {

  const commandMessage: WsCommand = {
    type: WsCommandType.START,
    containerId: id
  };
  sendMessage(commandMessage)
  return {}
}

export const stopServer = (id: string) => {
  const commandMessage: WsCommand = {
    type: WsCommandType.STOP,
    containerId: id
  };
  sendMessage(commandMessage)
  return {}
}

export const restartServer = (id: string) => {
  const commandMessage: WsCommand = {
    type: WsCommandType.RESTART,
    containerId: id
  };
  sendMessage(commandMessage)
  return {}
}

export const createServer = () => {
  const commandMessage: WsCommand = {
    type: WsCommandType.CREATE
  };
  sendMessage(commandMessage)
  return {}
}

export const removeServer = (id: string) => {
  const commandMessage: WsCommand = {
    type: WsCommandType.REMOVE,
    containerId: id
  };
  sendMessage(commandMessage)
  return {}
}
