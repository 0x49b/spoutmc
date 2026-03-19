import React from 'react';
import { Link } from 'react-router-dom';
import { PageSection, Card, CardBody, Button } from '@patternfly/react-core';
import { UsersIcon, KeyIcon } from '@patternfly/react-icons';
import PageHeader from '../UI/PageHeader';

const Configuration: React.FC = () => {
  return (
    <>
      <PageHeader title="Configuration" description="System settings and configuration" />
      <PageSection>
        <Card>
          <CardBody>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
              <Link to="/users">
                <Button variant="link" icon={<UsersIcon />}>
                  User Management
                </Button>
              </Link>
              <Link to="/configuration/roles">
                <Button variant="link" icon={<KeyIcon />}>
                  Role Management
                </Button>
              </Link>
            </div>
          </CardBody>
        </Card>
      </PageSection>
    </>
  );
};

export default Configuration;
