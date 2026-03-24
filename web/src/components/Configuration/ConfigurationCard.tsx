import React from 'react';
import {Card, CardBody, CardTitle} from '@patternfly/react-core';

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
