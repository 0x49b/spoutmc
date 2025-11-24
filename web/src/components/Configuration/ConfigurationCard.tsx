import React from 'react';
import { Card, CardTitle, CardBody } from '@patternfly/react-core';

interface ConfigurationCardProps {
  title: string;
  children: React.ReactNode;
}

const ConfigurationCard: React.FC<ConfigurationCardProps> = ({ title, children }) => {
  return (
    <Card>
      <CardTitle>{title}</CardTitle>
      <CardBody>{children}</CardBody>
    </Card>
  );
};

export default ConfigurationCard;
