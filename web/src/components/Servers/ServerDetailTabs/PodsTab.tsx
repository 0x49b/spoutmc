import React from 'react';
import { Card, CardBody, Alert } from '@patternfly/react-core';

const PodsTab: React.FC = () => {
  return (
    <Card>
      <CardBody>
        <Alert variant="info" isInline title="Deprecated Feature">
          This feature is for Kubernetes and is not applicable to Docker-only deployments.
        </Alert>
      </CardBody>
    </Card>
  );
};

export default PodsTab;
