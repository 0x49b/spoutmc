import React from 'react';
import { PageSection, Card, CardBody, Button } from '@patternfly/react-core';
import { PlusIcon } from '@patternfly/react-icons';
import PageHeader from '../../UI/PageHeader';
import { useAuthStore } from '../../../store/authStore';

const UsersList: React.FC = () => {
  const { users } = useAuthStore();

  return (
    <>
      <PageHeader
        title="Users"
        description="Manage system users"
        actions={<Button variant="primary" icon={<PlusIcon />}>Add User</Button>}
      />
      <PageSection>
        <Card>
          <CardBody>
            <p>Users: {users.length}</p>
            {users.map(user => (
              <div key={user.id}>{user.displayName || user.email}</div>
            ))}
          </CardBody>
        </Card>
      </PageSection>
    </>
  );
};

export default UsersList;
