import React, {useEffect, useState} from 'react';
import {useNavigate} from 'react-router-dom';
import {
    Button,
    Card,
    CardBody,
    EmptyState,
    EmptyStateBody,
    EmptyStateVariant,
    Gallery,
    PageSection,
    SearchInput
} from '@patternfly/react-core';
import {PlusIcon, SyncAltIcon} from '@patternfly/react-icons';
import {useServerStore} from '../../store/serverStore';
import PageHeader from '../UI/PageHeader';
import AddServerModal from './Modals/AddServerModal.tsx';
import ServerCardSkeleton from './ServerCardSkeleton';
import ServerCard from './ServerCard.tsx';
import * as api from '../../service/apiService.ts';

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
    const [gitOpsStatus, setGitOpsStatus] = useState<api.GitOpsStatus | null>(null);
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

    useEffect(() => {
        let isMounted = true;

        const loadGitOpsStatus = async () => {
            try {
                const response = await api.getGitOpsStatus();
                if (isMounted) {
                    setGitOpsStatus(response.data);
                }
            } catch (error) {
                console.error('Failed to load GitOps status:', error);
            }
        };

        loadGitOpsStatus();
        const interval = setInterval(loadGitOpsStatus, 10000);

        return () => {
            isMounted = false;
            clearInterval(interval);
        };
    }, []);

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

    const isGitOpsEnabled = gitOpsStatus?.enabled === true;

    const serversGridContent = (() => {
        if (loading && servers.length === 0) {
            return (
                <Gallery hasGutter minWidths={{default: '100%', md: '50%', lg: '33.33%'}}>
                    {[...Array(6)].map((_, index) => (
                        <ServerCardSkeleton key={index}/>
                    ))}
                </Gallery>
            );
        }

        if (filteredServers.length === 0) {
            return (
                <Card>
                    <CardBody>
                        <EmptyState variant={EmptyStateVariant.lg}>
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
            );
        }

        return (
            <Gallery hasGutter minWidths={{default: '25%'}}>
                {filteredServers.map((server) => (
                    <ServerCard
                        key={server.id}
                        server={server}
                        onClick={handleServerClick}
                    />
                ))}
            </Gallery>
        );
    })();

    return (
        <>
            <PageHeader
                title="Game Server"
                description="Manage and monitor your game servers"
                actions={
                    <>
                        <Button
                            variant="primary"
                            icon={<PlusIcon/>}
                            onClick={() => setIsAddModalOpen(true)}
                            isDisabled={isGitOpsEnabled}>
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
                {serversGridContent}

                {/* Add Server Modal */}
                <AddServerModal
                    isOpen={isAddModalOpen && !isGitOpsEnabled}
                    onClose={() => setIsAddModalOpen(false)}
                    servers={servers}
                    onAdd={handleAddServer}
                />
            </PageSection>
        </>
    )
        ;
};

export default ServersList;
