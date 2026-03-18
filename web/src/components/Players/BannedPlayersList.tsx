import React, { useEffect } from 'react';
import { PageSection, Card, CardBody } from '@patternfly/react-core';
import PageHeader from '../UI/PageHeader';
import { usePlayerStore } from '../../store/playerStore';

const BannedPlayersList: React.FC = () => {
  const { getBannedPlayers, fetchPlayers } = usePlayerStore();
  const bannedPlayers = getBannedPlayers();

  useEffect(() => {
    fetchPlayers();
  }, [fetchPlayers]);

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
