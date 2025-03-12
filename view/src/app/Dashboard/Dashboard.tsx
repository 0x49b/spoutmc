import * as React from 'react';
import {useCallback, useEffect, useState} from 'react';
import useWebsocket, {ReadyState} from "react-use-websocket";
import {
  Button,
  Flex,
  Form,
  FormGroup,
  FormSelect,
  FormSelectOption,
  Label,
  PageSection,
  Title
} from '@patternfly/react-core';
import {Loader} from "@app/utils/Loader";
import {Server} from "@app/model/server";
import {Command, CommandType, Reply} from "@app/model/command";
import {Table, Tbody, Td, Th, Thead, Tr} from '@patternfly/react-table';

const Dashboard: React.FunctionComponent = () => {

  // Websocket
  const [socketUrl, setSocketUrl] = useState<string>('ws://localhost:3000/ws/');
  const [messageHistory, setMessageHistory] = useState<MessageEvent<any>[]>([]);

  // Server
  const [server, setServer] = useState<Server[]>([]);
  const [reloadTime, setReloadTime] = useState(5);
  const [command, setCommand] = useState()
  const [loading, setLoading] = useState(true)

  //Table
  const columnNames = {
    name: 'Name',
    state: 'State',
    status: 'Status'
  };

  const heartbeat: Command = {
    type: CommandType.HEARTBEAT
  }

  const {sendMessage, lastMessage, readyState} = useWebsocket(socketUrl, {

    heartbeat: {
      message: JSON.stringify(heartbeat),
      returnMessage: 'pong',
      timeout: 12_000,
      interval: 60_000
    }
  });

  useEffect(() => {
    if (lastMessage !== null) {
      setMessageHistory((prev) => prev.concat(lastMessage))
      messageParser(lastMessage)
    }
    setLoading(false)
  }, [lastMessage]);


  const messageParser = (message: MessageEvent<any>) => {

    const messageJSON: Reply = JSON.parse(message.data)

    switch (messageJSON.type) {
      case CommandType.CONTAINERLIST:
        updateServerList(messageJSON.data)
        break
      default:
        console.error("Could not parse reply message")
    }
  }

  const updateServerList = (serverData: any) => {
    if (!Array.isArray(serverData)) {
      console.error("serverData is not an array:", serverData);
      return;
    }
    setServer(serverData); // Directly update with array
  };

  const loadServerlist = useCallback(() => {
    const commandMessage: Command = {
      type: CommandType.CONTAINERLIST
    };
    setLoading(true)
    sendMessage(JSON.stringify(commandMessage))
  }, [])

  useEffect(() => {
    const interval = setInterval(() => {
      loadServerlist(); // Reload server list every 5 seconds
    }, 5000);
    return () => clearInterval(interval); // Cleanup on unmount
  }, [loadServerlist]); // Depend on the function to reload


  const connectionStatus = {
    [ReadyState.CONNECTING]: 'Connecting',
    [ReadyState.OPEN]: 'Open',
    [ReadyState.CLOSING]: 'Closing',
    [ReadyState.CLOSED]: 'Closed',
    [ReadyState.UNINSTANTIATED]: 'Uninstantiated',
  }[readyState];


  const reloadOptions = [
    {value: 5, label: 'every 5 Seconds', disabled: false},
    {value: 10, label: 'every 10 Seconds', disabled: false},
    {value: 30, label: 'every 30 Seconds', disabled: false},
    {value: 60, label: 'every minute', disabled: false},
    {value: 0, label: 'never', disabled: false},
  ];

  const handleTimeFrameChange = (_event: React.FormEvent<HTMLSelectElement>, value: string) => {
    setReloadTime(parseInt(value));
  };


  return (
    <PageSection hasBodyWrapper={false}>
      <Title headingLevel="h1" size="lg">Serverlist</Title>
      <Form label="" isHorizontal>
        <FormGroup label="Reload" fieldId="horizontal-form-title">
          <FormSelect
            value={reloadTime}
            onChange={handleTimeFrameChange}
            id="horizontal-form-title"
            name="horizontal-form-title"
            aria-label="Your title"
          >
            {reloadOptions.map((option, index) => (
              <FormSelectOption isDisabled={option.disabled} key={index} value={option.value}
                                label={option.label}/>
            ))}
          </FormSelect>
        </FormGroup>
      </Form>

      {loading ? <Loader/> :
        <Table aria-label="server table" variant={"compact"}>
          <Thead>
            <Tr>
              <Th>{columnNames.name}</Th>
              <Th>{columnNames.state}</Th>
              <Th>{columnNames.status}</Th>
              <Th screenReaderText="Secondary action"/>
            </Tr>
          </Thead>
          <Tbody>
            {server.map((server) => (


              <Tr key={server.Id}>
                <Td dataLabel={columnNames.name}>{server.Names[0]}</Td>
                <Td dataLabel={columnNames.state}>
                  {server.State === 'running' ?
                    <Label variant="outline" color="green">{server.State}</Label> :
                    <Label variant="outline" color="red">{server.State}</Label>
                  }
                </Td>
                <Td dataLabel={columnNames.status}>{server.Status}</Td>
                <Td isActionCell>


                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>}


      <Flex columnGap={{default: 'columnGapSm'}}>
        <Button
          onClick={loadServerlist}
          variant="primary" size="sm"
          disabled={readyState !== ReadyState.OPEN}
        >
          Reload Serverlist
        </Button>

        <span>The WebSocket is currently {connectionStatus}</span>
      </Flex>
    </PageSection>
  );
};

export {Dashboard};
