import React from 'react';
import {
    Card,
    CardBody,
    CardTitle,
    Divider,
    Flex,
    FlexItem,
    Progress,
    ProgressMeasureLocation,
    ProgressVariant
} from '@patternfly/react-core';
import { NetworkIcon, OutlinedHddIcon, ServerGroupIcon } from '@patternfly/react-icons';
import StatusBadge from '../UI/StatusBadge';
import { Server } from '../../types';

interface ServerCardProps {
    server: Server;
    onClick: (serverId: string) => void;
}

const getCPUVariant = (cpu: number): ProgressVariant => {
    if (cpu > 80) return ProgressVariant.danger;
    if (cpu > 50) return ProgressVariant.warning;
    return ProgressVariant.success;
};

const getMemoryVariant = (memory: number): ProgressVariant => {
    if (memory > 80) return ProgressVariant.danger;
    if (memory > 50) return ProgressVariant.warning;
    return ProgressVariant.success;
};

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
    return (
        <Card onClick={() => onClick(server.id)}>
            <CardTitle>
                <Flex justifyContent={{default: 'justifyContentSpaceBetween'}} alignItems={{default: 'alignItemsFlexStart'}}>
                    <FlexItem>
                        {getServerIcon(server.type)} <strong>{server.name}</strong>
                    </FlexItem>
                    <FlexItem>
                        <StatusBadge status={server.status}/>
                    </FlexItem>
                </Flex>
            </CardTitle>
            <CardBody>
                <div style={{
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 'var(--pf-v6-global--spacer--sm)'
                }}>
                    <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
                        <FlexItem>Type:</FlexItem>
                        <FlexItem><strong>{getServerTypeLabel(server.type)}</strong></FlexItem>
                    </Flex>
                    <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
                        <FlexItem>Version:</FlexItem>
                        <FlexItem><strong>{server.version}</strong></FlexItem>
                    </Flex>
                    <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
                        <FlexItem>Players:</FlexItem>
                        <FlexItem><strong>{server.players} / {server.maxPlayers}</strong></FlexItem>
                    </Flex>
                    <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
                        <FlexItem>Uptime:</FlexItem>
                        <FlexItem><strong>{server.uptime}</strong></FlexItem>
                    </Flex>
                    <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
                        <FlexItem>Address:</FlexItem>
                        <FlexItem><strong>{server.ip}:{server.port}</strong></FlexItem>
                    </Flex>
                </div>

                <Divider className="pf-v6-u-my-md"/>

                <div style={{
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 'var(--pf-v6-global--spacer--md)'
                }}>
                    <div>
                        <div className="pf-v6-u-mb-xs">
                            <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
                                <FlexItem className="pf-v6-u-font-size-sm pf-v6-u-color-200">
                                    CPU Usage
                                </FlexItem>
                                <FlexItem className="pf-v6-u-font-size-sm">
                                    <strong>{server.cpu}%</strong>
                                </FlexItem>
                            </Flex>
                        </div>
                        <Progress
                            value={Math.min(server.cpu, 100)}
                            variant={getCPUVariant(server.cpu)}
                            measureLocation={ProgressMeasureLocation.none}
                            aria-label="CPU usage"
                        />
                    </div>

                    <div>
                        <div className="pf-v6-u-mb-xs">
                            <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
                                <FlexItem className="pf-v6-u-font-size-sm pf-v6-u-color-200">
                                    Memory Usage
                                </FlexItem>
                                <FlexItem className="pf-v6-u-font-size-sm">
                                    <strong>{server.memory}%</strong>
                                </FlexItem>
                            </Flex>
                        </div>
                        <Progress
                            value={Math.min(server.memory, 100)}
                            variant={getMemoryVariant(server.memory)}
                            measureLocation={ProgressMeasureLocation.none}
                            aria-label="Memory usage"
                        />
                    </div>
                </div>
            </CardBody>
        </Card>
    );
};

export default ServerCard;
