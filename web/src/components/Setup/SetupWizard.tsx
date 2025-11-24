import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { PageSection, Card, CardBody, Button, Form, FormGroup, TextInput, Title } from '@patternfly/react-core';

const SetupWizard: React.FC = () => {
  const navigate = useNavigate();
  const [adminEmail, setAdminEmail] = useState('admin@example.com');
  const [adminPassword, setAdminPassword] = useState('password');

  const handleComplete = () => {
    localStorage.setItem('setupCompleted', 'true');
    navigate('/login');
  };

  return (
    <PageSection>
      <div style={{ maxWidth: '600px', margin: '0 auto', marginTop: '4rem' }}>
        <Card>
          <CardBody>
            <Title headingLevel="h1" size="2xl" className="pf-v6-u-mb-md">Welcome to SpoutMC</Title>
            <p className="pf-v6-u-mb-lg">Complete the setup to get started</p>
            <Form>
              <FormGroup label="Admin Email" isRequired fieldId="admin-email">
                <TextInput id="admin-email" value={adminEmail} onChange={(_event, value) => setAdminEmail(value)} isRequired />
              </FormGroup>
              <FormGroup label="Admin Password" isRequired fieldId="admin-password">
                <TextInput id="admin-password" type="password" value={adminPassword} onChange={(_event, value) => setAdminPassword(value)} isRequired />
              </FormGroup>
              <Button variant="primary" onClick={handleComplete} className="pf-v6-u-mt-md">Complete Setup</Button>
            </Form>
          </CardBody>
        </Card>
      </div>
    </PageSection>
  );
};

export default SetupWizard;
