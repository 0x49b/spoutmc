import React, {useEffect, useState} from 'react';
import {useNavigate} from 'react-router-dom';
import {
    Button,
    Card,
    CardBody,
    CardTitle,
    Divider,
    EmptyState,
    EmptyStateBody,
    EmptyStateVariant,
    Flex,
    FlexItem,
    Gallery,
    PageSection,
    Progress,
    ProgressMeasureLocation,
    ProgressVariant,
    SearchInput
} from '@patternfly/react-core';
import {PlusIcon, SyncAltIcon,} from '@patternfly/react-icons';
import {useServerStore} from '../../store/serverStore';
import StatusBadge from '../UI/StatusBadge';
import PageHeader from '../UI/PageHeader';
import AddServerModal from './Modals/AddServerModal.tsx';
import ServerCardSkeleton from './ServerCardSkeleton';

const ServersList: React.FC = () => {
    const {
        servers,
        loading,
        fetchServers,
        setSelectedServer,
        addServer,
        connectSSE,
        disconnectSSE
    } = useServerStore();
    const [searchTerm, setSearchTerm] = useState('');
    const [isAddModalOpen, setIsAddModalOpen] = useState(false);
    const [isLoading, setLoading] = useState(false);
    const navigate = useNavigate();

    useEffect(() => {
        // Initial fetch
        loadServers();

        // Connect to SSE for real-time updates
        connectSSE();

        // Cleanup on unmount
        return () => {
            disconnectSSE();
        };
    }, [fetchServers, connectSSE, disconnectSSE]);

    const loadServers = () => {
        setLoading(true);
        fetchServers().then(() => setLoading(false));
    }

    // Filter servers based on search term
    const filteredServers = servers.filter(server =>
        server.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        server.ip.includes(searchTerm)
    );

    const handleServerClick = (serverId: string) => {
        setSelectedServer(serverId);
        navigate(`/servers/${serverId}`);
    };

    const handleAddServer = async (serverData: any) => {
        await addServer(serverData);
    };

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

    return (
        <>
            <PageHeader
                title="Servers"
                description="Manage and monitor your game servers"
                actions={
                    <>
                        <Button
                            variant="primary"
                            icon={<PlusIcon/>}
                            onClick={() => setIsAddModalOpen(true)}>
                            Add Server
                        </Button>
                        <Button
                            variant="secondary"
                            icon={<SyncAltIcon
                                className={loading ? 'pf-v6-u-animation-spin' : ''}/>}
                            onClick={() => loadServers()}
                            isDisabled={loading}>
                            {isLoading ? 'Refreshing...' : 'Refresh'}
                        </Button>
                    </>
                }
            />

            <PageSection>
                {/* Search */}
                <div className="pf-v6-u-mb-md mb-2">
                    <SearchInput
                        placeholder="Search servers by name or IP..."
                        value={searchTerm}
                        onChange={(_event, value) => setSearchTerm(value)}
                        onClear={() => setSearchTerm('')}
                    />
                </div>

                {/* Servers grid */}
                {loading && servers.length === 0 ? (
                    <Gallery hasGutter minWidths={{default: '100%', md: '50%', lg: '33.33%'}}>
                        {[...Array(6)].map((_, index) => (
                            <ServerCardSkeleton key={index}/>
                        ))}
                    </Gallery>
                ) : filteredServers.length === 0 ? (
                    <Card>
                        <CardBody>
                            <EmptyState variant={EmptyStateVariant.lg}>
                                {/*<EmptyStateIcon icon={ServerIcon} />*/}
                                <EmptyStateBody>
                                    <strong>No servers found</strong>
                                    <p>
                                        {searchTerm
                                            ? "No servers match your search criteria."
                                            : "There are no servers to display."}
                                    </p>
                                </EmptyStateBody>
                            </EmptyState>
                        </CardBody>
                    </Card>
                ) : (
                    <Gallery hasGutter minWidths={{default: '25%'}}>
                        {filteredServers.map((server) => (
                            <Card
                                key={server.id}
                                onClick={() => handleServerClick(server.id)}
                            >
                                <CardTitle>
                                    <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}
                                          alignItems={{default: 'alignItemsFlexStart'}}>
                                        <FlexItem>
                                            <strong>{server.name}</strong>
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
                                        <Flex
                                            justifyContent={{default: 'justifyContentSpaceBetween'}}>
                                            <FlexItem>Version:</FlexItem>
                                            <FlexItem><strong>{server.version}</strong></FlexItem>
                                        </Flex>
                                        <Flex
                                            justifyContent={{default: 'justifyContentSpaceBetween'}}>
                                            <FlexItem>Players:</FlexItem>
                                            <FlexItem><strong>{server.players} / {server.maxPlayers}</strong></FlexItem>
                                        </Flex>
                                        <Flex
                                            justifyContent={{default: 'justifyContentSpaceBetween'}}>
                                            <FlexItem>Uptime:</FlexItem>
                                            <FlexItem><strong>{server.uptime}</strong></FlexItem>
                                        </Flex>
                                        <Flex
                                            justifyContent={{default: 'justifyContentSpaceBetween'}}>
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
                                                <Flex
                                                    justifyContent={{default: 'justifyContentSpaceBetween'}}>
                                                    <FlexItem
                                                        className="pf-v6-u-font-size-sm pf-v6-u-color-200">CPU
                                                        Usage</FlexItem>
                                                    <FlexItem
                                                        className="pf-v6-u-font-size-sm"><strong>{server.cpu}%</strong></FlexItem>
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
                                                <Flex
                                                    justifyContent={{default: 'justifyContentSpaceBetween'}}>
                                                    <FlexItem
                                                        className="pf-v6-u-font-size-sm pf-v6-u-color-200">Memory
                                                        Usage</FlexItem>
                                                    <FlexItem
                                                        className="pf-v6-u-font-size-sm"><strong>{server.memory}%</strong></FlexItem>
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
                        ))}
                    </Gallery>
                )}

                {/* Add Server Modal */}
                <AddServerModal
                    isOpen={isAddModalOpen}
                    onClose={() => setIsAddModalOpen(false)}
                    servers={servers}
                    onAdd={handleAddServer}
                />
            </PageSection>
        </>
    );
};

export default ServersList;
