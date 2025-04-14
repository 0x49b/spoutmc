import * as React from 'react';
import { useCallback, useEffect } from 'react';
import { ReadyState } from 'react-use-websocket';
import { Button, Flex, Icon, Label, PageSection, Skeleton, Title } from '@patternfly/react-core';
import { Server } from '@app/model/server';
import { Subscription, WsCommand, WsCommandType, WsReply } from '@app/model/wsCommand';
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
import StopIcon from '@patternfly/react-icons/dist/esm/icons/stop-icon';
import PlayIcon from '@patternfly/react-icons/dist/esm/icons/play-icon';
import { Link } from 'react-router-dom';
import { useDispatch, useSelector } from 'react-redux';
import { RootState } from '@app/store/store';
import { registerSubscriptions } from '@app/connection/WebSocketContext';
import { useMqtt } from '@app/connection/MqttContext';
import { setServers } from '@app/store/serverSlice';

const ServerList: React.FunctionComponent = () => {

  const { subscribe, publish, isConnected } = useMqtt();
  const [readyState, setReadyState] = React.useState<number>();

  const dispatch = useDispatch();

  useEffect(() => {
    const handleMsg = (msg: string) => {
      const messageJSON: WsReply = JSON.parse(msg);
      // @ts-ignore
      dispatch(setServers(messageJSON.data));
    };

    if (isConnected) {
      subscribe('server', handleMsg);
      setReadyState(1);
    }

    return () => {
      // Optional cleanup if needed
    };
  }, []);

  //const { sendMessage, readyState } = useSharedWebSocket();
  const sendMessage = (s: string) => {
  };
  const servers = useSelector((state: RootState) => state.server.servers);

  //Table
  const columnNames = {
    name: 'Name',
    created: 'Created',
    image: 'Image',
    ip: 'IP',
    state: 'State',
    status: 'Status'
  };

  // use this to unsubscribe from serverdetails when moving away of this component
  useEffect(() => {
    // When unmounting this component, send an unsubscribe
    return () => {
      if (readyState === ReadyState.OPEN) {

        const commandMessage: WsCommand = {
          type: WsCommandType.UNREGISTER_SUBSCRIPTIONS,
          subscriptions: [Subscription.SUB_DETAIL, Subscription.SUB_LOGS, Subscription.SUB_STATS, Subscription.SUB_LIST]
        };
        sendMessage(JSON.stringify(commandMessage));
      }
    };
  }, [readyState]);


  const stopServer = (id: string) => {
    const commandMessage: WsCommand = {
      type: WsCommandType.STOP,
      containerId: id
    };
    if (isConnected) publish('server', JSON.stringify(commandMessage));
  };

  const startServer = (id: string) => {
    const commandMessage: WsCommand = {
      type: WsCommandType.START,
      containerId: id
    };
    sendMessage(JSON.stringify(commandMessage));
  };

  const restartServer = (id: string) => {
    const commandMessage: WsCommand = {
      type: WsCommandType.RESTART,
      containerId: id
    };
    sendMessage(JSON.stringify(commandMessage));
  };

  const deleteServer = (id: string) => {
    const commandMessage: WsCommand = {
      type: WsCommandType.REMOVE,
      containerId: id
    };
    sendMessage(JSON.stringify(commandMessage));
  };

  const loadServerlist = useCallback(() => {
    const commandMessage: WsCommand = {
      type: WsCommandType.CONTAINERLIST
    };

    if (isConnected) publish('server', JSON.stringify(commandMessage));

  }, []);

  useEffect(() => {
    registerSubscriptions(sendMessage, [Subscription.SUB_LIST]);
  }, []);

  const defaultActions = (server: Server): IAction[] => [
    {
      title: 'Restart',
      onClick: () => restartServer(server.Id)
    },
    {
      title: 'Delete',
      onClick: () => deleteServer(server.Id)
    }
  ];

  return (
    <PageSection hasBodyWrapper={false}>
      <Title headingLevel="h1" size="lg">Serverlist</Title>

      {servers && servers.length > 0 ?

        <Table aria-label="Serverlist" variant={'compact'}>
          <Thead>
            <Tr>
              <Th>{columnNames.name}</Th>
              <Th>{columnNames.created}</Th>
              <Th>{columnNames.ip}</Th>
              <Th>{columnNames.image}</Th>
              <Th>{columnNames.state}</Th>
              <Th>{columnNames.status}</Th>
              <Th screenReaderText="Primary action" />
              <Th screenReaderText="Secondary action" />
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
                        <Button variant="secondary" size="sm" isDanger title={'ServerStopButton'}
                                onClick={() => {
                                  stopServer(server.Id);
                                }}>
                          <StopIcon />
                        </Button> :
                        <Button variant="secondary" size="sm" title={'ServerStartButton'}
                                onClick={() => {
                                  startServer(server.Id);
                                }}>
                          <Icon>
                            <PlayIcon />
                          </Icon>
                        </Button>}
                    </TableText>
                  </Td>

                  <Td isActionCell>
                    {rowActions ? <ActionsColumn items={rowActions} /> : null}
                  </Td>
                </Tr>
              );
            })}
          </Tbody>
        </Table>
        :

        <Skeleton screenreaderText="Loading contents" />
      }


      <Flex columnGap={{ default: 'columnGapSm' }}>
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

export { ServerList };
