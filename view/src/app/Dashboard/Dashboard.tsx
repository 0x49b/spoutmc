import * as React from 'react';
import { useEffect, useState } from 'react';
import {
  Button,
  Flex,
  Form,
  FormGroup,
  FormSelect,
  FormSelectOption,
  PageSection,
  Title
} from '@patternfly/react-core';
import socket from "@app/connection/socketConfig";

const Dashboard: React.FunctionComponent = () => {

  const [server, setServer] = useState();
  const [reloadTime, setReloadTime] = useState(5);
  const [intervalId, setIntervalId] = useState<NodeJS.Timeout | null>(null);

  useEffect(() => {
    socket.addEventListener("open", event => {
      socket.send("Connection established from client");
      socket.send("server");
    });
    socket.addEventListener("message", event => {
      console.log("got message from server ");
      console.dir(event.data);
      setServer(event.data);
    });
  }, []);

  useEffect(() => {
    if (intervalId) {
      clearInterval(intervalId);
      setIntervalId(null);
    }

    if (reloadTime > 0) {
      const newIntervalId = setInterval(() => {
        socket.send("server");
      }, reloadTime * 1000);
      setIntervalId(newIntervalId);
    }

    return () => {
      if (intervalId) clearInterval(intervalId);
    };
  }, [reloadTime]);

  const options = [
    { value: 5, label: '5 Seconds', disabled: false },
    { value: 10, label: '10 Seconds', disabled: false },
    { value: 30, label: '30 Seconds', disabled: false },
    { value: 60, label: '60 Seconds', disabled: false },
    { value: 0, label: 'Never', disabled: false },
  ];

  const handleTimeFrameChange = (_event: React.FormEvent<HTMLSelectElement>, value: string) => {
    setReloadTime(parseInt(value));
  };

  return (
    <PageSection hasBodyWrapper={false}>
      <Title headingLevel="h1" size="lg">Serverlist</Title>
      <Form label="" isHorizontal>
        <FormGroup label="Reload every" fieldId="horizontal-form-title">
          <FormSelect
            value={reloadTime}
            onChange={handleTimeFrameChange}
            id="horizontal-form-title"
            name="horizontal-form-title"
            aria-label="Your title"
          >
            {options.map((option, index) => (
              <FormSelectOption isDisabled={option.disabled} key={index} value={option.value} label={option.label} />
            ))}
          </FormSelect>
        </FormGroup>
      </Form>
      <pre>{JSON.stringify(server, null, 2)}</pre>
      <Flex columnGap={{ default: 'columnGapSm' }}>
        <Button variant="primary" size="sm" onClick={() => socket.send("server")}>
          Send socket message
        </Button>
      </Flex>
    </PageSection>
  );
};

export { Dashboard };
