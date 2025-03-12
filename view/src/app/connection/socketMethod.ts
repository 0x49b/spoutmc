/**
 * Get a List of all running Network Containers
 */
import {CommandType, Command} from "@app/model/command";
import {socket} from "@app/connection/socketConfig";

const sendMessage = (commandMessage: Command) => {
  socket.send(JSON.stringify(commandMessage));
}

export const getContainerList = () => {

  const commandMessage: Command = {
    type: CommandType.CONTAINERLIST
  };
  sendMessage(commandMessage)
  return {}
}

export const startServer = (id: string) => {

  const commandMessage: Command = {
    type: CommandType.START,
    containerId: id
  };
  sendMessage(commandMessage)
  return {}
}

export const stopServer = (id: string) => {
  const commandMessage: Command = {
    type: CommandType.STOP,
    containerId: id
  };
  sendMessage(commandMessage)
  return {}
}

export const restartServer = (id: string) => {
  const commandMessage: Command = {
    type: CommandType.RESTART,
    containerId: id
  };
  sendMessage(commandMessage)
  return {}
}

export const createServer = () => {
  const commandMessage: Command = {
    type: CommandType.CREATE
  };
  sendMessage(commandMessage)
  return {}
}

export const removeServer = (id: string) => {
  const commandMessage: Command = {
    type: CommandType.REMOVE,
    containerId: id
  };
  sendMessage(commandMessage)
  return {}
}
