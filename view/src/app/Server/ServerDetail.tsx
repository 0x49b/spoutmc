import * as React from 'react';
import {useEffect, useState} from 'react';
import {PageSection, Skeleton, Title} from '@patternfly/react-core';
import {Link} from "react-router-dom";
import {ArrowLeftIcon} from "@patternfly/react-icons";
import {useParams} from "react-router";
import {Server} from "@app/model/server";
import {useSelector} from "react-redux";
import {RootState} from "@app/store/store";

const ServerDetail: React.FunctionComponent = () => {
  // Navigation
  const {serverId} = useParams<{ serverId: string }>();

  // Server state with an initial value of undefined
  const servers = useSelector((state: RootState) => state.server.servers)
  const [server, setServer] = useState<Server | null>(null)

  useEffect(() => {
    const foundServer = servers.find(storeServer => storeServer.Id === serverId);
    setServer(foundServer || null);
  }, [serverId, servers]);

  return (
    <PageSection hasBodyWrapper={false}>
      <Title headingLevel="h1" size="lg">
        <Link to="/">
          <ArrowLeftIcon/>
        </Link>
        {/* Check if server is undefined or loading */}
        {server ? `${server.Config.Hostname}` :
          <Skeleton width="25%" screenreaderText="Loading contents"/>}
      </Title>
    </PageSection>
  );
};

export {ServerDetail};
