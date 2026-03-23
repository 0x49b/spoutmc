import React, { useState, useEffect } from 'react';
import {
  Modal,
  ModalVariant,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Button,
  Form,
  FormGroup,
  TextInput,
  Checkbox,
  Alert
} from '@patternfly/react-core';
import { useAuthStore } from '../../../store/authStore';
import { getPermissions, getUser } from '../../../service/apiService';
import { RolePermissionsDualList } from '../Roles/RolePermissionsDualList';

interface EditUserModalProps {
  isOpen: boolean;
  userId: string;
  initialEmail: string;
  initialDisplayName: string;
  initialMinecraftName?: string;
  onClose: () => void;
  onSuccess: () => void;
}

const EditUserModal: React.FC<EditUserModalProps> = ({
  isOpen,
  userId,
  initialEmail,
  initialDisplayName,
  initialMinecraftName,
  onClose,
  onSuccess
}) => {
  const { updateUser, fetchRoles, roles } = useAuthStore();
  const [email, setEmail] = useState(initialEmail);
  const [displayName, setDisplayName] = useState(initialDisplayName);
  const [minecraftName, setMinecraftName] = useState(initialMinecraftName || '');
  const [selectedRoleIds, setSelectedRoleIds] = useState<number[]>([]);
  const [directPermissionIds, setDirectPermissionIds] = useState<number[]>([]);
  const [allPermissions, setAllPermissions] = useState<{ id: number; key: string; description?: string }[]>([]);
  const [permError, setPermError] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isOpen) {
      setEmail(initialEmail);
      setDisplayName(initialDisplayName);
      setMinecraftName(initialMinecraftName || '');
      setPermError('');
      if (roles.length === 0) fetchRoles();
      Promise.all([getUser(userId), getPermissions()])
        .then(([{ data: u }, { data: perms }]) => {
          setSelectedRoleIds(u.roles.map((r) => r.id));
          setDirectPermissionIds(u.directPermissions?.map((p) => p.id) ?? []);
          setAllPermissions(perms);
        })
        .catch(() => {
          setPermError('Failed to load user or permissions');
        });
    }
  }, [isOpen, userId, initialEmail, initialDisplayName, initialMinecraftName]); // eslint-disable-line react-hooks/exhaustive-deps -- fetchRoles when needed, roles from store

  const handleRoleToggle = (roleId: number, checked: boolean) => {
    setSelectedRoleIds((prev) =>
      checked ? [...prev, roleId] : prev.filter((id) => id !== roleId)
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (!email || !displayName) {
      setError('Email and display name are required');
      return;
    }
    setLoading(true);
    try {
      await updateUser(userId, {
        email,
        displayName,
        minecraftName: minecraftName || undefined,
        roleIds: selectedRoleIds,
        permissionIds: directPermissionIds
      });
      onSuccess();
      onClose();
    } catch {
      setError('Failed to update user');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal variant={ModalVariant.large} isOpen={isOpen} onClose={onClose}>
      <ModalHeader title="Edit User" />
      <ModalBody>
        {error && (
          <Alert variant="danger" title={error} className="pf-v6-u-mb-md" />
        )}
        {permError && (
          <Alert variant="danger" title={permError} className="pf-v6-u-mb-md" />
        )}
        <Form id="edit-user-form" onSubmit={handleSubmit}>
          <FormGroup label="Email" isRequired fieldId="email">
            <TextInput
              id="email"
              type="email"
              value={email}
              onChange={(_event, value) => setEmail(value)}
              isRequired
            />
          </FormGroup>
          <FormGroup label="Display Name" isRequired fieldId="displayName">
            <TextInput
              id="displayName"
              value={displayName}
              onChange={(_event, value) => setDisplayName(value)}
              isRequired
            />
          </FormGroup>
          <FormGroup label="Minecraft Name" fieldId="minecraftName">
            <TextInput
              id="minecraftName"
              value={minecraftName}
              onChange={(_event, value) => setMinecraftName(value)}
            />
          </FormGroup>
          <FormGroup label="Roles" fieldId="roles">
            {roles.map((role) => (
              <Checkbox
                key={role.id}
                id={`role-${role.id}`}
                label={role.displayName || role.name}
                isChecked={selectedRoleIds.includes(role.id)}
                onChange={(_event, checked) => handleRoleToggle(role.id, checked)}
                className="pf-v6-u-mb-sm"
              />
            ))}
          </FormGroup>
          {allPermissions.length > 0 && (
            <FormGroup label="Direct permissions" fieldId="user-direct-perms">
              <p className="pf-v6-u-mb-md" style={{ color: 'var(--pf-v6-global--Color--200)', fontSize: '0.875rem' }}>
                Optional extra grants in addition to role permissions. Effective access is the union of both.
              </p>
              <RolePermissionsDualList
                id="user-direct-permissions"
                allPermissions={allPermissions}
                chosenIds={directPermissionIds}
                onChosenIdsChange={setDirectPermissionIds}
                isDisabled={loading || !!permError}
                availableTitle="Available permissions"
                chosenTitle="Granted directly to this user"
              />
            </FormGroup>
          )}
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button
          variant="primary"
          type="submit"
          form="edit-user-form"
          isDisabled={loading || !!permError}
        >
          {loading ? 'Saving...' : 'Save'}
        </Button>
        <Button variant="link" onClick={onClose}>
          Cancel
        </Button>
      </ModalFooter>
    </Modal>
  );
};

export default EditUserModal;
