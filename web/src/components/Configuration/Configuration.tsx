import React from 'react';
import { Link } from 'react-router-dom';
import { PageSection, Card, CardBody, Button } from '@patternfly/react-core';
import { UsersIcon } from '@patternfly/react-icons';
import PageHeader from '../UI/PageHeader';

const Configuration: React.FC = () => {
  return (
    <>
      <PageHeader title="Configuration" description="System settings and configuration" />
      <PageSection>
        <Card>
          <CardBody>
            <Link to="/users">
              <Button variant="link" icon={<UsersIcon />}>User Management</Button>
            </Link>
          </CardBody>
        </Card>
      </PageSection>
    </>
  );
};

export default Configuration;
