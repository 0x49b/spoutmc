import React from 'react';
import { PageSection, Card, CardBody, Button } from '@patternfly/react-core';
import { PlusIcon } from '@patternfly/react-icons';
import PageHeader from '../UI/PageHeader';
import { usePluginStore } from '../../store/pluginStore';

const PluginsList: React.FC = () => {
  const { plugins } = usePluginStore();

  return (
    <>
      <PageHeader
        title="Plugins"
        description="Manage server plugins"
        actions={<Button variant="primary" icon={<PlusIcon />}>Add Plugin</Button>}
      />
      <PageSection>
        <Card>
          <CardBody>
            <p>Total Plugins: {plugins.length}</p>
          </CardBody>
        </Card>
      </PageSection>
    </>
  );
};

export default PluginsList;
