import * as React from 'react';
import {useEffect} from 'react';
import {Flex, PageSection, Skeleton, Tab, Tabs, TabTitleText, Title} from '@patternfly/react-core';
import {Link} from "react-router-dom";
import {ArrowLeftIcon} from "@patternfly/react-icons";
import {useParams} from "react-router";
import {useSelector} from "react-redux";
import {RootState} from "@app/store/store";
import {WsCommand, WsCommandType} from "@app/model/wsCommand";
import {useServerWebSocket} from "../../../services/websocketService";
import {ServerDetailsInspect} from "@app/Server/details/ServerDetailsInspect";
import {ServerDetailsStats} from "@app/Server/details/ServerDetailsStats";
import {ServerDetailsLogs} from "@app/Server/details/ServerDetailsLogs";

enum ActiveTab {
  LOGS = 0,
  INSPECT = 1,
  STATS = 2
}

const ServerDetail: React.FunctionComponent = () => {
  // const
  const {sendMessage} = useServerWebSocket();
  const {serverId} = useParams<{ serverId: string }>();
  const server = useSelector((state: RootState) => state.server.server)
  const [activeTabKey, setActiveTabKey] = React.useState<string | number>(0);

  //Effects
  useEffect(() => {
    loadServerDetail()
  }, [serverId]);

  // Functions
  const loadServerDetail = () => {
    const commandMessage: WsCommand = {
      type: WsCommandType.CONTAINERDETAIL,
      containerId: serverId,
    }
    sendMessage(JSON.stringify(commandMessage))
  }

  const handleTabClick = (
    event: React.MouseEvent<any> | React.KeyboardEvent | MouseEvent,
    tabIndex: string | number
  ) => {
    setActiveTabKey(tabIndex);
  };


  return (
    <PageSection hasBodyWrapper={false}>
      <Title headingLevel="h1" size="lg">
        <Flex>
          <Link to="/server">
            <ArrowLeftIcon/>
          </Link>
          {/* Check if server is undefined or loading */}
          {server ? <span>{server.Config.Hostname}</span> :
            <Skeleton width="25%" screenreaderText="Loading contents"/>}
        </Flex>
      </Title>

      <Tabs
        activeKey={activeTabKey}
        onSelect={handleTabClick}
        aria-label="Server details Tabs"
      >
        <Tab eventKey={ActiveTab.LOGS} title={<TabTitleText>Logs</TabTitleText>}
             aria-label="Default content - users">
          <ServerDetailsLogs/>
        </Tab>
        <Tab eventKey={ActiveTab.INSPECT} title={<TabTitleText>Inspect</TabTitleText>}>
          <ServerDetailsInspect inspectJson={JSON.stringify(server, null, 2)}/>
        </Tab>
        <Tab eventKey={ActiveTab.STATS} title={<TabTitleText>Stats</TabTitleText>}>
          <ServerDetailsStats/>
        </Tab>
      </Tabs>

    </PageSection>
  );
};

export {ServerDetail};
