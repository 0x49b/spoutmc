import React from 'react';
import { Skeleton as PFSkeleton } from '@patternfly/react-core';

interface SkeletonProps {
  className?: string;
  variant?: 'text' | 'circular' | 'rectangular';
  width?: string | number;
  height?: string | number;
}

const Skeleton: React.FC<SkeletonProps> = ({
  className = '',
  variant = 'rectangular',
  width,
  height
}) => {
  // Map our variants to PatternFly shape types
  const shape = variant === 'circular' ? 'circle' : variant === 'text' ? 'square' : 'square';

  const style: React.CSSProperties = {
    width: width,
    height: height
  };

  return (
    <PFSkeleton
      shape={shape}
      width={width ? `${width}` : undefined}
      height={height ? `${height}` : undefined}
      className={className}
      style={style}
    />
  );
};

export default Skeleton;
