import * as React from 'react';
import {useEffect, useState} from 'react';
import {Button, Flex, Form, FormGroup, FormSelect, FormSelectOption, PageSection, Title} from '@patternfly/react-core';

import {Command, CommandType} from "@app/model/command";
import {Loader} from "@app/utils/Loader";
import {socket} from '@app/connection/socketConfig';
import {Server} from "@app/model/server";

const Dashboard: React.FunctionComponent = () => {

  const [server, setServer] = useState<Server[]>([]);
  const [reloadTime, setReloadTime] = useState(5);
  const [intervalId, setIntervalId] = useState<NodeJS.Timeout | null>(null);
  const [command, setCommand] = useState<Command | undefined>()
  const [loading, setLoading] = useState(true)


  useEffect(() => {
    socket.addEventListener("message", event => {
      setServer(event.data);
    });

    const getContainerListMessage: Command = {
      type: CommandType.CONTAINERLIST
    };
    setCommand(getContainerListMessage);

    if (intervalId) {
      clearInterval(intervalId);
      setIntervalId(null);
    }

    if (reloadTime > 0) {
      const newIntervalId = setInterval(() => {
        setCommand((prevMessage) => {
          const updatedMessage = {...prevMessage};
          socket.send(JSON.stringify(updatedMessage));
          console.log(updatedMessage);
          setLoading(false)
          return updatedMessage;
        });
      }, reloadTime * 1000);

      setIntervalId(newIntervalId);
    }

    return () => {
      if (intervalId) clearInterval(intervalId);
    };
  }, [reloadTime]);


  const options = [
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
            {options.map((option, index) => (
              <FormSelectOption isDisabled={option.disabled} key={index} value={option.value} label={option.label}/>
            ))}
          </FormSelect>
        </FormGroup>
      </Form>

      {loading ? <Loader/> : <React.Fragment/>}

      <pre>{JSON.stringify(server, null, 2)}</pre>


      <Flex columnGap={{default: 'columnGapSm'}}>
        <Button variant="primary" size="sm" onClick={() => {
          setLoading(true)
          socket.send(JSON.stringify(command))
        }}>
          Reload Serverlist
        </Button>
      </Flex>
    </PageSection>
  );
};

export {Dashboard};
