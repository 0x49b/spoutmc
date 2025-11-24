import React from 'react';
import { PageSection, Card, CardBody, Title } from '@patternfly/react-core';
import PageHeader from '../../UI/PageHeader';
import { useAuthStore } from '../../../store/authStore';

const UserProfile: React.FC = () => {
  const { user } = useAuthStore();

  return (
    <>
      <PageHeader title="Your Profile" description="View and edit your profile" />
      <PageSection>
        <Card>
          <CardBody>
            <Title headingLevel="h3" size="lg">Profile Information</Title>
            <p>Email: {user?.email}</p>
            <p>Display Name: {user?.displayName}</p>
            <p>Roles: {user?.roles.join(', ')}</p>
          </CardBody>
        </Card>
      </PageSection>
    </>
  );
};

export default UserProfile;
