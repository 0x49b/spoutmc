import React from 'react';
import { NetworkIcon, OutlinedHddIcon, ServerGroupIcon } from '@patternfly/react-icons';
import ContainerCard, { ContainerCardDetail } from '../Shared/ContainerCard';
import { Server } from '../../types';

interface ServerCardProps {
    server: Server;
    onClick: (serverId: string) => void;
}

const getServerIcon = (type: 'proxy' | 'lobby' | 'game') => {
    switch (type) {
        case 'proxy':
            return <NetworkIcon />;
        case 'lobby':
            return <OutlinedHddIcon />;
        case 'game':
            return <ServerGroupIcon />;
        default:
            return <ServerGroupIcon />;
    }
};

const getServerTypeLabel = (type: 'proxy' | 'lobby' | 'game') => {
    switch (type) {
        case 'proxy':
            return 'Proxy';
        case 'lobby':
            return 'Lobby';
        case 'game':
            return 'Game';
        default:
            return 'Server';
    }
};

const ServerCard: React.FC<ServerCardProps> = ({ server, onClick }) => {
    const details: ContainerCardDetail[] = [
        { label: 'Version', value: server.version },
        { label: 'Players', value: `${server.players} / ${server.maxPlayers}` },
        { label: 'Uptime', value: server.uptime },
        { label: 'Address', value: `${server.ip}:${server.port}` }
    ];

    return (
        <ContainerCard
            id={server.id}
            name={server.name}
            status={server.status}
            icon={getServerIcon(server.type)}
            typeLabel={getServerTypeLabel(server.type)}
            details={details}
            cpu={server.cpu}
            memory={server.memory}
            onClick={() => onClick(server.id)}
        />
    );
};

export default ServerCard;
