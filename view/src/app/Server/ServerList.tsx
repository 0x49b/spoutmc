import * as React from 'react';
import {useCallback, useEffect} from 'react';
import {ReadyState} from "react-use-websocket";
import {Button, Flex, Icon, Label, PageSection, Title} from '@patternfly/react-core';
import {Server} from "@app/model/server";
import {Command, CommandType} from "@app/model/command";
import {
  ActionsColumn,
  IAction,
  Table,
  TableText,
  Tbody,
  Td,
  Th,
  Thead,
  Tr
} from '@patternfly/react-table';
import StopIcon from "@patternfly/react-icons/dist/esm/icons/stop-icon";
import PlayIcon from "@patternfly/react-icons/dist/esm/icons/play-icon";
import {Link, useNavigate} from "react-router-dom";
import {useSelector} from "react-redux";
import {useServerWebSocket} from "../../services/websocketService";
import {RootState} from "@app/store/store";

const ServerList: React.FunctionComponent = () => {

  const {sendMessage} = useServerWebSocket();
  const navigate = useNavigate();

  // Store
  const servers = useSelector((state: RootState) => state.server.servers)
  const readyState = useSelector((state: RootState) => state.socket.readyState)

  //Table
  const columnNames = {
    name: 'Name',
    created: 'Created',
    image: 'Image',
    ip: 'IP',
    state: 'State',
    status: 'Status'
  };

  const loadServerlist = useCallback(() => {
    const commandMessage: Command = {
      type: CommandType.CONTAINERLIST
    };
    sendMessage(JSON.stringify(commandMessage))
  }, [])

  const stopServer = (id: string) => {
    const commandMessage: Command = {
      type: CommandType.STOP,
      containerId: id
    }
    sendMessage(JSON.stringify(commandMessage))
  }

  const startServer = (id: string) => {
    const commandMessage: Command = {
      type: CommandType.START,
      containerId: id
    }
    sendMessage(JSON.stringify(commandMessage))
  }

  const restartServer = (id: string) => {
    const commandMessage: Command = {
      type: CommandType.RESTART,
      containerId: id,
    }
    sendMessage(JSON.stringify(commandMessage))
  }

  const deleteServer = (id: string) => {
    const commandMessage: Command = {
      type: CommandType.REMOVE,
      containerId: id,
    }
    sendMessage(JSON.stringify(commandMessage))
  }

  useEffect(() => {
    loadServerlist()
  }, [loadServerlist]);

  const defaultActions = (server: Server): IAction[] => [
    {
      title: "Restart",
      onClick: () => restartServer(server.Id),
    },
    {
      title: "Delete",
      onClick: () => deleteServer(server.Id),
    },
  ];

  return (
    <PageSection hasBodyWrapper={false}>
      <Title headingLevel="h1" size="lg">Serverlist</Title>
      <Table aria-label="Serverlist" variant={"compact"}>
        <Thead>
          <Tr>
            <Th>{columnNames.name}</Th>
            <Th>{columnNames.created}</Th>
            <Th>{columnNames.ip}</Th>
            <Th>{columnNames.image}</Th>
            <Th>{columnNames.state}</Th>
            <Th>{columnNames.status}</Th>
            <Th screenReaderText="Primary action"/>
            <Th screenReaderText="Secondary action"/>
          </Tr>
        </Thead>
        <Tbody>
          {servers.map((server) => {
            let rowActions: IAction[] | null = defaultActions(server);
            return (<Tr key={server.Id}>
                <Td dataLabel={columnNames.name}>
                  <Link to={`/server/detail/${server.Id}`}>
                    {server.Config.Hostname}
                  </Link>
                </Td>
                <Td dataLabel={columnNames.created}>{server.Created}</Td>
                <Td
                  dataLabel={columnNames.ip}>{server.NetworkSettings.Networks.spoutnetwork.IPAddress}</Td>
                <Td dataLabel={columnNames.image}>{server.Config.Image}</Td>
                <Td dataLabel={columnNames.state}>
                  {server.State.Status === 'running' ?
                    <Label variant="outline" color="green">{server.State.Status}</Label> :
                    server.State.Status === 'exited' ?
                      <Label variant="outline" color="red">{server.State.Status}</Label> :
                      <Label variant="outline" color="yellow">{server.State.Status}</Label>
                  }
                </Td>
                <Td dataLabel={columnNames.status}>{server.State.Health.Status}</Td>

                <Td>
                  <TableText>
                    {server.State.Status === 'running' ?
                      <Button variant="secondary" size="sm" isDanger title={"ServerStopButton"}
                              onClick={() => {
                                stopServer(server.Id)
                              }}>
                        <StopIcon/>
                      </Button> :
                      <Button variant="secondary" size="sm" title={"ServerStartButton"}
                              onClick={() => {
                                startServer(server.Id)
                              }}>
                        <Icon>
                          <PlayIcon/>
                        </Icon>
                      </Button>}
                  </TableText>
                </Td>

                <Td isActionCell>
                  {rowActions ? <ActionsColumn items={rowActions}/> : null}
                </Td>
              </Tr>
            )
          })}
        </Tbody>
      </Table>


      <Flex columnGap={{default: 'columnGapSm'}}>
        <Button
          onClick={loadServerlist}
          variant="primary" size="sm"
          disabled={readyState !== ReadyState.OPEN}
        >
          Reload Serverlist
        </Button>
      </Flex>
    </PageSection>
  );
};

export {ServerList};
