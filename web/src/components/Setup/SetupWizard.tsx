import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  PageSection,
  Card,
  CardBody,
  Button,
  Form,
  FormGroup,
  TextInput,
  Title,
  Checkbox,
  Alert
} from '@patternfly/react-core';
import { completeSetup } from '../../service/apiService';

const SetupWizard: React.FC = () => {
  const navigate = useNavigate();
  const [dataPath, setDataPath] = useState('./data');
  const [acceptEula, setAcceptEula] = useState(false);
  const [adminEmail, setAdminEmail] = useState('admin@example.com');
  const [adminPassword, setAdminPassword] = useState('');
  const [adminDisplayName, setAdminDisplayName] = useState('Admin');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleComplete = async () => {
    setError('');
    if (!acceptEula) {
      setError('You must accept the EULA to continue');
      return;
    }
    if (adminEmail && adminPassword && adminPassword.length < 6) {
      setError('Admin password must be at least 6 characters');
      return;
    }
    setLoading(true);
    try {
      await completeSetup({
        dataPath,
        acceptEula,
        adminEmail: adminEmail || undefined,
        adminPassword: adminPassword || undefined,
        adminDisplayName: adminDisplayName || undefined
      });
      localStorage.setItem('setupCompleted', 'true');
      navigate('/login');
    } catch (e: unknown) {
      setError((e as Error)?.message || 'Setup failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <PageSection>
      <div style={{ maxWidth: '600px', margin: '0 auto', marginTop: '4rem' }}>
        <Card>
          <CardBody>
            <Title headingLevel="h1" size="2xl" className="pf-v6-u-mb-md">
              Welcome to SpoutMC
            </Title>
            <p className="pf-v6-u-mb-lg">Complete the setup to get started</p>

            {error && (
              <Alert variant="danger" title={error} className="pf-v6-u-mb-md" />
            )}

            <Form>
              <FormGroup label="Data Path" isRequired fieldId="data-path">
                <TextInput
                  id="data-path"
                  value={dataPath}
                  onChange={(_event, value) => setDataPath(value)}
                  isRequired
                />
              </FormGroup>
              <FormGroup fieldId="eula">
                <Checkbox
                  id="eula"
                  label="I accept the Minecraft EULA"
                  isChecked={acceptEula}
                  onChange={(_event, checked) => setAcceptEula(checked)}
                />
              </FormGroup>
              <Title headingLevel="h2" size="lg" className="pf-v6-u-mt-lg pf-v6-u-mb-md">
                Initial Admin Account
              </Title>
              <FormGroup label="Admin Email" fieldId="admin-email">
                <TextInput
                  id="admin-email"
                  type="email"
                  value={adminEmail}
                  onChange={(_event, value) => setAdminEmail(value)}
                />
              </FormGroup>
              <FormGroup label="Admin Password" fieldId="admin-password">
                <TextInput
                  id="admin-password"
                  type="password"
                  value={adminPassword}
                  onChange={(_event, value) => setAdminPassword(value)}
                  placeholder="Min 6 characters"
                />
              </FormGroup>
              <FormGroup label="Admin Display Name" fieldId="admin-displayname">
                <TextInput
                  id="admin-displayname"
                  value={adminDisplayName}
                  onChange={(_event, value) => setAdminDisplayName(value)}
                />
              </FormGroup>
              <Button
                variant="primary"
                onClick={handleComplete}
                className="pf-v6-u-mt-md"
                isDisabled={loading || !acceptEula}
              >
                {loading ? 'Completing...' : 'Complete Setup'}
              </Button>
            </Form>
          </CardBody>
        </Card>
      </div>
    </PageSection>
  );
};

export default SetupWizard;
