import * as React from 'react';
import {useEffect, useState} from 'react';
import {Button, Flex, PageSection, Title} from '@patternfly/react-core';
import socket from "@app/connection/socketConfig";

const Dashboard: React.FunctionComponent = () => {

  const [server, setServer] = useState()

  useEffect(() => {
    socket.addEventListener("open", event => {
      socket.send("Connection established from client");
    });

    socket.addEventListener("message", event => {
      console.log("got message from server ");
      console.log(event.data);
      setServer(event.data);
    });
  }, []);

  return (
    <PageSection hasBodyWrapper={false}>
      <Title headingLevel="h1" size="lg">Dashboard Page Title!</Title>
      <pre>{JSON.stringify(server, null, 2)}</pre>
      <Flex columnGap={{default: 'columnGapSm'}}>
        <Button variant="primary" size="sm" onClick={()=> socket.send("server")}>
          Send socket message
        </Button>
      </Flex>

    </PageSection>
  )
};
export {Dashboard};
