import React, {useEffect, useState} from 'react';
import {
    Avatar,
    Button,
    Card,
    CardBody,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    PageSection,
    TextInput,
    Title
} from '@patternfly/react-core';
import PageHeader from '../../UI/PageHeader';
import {getUserAvatarDataUrl, useAuthStore} from '../../../store/authStore';
import {useNotificationStore} from '../../../store/notificationStore';

const UserProfile: React.FC = () => {
  const { user, updateProfile, clearError } = useAuthStore();
  const pushToast = useNotificationStore((s) => s.pushToast);
  const [email, setEmail] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [minecraftName, setMinecraftName] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
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

    if (password && password !== confirmPassword) {
      pushToast({ variant: 'danger', title: 'Passwords do not match' });
      return;
    }

    if (password && password.length < 6) {
      pushToast({ variant: 'danger', title: 'Password must be at least 6 characters' });
      return;
    }

    setSaving(true);
    try {
      await updateProfile({
        email: email || undefined,
        displayName: displayName || undefined,
        minecraftName,
        password: password || undefined
      });
      pushToast({ variant: 'success', title: 'Profile updated successfully' });
      setPassword('');
      setConfirmPassword('');
    } catch {
      const err = useAuthStore.getState().error;
      if (err) {
        pushToast({ variant: 'danger', title: err });
        clearError();
      }
    } finally {
      setSaving(false);
    }
  };

  if (!user) {
    return null;
  }

  const profileAvatarSrc = getUserAvatarDataUrl(user);

  return (
    <>
      <PageHeader title="Your Profile" description="View and edit your profile" />
      <PageSection>
        <Card>
          <CardBody>
            <Flex alignItems={{ default: 'alignItemsCenter' }} gap={{ default: 'gapMd' }} className="pf-v6-u-mb-md">
              <FlexItem>
                <Avatar src={profileAvatarSrc} alt="" size="lg" />
              </FlexItem>
              <FlexItem>
                <Title headingLevel="h3" size="lg">
                  Profile Information
                </Title>
                <p className="pf-v6-u-mt-sm" style={{ color: 'var(--pf-v6-global--Color--200)' }}>
                  Roles: {user.roles.join(', ')}
                </p>
                <p className="pf-v6-u-mt-xs" style={{ color: 'var(--pf-v6-global--Color--200)', fontSize: 'var(--pf-v6-global--FontSize--sm)' }}>
                  Avatar is generated from your Minecraft skin when you save a Minecraft name.
                </p>
              </FlexItem>
            </Flex>

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
