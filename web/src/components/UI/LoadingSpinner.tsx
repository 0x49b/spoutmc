import React from 'react';
import {Spinner} from '@patternfly/react-core';

const LoadingSpinner: React.FC = () => {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '16rem' }}>
      <Spinner size="xl" aria-label="Loading content" />
    </div>
  );
};

export default LoadingSpinner;
