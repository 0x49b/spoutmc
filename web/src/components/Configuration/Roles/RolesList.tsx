import React, {useEffect, useState} from 'react';
import {
    Alert,
    Button,
    Card,
    CardBody,
    EmptyState,
    EmptyStateBody,
    EmptyStateVariant,
    Form,
    FormGroup,
    Modal,
    ModalBody,
    ModalFooter,
    ModalHeader,
    ModalVariant,
    PageSection,
    TextInput
} from '@patternfly/react-core';
import {ActionsColumn, IAction, Table, Tbody, Td, Th, Thead, Tr} from '@patternfly/react-table';
import {TrashIcon} from '@patternfly/react-icons';
import PageHeader from '../../UI/PageHeader';
import {useAuthStore} from '../../../store/authStore';
import * as api from '../../../service/apiService';
import {RolePermissionsDualList} from './RolePermissionsDualList';

const RolesList: React.FC = () => {
  const { roles, fetchRoles, hasRole } = useAuthStore();
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [editRole, setEditRole] = useState<api.RoleDTO | null>(null);
  const [deleteRoleTarget, setDeleteRoleTarget] = useState<api.RoleDTO | null>(null);
  const [newRoleDisplayName, setNewRoleDisplayName] = useState('');
  const [editRoleDisplayName, setEditRoleDisplayName] = useState('');
  const [editRolePermissionIds, setEditRolePermissionIds] = useState<number[]>([]);
  const [allPermissions, setAllPermissions] = useState<api.PermissionDTO[]>([]);
  const [permLoadError, setPermLoadError] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [deleteError, setDeleteError] = useState('');

  useEffect(() => {
    fetchRoles();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps -- fetchRoles is stable, fetch once on mount

  const canManage = hasRole('admin');

  const openEditRole = async (role: api.RoleDTO) => {
    setEditRole(role);
    setEditRoleDisplayName(displayName(role));
    setEditRolePermissionIds([]);
    setAllPermissions([]);
    setPermLoadError('');
    setError('');
    try {
      const [roleRes, permRes] = await Promise.all([
        api.getRole(String(role.id)),
        api.getPermissions()
      ]);
      setAllPermissions(permRes.data);
      setEditRolePermissionIds(roleRes.data.permissions?.map((p) => p.id) ?? []);
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } };
      setPermLoadError(err?.response?.data?.error || (e as Error)?.message || 'Failed to load role permissions');
    }
  };

  const handleAddRole = async () => {
    if (!newRoleDisplayName.trim()) return;
    setLoading(true);
    setError('');
    try {
      await api.createRole(newRoleDisplayName.trim());
      await fetchRoles();
      setNewRoleDisplayName('');
      setAddModalOpen(false);
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } };
      setError(err?.response?.data?.error || (e as Error)?.message || 'Failed to create role');
    } finally {
      setLoading(false);
    }
  };

  const handleUpdateRole = async () => {
    if (!editRole || !editRoleDisplayName.trim()) return;
    setLoading(true);
    setError('');
    try {
      await api.updateRole(String(editRole.id), {
        displayName: editRoleDisplayName.trim(),
        permissionIds: editRolePermissionIds
      });
      await fetchRoles();
      setEditRole(null);
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } };
      setError(err?.response?.data?.error || (e as Error)?.message || 'Failed to update role');
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteRole = async () => {
    if (!deleteRoleTarget) return;
    setLoading(true);
    setDeleteError('');
    try {
      await api.deleteRole(String(deleteRoleTarget.id));
      await fetchRoles();
      setDeleteRoleTarget(null);
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } };
      setDeleteError(err?.response?.data?.error || (e as Error)?.message || 'Failed to delete role');
    } finally {
      setLoading(false);
    }
  };

  const displayName = (role: api.RoleDTO) => role.displayName || role.name;

  const getActions = (role: api.RoleDTO): IAction[] => [
    {
      title: 'Edit',
      onClick: () => {
        void openEditRole(role);
      }
    },
    {
      title: 'Delete',
      onClick: () => {
        setDeleteRoleTarget(role);
        setDeleteError('');
      },
      isDisabled: role.userCount !== undefined && role.userCount > 0
    }
  ];

  return (
    <>
      <PageHeader
        title="Roles"
        description="Manage user roles and permissions"
        actions={
          canManage && (
            <Button variant="primary" onClick={() => setAddModalOpen(true)}>
              Add Role
            </Button>
          )
        }
      />
      <PageSection>
        <Card>
          <CardBody>
            <Table aria-label="Roles table" variant="compact">
              <Thead>
                <Tr>
                  <Th>Display Name</Th>
                  <Th>Name</Th>
                  <Th>Slug</Th>
                  <Th>Users</Th>
                  {canManage && <Th />}
                </Tr>
              </Thead>
              <Tbody>
                {roles.map((role) => (
                  <Tr key={role.id}>
                    <Td dataLabel="Display Name">
                      <strong>{displayName(role)}</strong>
                    </Td>
                    <Td dataLabel="Name">{role.name}</Td>
                    <Td dataLabel="Slug">{role.slug}</Td>
                    <Td dataLabel="Users">
                      {role.userCount !== undefined
                        ? `${role.userCount} user${role.userCount !== 1 ? 's' : ''}`
                        : '-'}
                    </Td>
                    {canManage && (
                      <Td isActionCell>
                        <ActionsColumn items={getActions(role)} />
                      </Td>
                    )}
                  </Tr>
                ))}
              </Tbody>
            </Table>
            {roles.length === 0 && (
              <EmptyState variant={EmptyStateVariant.sm} titleText="No roles">
                <EmptyStateBody>
                  Default roles (Admin, Manager, Editor, Mod, Support) are seeded on first run.
                </EmptyStateBody>
              </EmptyState>
            )}
          </CardBody>
        </Card>
      </PageSection>

      {/* Add Role Modal - PatternFly structure */}
      <Modal
        variant={ModalVariant.small}
        isOpen={addModalOpen}
        onClose={() => {
          setAddModalOpen(false);
          setNewRoleDisplayName('');
          setError('');
        }}
      >
        <ModalHeader title="Add Role" />
        <ModalBody>
          {error && (
            <Alert variant="danger" title={error} className="pf-v6-u-mb-md" />
          )}
          <Form id="add-role-form" onSubmit={(e) => { e.preventDefault(); handleAddRole(); }}>
            <FormGroup label="Display Name" isRequired fieldId="newRoleDisplayName">
              <TextInput
                id="newRoleDisplayName"
                value={newRoleDisplayName}
                onChange={(_event, value) => setNewRoleDisplayName(value)}
                placeholder="e.g. Forum Moderator"
              />
              <p className="pf-v6-u-mt-sm" style={{ color: 'var(--pf-v6-global--Color--200)', fontSize: '0.875rem' }}>
                Name and slug will be derived: &quot;Forum Moderator&quot; → name: forumModerator, slug: forum-moderator
              </p>
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button
            key="add"
            variant="primary"
            type="submit"
            form="add-role-form"
            isDisabled={loading || !newRoleDisplayName.trim()}
          >
            {loading ? 'Adding...' : 'Add'}
          </Button>
          <Button
            key="cancel"
            variant="link"
            onClick={() => {
              setAddModalOpen(false);
              setNewRoleDisplayName('');
              setError('');
            }}
          >
            Cancel
          </Button>
        </ModalFooter>
      </Modal>

      {/* Edit Role Modal - PatternFly structure + dual list */}
      {editRole && (
        <Modal
          variant={ModalVariant.large}
          isOpen={!!editRole}
          onClose={() => {
            setEditRole(null);
            setError('');
            setPermLoadError('');
          }}
        >
          <ModalHeader title="Edit Role" />
          <ModalBody>
            {error && (
              <Alert variant="danger" title={error} className="pf-v6-u-mb-md" />
            )}
            {permLoadError && (
              <Alert variant="danger" title={permLoadError} className="pf-v6-u-mb-md" />
            )}
            <Form id="edit-role-form" onSubmit={(e) => { e.preventDefault(); handleUpdateRole(); }}>
              <FormGroup label="Display Name" isRequired fieldId="editRoleDisplayName">
                <TextInput
                  id="editRoleDisplayName"
                  value={editRoleDisplayName}
                  onChange={(_event, value) => setEditRoleDisplayName(value)}
                />
              </FormGroup>
              {allPermissions.length > 0 && (
                <FormGroup label="Permissions" fieldId="role-permissions-dual">
                  <RolePermissionsDualList
                    id="role-edit-permissions"
                    allPermissions={allPermissions.map((p) => ({
                      id: p.id,
                      key: p.key,
                      description: p.description
                    }))}
                    chosenIds={editRolePermissionIds}
                    onChosenIdsChange={setEditRolePermissionIds}
                    isDisabled={loading}
                    availableTitle="Available permissions"
                    chosenTitle="Permissions for this role"
                  />
                </FormGroup>
              )}
            </Form>
          </ModalBody>
          <ModalFooter>
            <Button
              key="save"
              variant="primary"
              type="submit"
              form="edit-role-form"
              isDisabled={loading || !editRoleDisplayName.trim() || !!permLoadError}
            >
              {loading ? 'Saving...' : 'Save'}
            </Button>
            <Button
              key="cancel"
              variant="link"
              onClick={() => {
                setEditRole(null);
                setError('');
                setPermLoadError('');
              }}
            >
              Cancel
            </Button>
          </ModalFooter>
        </Modal>
      )}

      {/* Delete Role Modal - PatternFly structure */}
      {deleteRoleTarget && (
        <Modal
          variant={ModalVariant.small}
          isOpen={!!deleteRoleTarget}
          onClose={() => {
            setDeleteRoleTarget(null);
            setDeleteError('');
          }}
        >
          <ModalHeader title="Delete Role" titleIconVariant="warning" />
          <ModalBody>
            {deleteError && (
              <Alert variant="danger" title={deleteError} className="pf-v6-u-mb-md" />
            )}
            {deleteRoleTarget.userCount !== undefined && deleteRoleTarget.userCount > 0 ? (
              <p>
                Cannot delete <strong>{displayName(deleteRoleTarget)}</strong>: this role is assigned to{' '}
                {deleteRoleTarget.userCount} user{deleteRoleTarget.userCount !== 1 ? 's' : ''}. Remove the role from all users first.
              </p>
            ) : (
              <p>
                Are you sure you want to delete the role <strong>{displayName(deleteRoleTarget)}</strong>?
              </p>
            )}
          </ModalBody>
          <ModalFooter>
            <Button
              key="delete"
              variant="danger"
              onClick={handleDeleteRole}
              isDisabled={loading || (deleteRoleTarget.userCount !== undefined && deleteRoleTarget.userCount > 0)}
              icon={<TrashIcon />}
            >
              {loading ? 'Deleting...' : 'Delete'}
            </Button>
            <Button
              key="cancel"
              variant="link"
              onClick={() => {
                setDeleteRoleTarget(null);
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

export default RolesList;
