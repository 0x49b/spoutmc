import { Label } from '@patternfly/react-core';
import { CheckCircleIcon, BanIcon, SyncAltIcon } from '@patternfly/react-icons';

type StatusType = 'online' | 'offline' | 'banned' | 'restarting' | 'enabled' | 'disabled';

interface StatusBadgeProps {
  status: StatusType;
  permanent?: boolean;
  className?: string;
}

const StatusBadge = ({ status, permanent, className = '' }: StatusBadgeProps) => {
  let color: 'green' | 'grey' | 'red' | 'orange' | 'blue' = 'grey';
  let icon: React.ReactNode = null;

  switch (status) {
    case 'online':
    case 'enabled':
      color = 'green';
      icon = <CheckCircleIcon />;
      break;
    case 'offline':
    case 'disabled':
      color = 'grey';
      break;
    case 'banned':
      color = 'red';
      icon = <BanIcon />;
      break;
    case 'restarting':
      color = 'orange';
      icon = <SyncAltIcon className="pf-v6-u-animation-spin" />;
      break;
    default:
      color = 'grey';
  }

  return (
    <Label
      color={color}
      icon={icon}
      className={className}
      isCompact
    >
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </Label>
  );
};

export default StatusBadge;
export { StatusBadge };
