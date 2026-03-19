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
  FormHelperText,
  Alert
} from '@patternfly/react-core';
import { useAuthStore } from '../../../store/authStore';

interface AddUserModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

const AddUserModal: React.FC<AddUserModalProps> = ({ isOpen, onClose, onSuccess }) => {
  const { addUser, fetchRoles, roles } = useAuthStore();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [minecraftName, setMinecraftName] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isOpen && roles.length === 0) {
      fetchRoles();
    }
  }, [isOpen]); // eslint-disable-line react-hooks/exhaustive-deps -- only fetch when modal opens and roles not yet loaded

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (!email || !password || !displayName) {
      setError('Email, password, and display name are required');
      return;
    }
    if (password.length < 6) {
      setError('Password must be at least 6 characters');
      return;
    }
    setLoading(true);
    try {
      await addUser({
        email,
        password,
        displayName,
        minecraftName: minecraftName || undefined
      });
      setEmail('');
      setPassword('');
      setDisplayName('');
      setMinecraftName('');
      onSuccess();
      onClose();
    } catch {
      setError('Failed to create user');
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setError('');
    setEmail('');
    setPassword('');
    setDisplayName('');
    setMinecraftName('');
    onClose();
  };

  return (
    <Modal variant={ModalVariant.medium} isOpen={isOpen} onClose={handleClose}>
      <ModalHeader title="Add User" />
      <ModalBody>
        {error && (
          <Alert variant="danger" title={error} className="pf-v6-u-mb-md" />
        )}
        <Form id="add-user-form" onSubmit={handleSubmit}>
          <FormGroup label="Email" isRequired fieldId="email">
            <TextInput
              id="email"
              type="email"
              value={email}
              onChange={(_event, value) => setEmail(value)}
              isRequired
            />
          </FormGroup>
          <FormGroup label="Password" isRequired fieldId="password">
            <TextInput
              id="password"
              type="password"
              value={password}
              onChange={(_event, value) => setPassword(value)}
              isRequired
            />
            <FormHelperText>Minimum 6 characters</FormHelperText>
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
              placeholder="In-game username (optional)"
            />
          </FormGroup>
        </Form>
      </ModalBody>
      <ModalFooter>
        <Button
          variant="primary"
          type="submit"
          form="add-user-form"
          isDisabled={loading}
        >
          {loading ? 'Adding...' : 'Add'}
        </Button>
        <Button variant="link" onClick={handleClose}>
          Cancel
        </Button>
      </ModalFooter>
    </Modal>
  );
};

export default AddUserModal;
