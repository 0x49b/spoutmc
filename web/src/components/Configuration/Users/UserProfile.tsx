import React, { useState, useEffect } from 'react';
import {
  PageSection,
  Card,
  CardBody,
  Title,
  Form,
  FormGroup,
  TextInput,
  Button,
  Alert
} from '@patternfly/react-core';
import PageHeader from '../../UI/PageHeader';
import { useAuthStore } from '../../../store/authStore';

const UserProfile: React.FC = () => {
  const { user, updateProfile, error } = useAuthStore();
  const [email, setEmail] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [minecraftName, setMinecraftName] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [success, setSuccess] = useState('');
  const [validationError, setValidationError] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (user) {
      setEmail(user.email);
      setDisplayName(user.displayName);
      setMinecraftName(user.minecraftName || '');
    }
  }, [user]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setValidationError('');
    setSuccess('');

    if (password && password !== confirmPassword) {
      setValidationError('Passwords do not match');
      return;
    }

    if (password && password.length < 6) {
      setValidationError('Password must be at least 6 characters');
      return;
    }

    setSaving(true);
    try {
      await updateProfile({
        email: email || undefined,
        displayName: displayName || undefined,
        minecraftName: minecraftName || undefined,
        password: password || undefined
      });
      setSuccess('Profile updated successfully');
      setPassword('');
      setConfirmPassword('');
    } catch {
      // Error handled by store
    } finally {
      setSaving(false);
    }
  };

  if (!user) {
    return null;
  }

  return (
    <>
      <PageHeader title="Your Profile" description="View and edit your profile" />
      <PageSection>
        <Card>
          <CardBody>
            <Title headingLevel="h3" size="lg">
              Profile Information
            </Title>
            <p className="pf-v6-u-mt-sm pf-v6-u-mb-md" style={{ color: 'var(--pf-v6-global--Color--200)' }}>
              Roles: {user.roles.join(', ')}
            </p>

            {(error || validationError) && (
              <Alert
                variant="danger"
                title={error || validationError}
                className="pf-v6-u-mb-md"
                onClose={() => {
                  setValidationError('');
                }}
              />
            )}
            {success && (
              <Alert variant="success" title={success} className="pf-v6-u-mb-md" />
            )}

            <Form onSubmit={handleSubmit}>
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
                  placeholder="Your in-game Minecraft username"
                />
              </FormGroup>
              <FormGroup label="New Password" fieldId="password">
                <TextInput
                  id="password"
                  type="password"
                  value={password}
                  onChange={(_event, value) => setPassword(value)}
                  placeholder="Leave blank to keep current password"
                />
              </FormGroup>
              {password && (
                <FormGroup label="Confirm Password" fieldId="confirmPassword">
                  <TextInput
                    id="confirmPassword"
                    type="password"
                    value={confirmPassword}
                    onChange={(_event, value) => setConfirmPassword(value)}
                  />
                </FormGroup>
              )}
              <Button type="submit" variant="primary" isDisabled={saving}>
                {saving ? 'Saving...' : 'Save Changes'}
              </Button>
            </Form>
          </CardBody>
        </Card>
      </PageSection>
    </>
  );
};

export default UserProfile;
