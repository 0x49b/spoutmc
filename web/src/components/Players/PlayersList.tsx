import React from 'react';
import { PageSection, Card, CardBody } from '@patternfly/react-core';
import PageHeader from '../UI/PageHeader';
import { usePlayerStore } from '../../store/playerStore';

const PlayersList: React.FC = () => {
  const { players } = usePlayerStore();

  return (
    <>
      <PageHeader title="Players" description="Manage players" />
      <PageSection>
        <Card>
          <CardBody>
            <p>Total Players: {players.length}</p>
          </CardBody>
        </Card>
      </PageSection>
    </>
  );
};

export default PlayersList;
