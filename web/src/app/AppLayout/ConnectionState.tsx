import * as React from 'react';
import { ReactElement, useEffect, useState } from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '@app/store/store';
import { Button, ButtonVariant, Icon, Tooltip } from '@patternfly/react-core';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';

const ConnectionState: React.FunctionComponent = () => {

  const lastMessage = useSelector((state: RootState) => state.message.lastMessage);
  const readyStateString = useSelector((state: RootState) => state.socket.readyStateString);
  const readyState = useSelector((state: RootState) => state.socket.readyState);
  const isConnected = useSelector((state: RootState) => state.socket.isConnected);

  const [stateIcon, setStateIcon] = useState<ReactElement>(<></>);

  const formatTimestamp = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleString('de-CH', {
      dateStyle: 'short', timeStyle: 'medium'
    });
  };

  useEffect(() => {
    setStateIcon(<Icon status="danger"><ExclamationCircleIcon /></Icon>);
    if (isConnected) setStateIcon(<Icon status="success"><CheckCircleIcon /></Icon>);
  }, [isConnected]);

  return (
    <Tooltip
      content={
        <div>
          Connection is {readyStateString} <br />[last Message on {formatTimestamp(lastMessage.ts)}]
        </div>
      }
    >
      <Button variant={ButtonVariant.plain}>
        {stateIcon}
      </Button>
    </Tooltip>
  );
};

export default ConnectionState;
