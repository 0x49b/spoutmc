import * as React from "react";
import {ReactElement, useEffect, useState} from "react";
import {useSelector} from "react-redux";
import {RootState} from "@app/store/store";
import {Button, ButtonVariant, Icon, Tooltip} from "@patternfly/react-core";
import {
  CheckCircleIcon,
  ExclamationCircleIcon,
  ExclamationTriangleIcon
} from "@patternfly/react-icons";
import {ReadyState} from "react-use-websocket";

const ConnectionState: React.FunctionComponent = () => {

  const lastMessage = useSelector((state: RootState) => state.message.lastMessage)
  const readyStateString = useSelector((state: RootState) => state.socket.readyStateString)
  const readyState = useSelector((state: RootState) => state.socket.readyState)

  const [stateIcon, setStateIcon] = useState<ReactElement>(<></>)

  const formatTimestamp = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleString("de-CH", {
      dateStyle: 'short', timeStyle: 'medium'
    });
  }

  useEffect(() => {

    switch (readyState) {
      case ReadyState.CLOSED:
        setStateIcon(<Icon status="danger"><ExclamationCircleIcon/></Icon>)
        break
      case ReadyState.OPEN:
        setStateIcon(<Icon status="success"><CheckCircleIcon/></Icon>)
        break
      default:
        setStateIcon(<Icon status="warning"><ExclamationTriangleIcon/></Icon>)
    }

  }, [readyState]);

  return (
    <Tooltip
      content={
        <div>
          Connection is {readyStateString} <br/>[last Message on {formatTimestamp(lastMessage.ts)}]
        </div>
      }
    >
      <Button variant={ButtonVariant.plain}>
        {stateIcon}
      </Button>
    </Tooltip>
  )
}

export default ConnectionState;
