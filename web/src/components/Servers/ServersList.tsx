import React, {useEffect, useState} from 'react';
import {useNavigate} from 'react-router-dom';
import {
    Button,
    Card,
    CardBody,
    CardTitle,
    EmptyState,
    EmptyStateBody,
    EmptyStateVariant,
    ExpandableSection,
    Flex,
    FlexItem,
    Gallery,
    PageSection,
    SearchInput
} from '@patternfly/react-core';
import {
    PlusIcon,
    SyncAltIcon
} from '@patternfly/react-icons';
import {useServerStore} from '../../store/serverStore';
import StatusBadge from '../UI/StatusBadge';
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
    const [isGitOpsSyncButtonLoading, setIsGitOpsSyncButtonLoading] = useState(false);
    const [gitOpsStatus, setGitOpsStatus] = useState<api.GitOpsStatus | null>(null);
    const [isGitOpsSectionExpanded, setIsGitOpsSectionExpanded] = useState(false);
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

    const handleTriggerGitOpsSync = async () => {
        setIsGitOpsSyncButtonLoading(true);
        try {
            await api.triggerGitOpsSync();
            const statusResponse = await api.getGitOpsStatus();
            setGitOpsStatus(statusResponse.data);
        } catch (error) {
            console.error('Failed to trigger GitOps sync:', error);
        } finally {
            setIsGitOpsSyncButtonLoading(false);
        }
    };

    const isGitOpsEnabled = gitOpsStatus?.enabled === true;
    const formatTime = (value?: string) => {
        if (!value) return 'Never';
        return new Date(value).toLocaleString();
    };
    const gitOpsBadgeStatus = (() => {
        if (!isGitOpsEnabled) {
            return 'offline';
        }
        if (gitOpsStatus?.state === 'error') {
            return 'offline';
        }
        if (gitOpsStatus?.state === 'syncing') {
            return 'syncing';
        }
        return 'online';
    })();
    const isGitOpsSyncInProgress = isGitOpsSyncButtonLoading || gitOpsStatus?.state === 'syncing';

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

                <div className="pf-v6-u-mt-xl mt-8">
                    <ExpandableSection
                        isExpanded={isGitOpsSectionExpanded}
                        onToggle={(_event, expanded) => setIsGitOpsSectionExpanded(expanded)}
                        displaySize="lg"
                        toggleContent={
                            <Flex justifyContent={{default: 'justifyContentFlexStart'}}
                                  alignItems={{default: 'alignItemsCenter'}}
                                  style={{width: '100%'}}>
                                <FlexItem>
                                    <CardTitle>GitOpsSync</CardTitle>
                                </FlexItem>
                                <FlexItem>
                                    <StatusBadge
                                        status={gitOpsBadgeStatus}
                                    />
                                </FlexItem>
                            </Flex>
                        }
                    >
                        <div className="pf-v6-u-mt-sm pf-v6-u-color-200">
                            Mode: <strong>{isGitOpsEnabled ? 'Enabled' : 'Disabled'}</strong> | Last
                            sync: <strong>{formatTime(gitOpsStatus?.lastSyncAt)}</strong>
                            {gitOpsStatus?.lastSyncCommit ? <> |
                                Commit: <strong>{gitOpsStatus.lastSyncCommit}</strong></> : null}
                        </div>
                        {gitOpsStatus?.lastSyncCommitMessage ? (
                            <div className="pf-v6-u-mt-xs pf-v6-u-color-200">
                                Commit
                                message: <strong>{gitOpsStatus.lastSyncCommitMessage}</strong>
                            </div>
                        ) : null}
                        {gitOpsStatus?.lastError ? (
                            <div className="pf-v6-u-mt-xs pf-v6-u-danger-color-100">
                                Last error: {gitOpsStatus.lastError}
                            </div>
                        ) : null}

                        <div className="pf-v6-u-mt-lg-on-md mt-5">
                            <Button
                                variant="secondary"
                                size="sm"
                                onClick={handleTriggerGitOpsSync}
                                isDisabled={!isGitOpsEnabled || isGitOpsSyncInProgress}
                                isLoading={isGitOpsSyncInProgress}
                                spinnerAriaLabel="GitOps sync in progress"
                            >
                                Trigger Sync
                            </Button>
                        </div>
                    </ExpandableSection>
                </div>


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
