import React from 'react';
import { PageSection, Card, CardBody } from '@patternfly/react-core';
import PageHeader from '../UI/PageHeader';
import { usePlayerStore } from '../../store/playerStore';

const BannedPlayersList: React.FC = () => {
  const { getBannedPlayers } = usePlayerStore();
  const bannedPlayers = getBannedPlayers();

  return (
    <>
      <PageHeader title="Banned Players" description="View banned players" />
      <PageSection>
        <Card>
          <CardBody>
            <p>Banned Players: {bannedPlayers.length}</p>
          </CardBody>
        </Card>
      </PageSection>
    </>
  );
};

export default BannedPlayersList;
