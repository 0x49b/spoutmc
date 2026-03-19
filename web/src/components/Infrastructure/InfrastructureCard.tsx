import React from 'react';
import { DatabaseIcon } from '@patternfly/react-icons';
import ContainerCard, { ContainerCardDetail } from '../Shared/ContainerCard';
import { InfrastructureContainer, getContainerId } from '../../types';

interface InfrastructureCardProps {
  container: InfrastructureContainer & { cpu?: number; memory?: number };
  onClick: (containerId: string) => void;
}

const getTypeLabel = (type: string) => {
  switch (type) {
    case 'database':
      return 'Database';
    default:
      return 'Unknown';
  }
};

const normalizeStatus = (state: string): 'online' | 'offline' | 'restarting' => {
  switch (state) {
    case 'running':
      return 'online';
    case 'restarting':
    case 'paused':
      return 'restarting';
    case 'exited':
    case 'dead':
    default:
      return 'offline';
  }
};

const InfrastructureCard: React.FC<InfrastructureCardProps> = ({ container, onClick }) => {
  const name = container.summary.Names?.[0]?.replace(/^\//, '') || 'Unnamed';
  const ports =
    container.summary.Ports && container.summary.Ports.length > 0
      ? container.summary.Ports.map((port: { PublicPort?: number; PrivatePort: number }) =>
          port.PublicPort ? `${port.PublicPort}:${port.PrivatePort}` : String(port.PrivatePort)
        ).join(', ')
      : '-';

  const details: ContainerCardDetail[] = [
    { label: 'Image', value: container.summary.Image },
    { label: 'Ports', value: ports }
  ];

  const containerId = getContainerId(container.summary);

  return (
    <ContainerCard
      id={containerId}
      name={name}
      status={normalizeStatus(container.summary.State)}
      icon={<DatabaseIcon />}
      typeLabel={getTypeLabel(container.type)}
      details={details}
      cpu={container.cpu ?? 0}
      memory={container.memory ?? 0}
      onClick={() => onClick(containerId)}
    />
  );
};

export default InfrastructureCard;
