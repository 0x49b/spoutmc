import React, { useEffect, useState } from 'react';
import {
  PageSection,
  Card,
  CardBody,
  Button,
  EmptyState,
  EmptyStateBody,
  EmptyStateVariant,
  Modal,
  ModalVariant,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Alert
} from '@patternfly/react-core';
import { ActionsColumn, IAction, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { TrashIcon } from '@patternfly/react-icons';
import PageHeader from '../../UI/PageHeader';
import { useAuthStore } from '../../../store/authStore';
import AddUserModal from './AddUserModal';
import EditUserModal from './EditUserModal';
import type { UserProfile } from '../../../types';

const UsersList: React.FC = () => {
  const { users, fetchUsers, fetchRoles, deleteUser, hasPermission, roles } = useAuthStore();
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [editUser, setEditUser] = useState<{
    id: string;
    email: string;
    displayName: string;
    minecraftName?: string;
    roleIds: number[];
  } | null>(null);
  const [deleteUserTarget, setDeleteUserTarget] = useState<UserProfile | null>(null);
  const [deleteError, setDeleteError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchUsers();
    if (roles.length === 0) fetchRoles();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps -- fetch once on mount

  const canManage = hasPermission('manage', 'users') || hasPermission('manage', 'all');

  const handleDelete = async () => {
    if (!deleteUserTarget) return;
    setLoading(true);
    setDeleteError('');
    try {
      await deleteUser(deleteUserTarget.id);
      setDeleteUserTarget(null);
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } };
      setDeleteError(err?.response?.data?.error || (e as Error)?.message || 'Failed to delete user');
    } finally {
      setLoading(false);
    }
  };

  const getActions = (user: UserProfile): IAction[] => [
    {
      title: 'Edit',
      onClick: () => {
        setEditUser({
          id: user.id,
          email: user.email,
          displayName: user.displayName || '',
          minecraftName: user.minecraftName,
          roleIds: []
        });
      }
    },
    {
      title: 'Delete',
      onClick: () => {
        setDeleteUserTarget(user);
        setDeleteError('');
      }
    }
  ];

  return (
    <>
      <PageHeader
        title="Users"
        description="Manage system users"
        actions={
          canManage && (
            <Button variant="primary" onClick={() => setAddModalOpen(true)}>
              Add User
            </Button>
          )
        }
      />
      <PageSection>
        <Card>
          <CardBody>
            <Table aria-label="Users table" variant="compact">
              <Thead>
                <Tr>
                  <Th>Name</Th>
                  <Th>Email</Th>
                  <Th>Roles</Th>
                  {canManage && <Th />}
                </Tr>
              </Thead>
              <Tbody>
                {users.map((user) => (
                  <Tr key={user.id}>
                    <Td dataLabel="Name">
                      <strong>{user.displayName || user.email}</strong>
                      {user.minecraftName && (
                        <span style={{ color: 'var(--pf-v6-global--Color--200)', marginLeft: 8 }}>
                          (@{user.minecraftName})
                        </span>
                      )}
                    </Td>
                    <Td dataLabel="Email">{user.email}</Td>
                    <Td dataLabel="Roles">
                      {user.roles
                        .map((r) => roles.find((role) => role.name === r)?.displayName || r)
                        .join(', ')}
                    </Td>
                    {canManage && (
                      <Td isActionCell>
                        <ActionsColumn items={getActions(user)} />
                      </Td>
                    )}
                  </Tr>
                ))}
              </Tbody>
            </Table>
            {users.length === 0 && (
              <EmptyState variant={EmptyStateVariant.sm} titleText="No users yet">
                <EmptyStateBody>Add one to get started.</EmptyStateBody>
              </EmptyState>
            )}
          </CardBody>
        </Card>
      </PageSection>

      <AddUserModal
        isOpen={addModalOpen}
        onClose={() => setAddModalOpen(false)}
        onSuccess={() => setAddModalOpen(false)}
      />
      {editUser && (
        <EditUserModal
          isOpen={!!editUser}
          userId={editUser.id}
          initialEmail={editUser.email}
          initialDisplayName={editUser.displayName}
          initialMinecraftName={editUser.minecraftName}
          onClose={() => setEditUser(null)}
          onSuccess={() => setEditUser(null)}
        />
      )}

      {deleteUserTarget && (
        <Modal
          variant={ModalVariant.small}
          isOpen={!!deleteUserTarget}
          onClose={() => {
            setDeleteUserTarget(null);
            setDeleteError('');
          }}
        >
          <ModalHeader title="Delete User" titleIconVariant="warning" />
          <ModalBody>
            {deleteError && (
              <Alert variant="danger" title={deleteError} className="pf-v6-u-mb-md" />
            )}
            <p>
              Are you sure you want to delete{' '}
              <strong>{deleteUserTarget.displayName || deleteUserTarget.email}</strong>?
            </p>
          </ModalBody>
          <ModalFooter>
            <Button
              variant="danger"
              onClick={handleDelete}
              isDisabled={loading}
              icon={<TrashIcon />}
            >
              {loading ? 'Deleting...' : 'Delete'}
            </Button>
            <Button
              variant="link"
              onClick={() => {
                setDeleteUserTarget(null);
                setDeleteError('');
              }}
            >
              Cancel
            </Button>
          </ModalFooter>
        </Modal>
      )}
    </>
  );
};

export default UsersList;
