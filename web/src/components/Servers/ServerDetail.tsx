import React, {useEffect, useRef, useState} from 'react';
import {useNavigate, useParams} from 'react-router-dom';
import {
    Alert,
    Button,
    Card,
    CardBody,
    EmptyState,
    EmptyStateBody,
    EmptyStateVariant,
    Flex,
    FlexItem,
    Grid,
    GridItem,
    Label,
    PageSection,
    Spinner,
    Tab,
    Tabs,
    TabTitleIcon,
    TabTitleText,
    Title
} from '@patternfly/react-core';
import {
    ArrowLeftIcon,
    ChartLineIcon,
    ClockIcon,
    CubeIcon,
    EditIcon,
    FileIcon,
    NetworkIcon,
    PowerOffIcon,
    SyncAltIcon,
    TerminalIcon,
    TrashIcon,
    UsersIcon
} from '@patternfly/react-icons';
import {useServerStore} from '../../store/serverStore.ts';
import {usePluginStore} from '../../store/pluginStore.ts';
import {Table, Tbody, Td, Th, Thead, Tr} from '@patternfly/react-table';
import PageHeader from '../UI/PageHeader.tsx';
import StatusBadge from '../UI/StatusBadge.tsx';
import {ConsoleTab} from './ServerDetailTabs/ConsoleTab.tsx';
import {OverviewTab} from './ServerDetailTabs/OverviewTab.tsx';
import {ProxyMotdTab} from './ServerDetailTabs/ProxyMotdTab.tsx';
import {ServerStats} from '../../model/ServerStats.ts';
import DeleteServerModal from './Modals/DeleteServerModal.tsx';
import StopServerModal from './Modals/StopServerModal.tsx';
import EditServerModal from './Modals/EditServerModal.tsx';
import FileBrowser from './FileBrowser.tsx';
import FileEditorModal from './Modals/FileEditorModal.tsx';
import * as api from '../../service/apiService.ts';
import RestartServerModal from "./Modals/RestartServerModal.tsx";
import {
    acquireServerRealtimeWsClient,
    RealtimeMessage,
    releaseServerRealtimeWsClient,
    ServerRealtimeWsClient,
    useServerDetailWsTransport
} from '../../service/serverRealtimeWs.ts';

const ServerDetail: React.FC = () => {
    const {id} = useParams<{ id: string }>();
    const navigate = useNavigate();
    const {
        getServerById,
        fetchServers,
        restartServer,
        stopServer,
        startServer,
        deleteServer,
        updateServer
    } = useServerStore();
    const {fetchPlugins, getPluginsForServer} = usePluginStore();
    const [isRestarting, setIsRestarting] = useState(false);
    const [isPowerActionLoading, setIsPowerActionLoading] = useState(false);
    const [activeTab, setActiveTab] = useState<string | number>('overview');
    const [stats, setStats] = useState<ServerStats | null>(null);
    const [isInitialLoading, setIsInitialLoading] = useState(true);
    const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
    const [isDeleting, setIsDeleting] = useState(false);
    const [isStopModalOpen, setIsStopModalOpen] = useState(false);
    const [isEditModalOpen, setIsEditModalOpen] = useState(false);
    const [isRestartModalOpen, setIsRestartModalOpen] = useState(false);
    const [volumeFiles, setVolumeFiles] = useState<api.VolumeFiles[]>([]);
    const [isLoadingFiles, setIsLoadingFiles] = useState(false);
    const [fileNotice, setFileNotice] = useState<string | null>(null);
    const [selectedFile, setSelectedFile] = useState<{
        path: string;
        fileName: string;
        volume?: string
    } | null>(null);
    const [gitOpsStatus, setGitOpsStatus] = useState<api.GitOpsStatus | null>(null);
    const statsEventSourceRef = useRef<EventSource | null>(null);
    const realtimeWsRef = useRef<ServerRealtimeWsClient | null>(null);
    const useWsTransport = useServerDetailWsTransport();

    const server = getServerById(id || '');

    const registryPluginsForServer = server ? getPluginsForServer(server.name) : [];

    useEffect(() => {
        fetchPlugins();
    }, [fetchPlugins]);

    useEffect(() => {
        if (!server && id) {
            void fetchServers();
        }
    }, [server, id, fetchServers]);

    // Set up SSE connection for server stats
    useEffect(() => {
        if (server?.id && !useWsTransport) {
            const statsUrl = api.withSSEAuth(api.buildApiUrl(`/server/${server.id}/stats`));

            statsEventSourceRef.current = new EventSource(statsUrl);
            statsEventSourceRef.current.onmessage = (event: MessageEvent) => {
                const parsed = JSON.parse(event.data);
                setStats(parsed);
                setIsInitialLoading(false);
            };

            statsEventSourceRef.current.onerror = () => {
                setIsInitialLoading(false);
            };
        }

        return () => {
            if (statsEventSourceRef.current) {
                statsEventSourceRef.current.close();
                statsEventSourceRef.current = null;
            }
        };
    }, [server?.id, useWsTransport]);

    useEffect(() => {
        if (!server?.id || !useWsTransport) {
            return;
        }

        const wsClient = acquireServerRealtimeWsClient(server.id, api.buildWsUrl(`/ws/server/${server.id}`));
        realtimeWsRef.current = wsClient;

        wsClient.addListener({
            id: 'server-detail-stats',
            onOpen: () => {
                wsClient.subscribe('stats');
            },
            onMessage: (message: RealtimeMessage) => {
                if (message.type === 'stats' && message.payload) {
                    setStats(message.payload as ServerStats);
                    setIsInitialLoading(false);
                }
                if (message.type === 'error' && message.channel === 'stats') {
                    setIsInitialLoading(false);
                }
            },
            onError: () => {
                setIsInitialLoading(false);
            },
            onClose: () => {
                setIsInitialLoading(false);
            }
        });
        wsClient.connect();

        return () => {
            wsClient.unsubscribe('stats');
            wsClient.removeListener('server-detail-stats');
            releaseServerRealtimeWsClient(server.id);
            if (realtimeWsRef.current === wsClient) {
                realtimeWsRef.current = null;
            }
        };
    }, [server?.id, useWsTransport]);

    // Load files when Files tab is opened
    useEffect(() => {
        if (activeTab === 'files' && server?.id) {
            loadServerFiles();
        }
    }, [activeTab, server?.id]);

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

    const loadServerFiles = async () => {
        if (!server?.id) return;

        setIsLoadingFiles(true);
        try {
            const response = await api.listServerFiles(server.id);
            setVolumeFiles(response.data.volumes);
        } catch (error) {
            console.error('Failed to load server files:', error);
        } finally {
            setIsLoadingFiles(false);
        }
    };

    const handleFileClick = (filePath: string, volume?: string) => {
        const fileName = filePath.split('/').pop() || filePath;
        const ext = fileName.includes('.') ? fileName.split('.').pop()?.toLowerCase() : '';

        const nonEditableExts = new Set([
            'jar', 'zip', '7z', 'gz', 'bz2', 'xz', 'tar',
            'png', 'jpg', 'jpeg', 'gif', 'webp', 'ico',
            'mp3', 'mp4', 'wav', 'ogg',
            'class', 'so', 'dll', 'dylib', 'exe'
        ]);
        if (ext && nonEditableExts.has(ext)) {
            setFileNotice(`"${fileName}" looks like a binary file and is not editable in the UI.`);
            return;
        }

        setFileNotice(null);
        setSelectedFile({path: filePath, fileName, volume});
    };

    const handleFileSave = async (content: string) => {
        if (!server?.id || !selectedFile) return;

        await api.updateServerFile(server.id, selectedFile.path, content, selectedFile.volume);
    };

    const handleUpdateServer = async (serverId: string, data: {
        name?: string;
        env?: Record<string, string>
    }) => {
        await updateServer(serverId, data);
    };

    if (!server) {
        return (
            <PageSection>
                <Button
                    variant="secondary"
                    icon={<ArrowLeftIcon/>}
                    onClick={() => navigate('/servers')}
                >
                    Back to Servers
                </Button>
                <EmptyState variant={EmptyStateVariant.lg} titleText="Server not found"
                            className="pf-v6-u-mt-lg">
                </EmptyState>
            </PageSection>
        );
    }

    const handleRestart = async () => {
        setIsRestarting(true);
        try {
            await restartServer(server.id);
        } finally {
            setIsRestartModalOpen(false);
            setIsRestarting(false);
        }
    };

    const handleRestartAction = () => {
        setIsRestartModalOpen(true);
    }

    const handlePowerAction = async () => {
        if (server.status === 'online') {
            setIsStopModalOpen(true);
        } else {
            setIsPowerActionLoading(true);
            try {
                await startServer(server.id);
            } finally {
                setIsPowerActionLoading(false);
            }
        }
    };

    const handleStopServer = async () => {
        setIsPowerActionLoading(true);
        try {
            await stopServer(server.id);
        } finally {
            setIsPowerActionLoading(false);
            setIsStopModalOpen(false);
        }
    };

    const handleDeleteServer = async (removeData: boolean) => {
        if (gitOpsStatus?.enabled) {
            return;
        }
        setIsDeleting(true);
        try {
            await deleteServer(server.id, removeData);
            await new Promise(resolve => setTimeout(resolve, 500));
            navigate('/servers', {replace: true});
        } catch (error) {
            console.error('Failed to delete server:', error);
            alert('Failed to delete server. Please try again.');
            setIsDeleting(false);
            setIsDeleteModalOpen(false);
        }
    };

    return (
        <>
            <PageHeader
                title={server.name}
                description={`Server details and management for ${server.name}`}
                serverStatus={<StatusBadge status={server.status}/>}
                actions={
                    <>

                        <Button
                            variant="secondary"
                            icon={<ArrowLeftIcon/>}
                            onClick={() => navigate('/servers')}
                        >
                            Back to Servers
                        </Button>
                        <Button
                            variant="secondary"
                            icon={<EditIcon/>}
                            onClick={() => setIsEditModalOpen(true)}
                            isDisabled={isRestarting || isPowerActionLoading}
                        >
                            Edit Server
                        </Button>
                        {server.status === 'online' && (
                            <Button
                                variant="secondary"
                                icon={<SyncAltIcon
                                    className={isRestarting ? 'pf-v6-u-animation-spin' : ''}/>}
                                onClick={handleRestartAction}
                                isDisabled={isRestarting || isPowerActionLoading}
                            >

                                {isRestarting ? 'Restarting...' : 'Restart Server'}
                            </Button>
                        )}
                        <Button
                            variant={server.status === 'online' ? 'danger' : 'primary'}
                            icon={isPowerActionLoading ? <Spinner size="md"/> : <PowerOffIcon/>}
                            onClick={handlePowerAction}
                            isDisabled={server.status === 'restarting' || isRestarting || isPowerActionLoading}
                        >
                            {isPowerActionLoading ? (
                                server.status === 'online' ? 'Stopping...' : 'Starting...'
                            ) : (
                                server.status === 'online' ? 'Stop Server' : 'Start Server'
                            )}
                        </Button>
                        <Button
                            variant="danger"
                            icon={<TrashIcon/>}
                            onClick={() => setIsDeleteModalOpen(true)}
                            isDisabled={isRestarting || isPowerActionLoading || gitOpsStatus?.enabled === true}
                        />
                    </>
                }
            />

            {/* Modals */}
            <StopServerModal
                isOpen={isStopModalOpen}
                onClose={() => setIsStopModalOpen(false)}
                onConfirm={handleStopServer}
                serverName={server.name}
                isLoading={isPowerActionLoading}
            />

            <DeleteServerModal
                isOpen={isDeleteModalOpen}
                onClose={() => setIsDeleteModalOpen(false)}
                onConfirm={handleDeleteServer}
                serverName={server.name}
                isLoading={isDeleting}
            />

            <EditServerModal
                isOpen={isEditModalOpen}
                onClose={() => setIsEditModalOpen(false)}
                server={server}
                onUpdate={handleUpdateServer}
            />

            <RestartServerModal
                isOpen={isRestartModalOpen}
                onClose={() => setIsRestartModalOpen(false)}
                onConfirm={handleRestart}
                serverName={server.name}
                isLoading={isRestarting}
            />

            {selectedFile && (
                <FileEditorModal
                    isOpen={!!selectedFile}
                    onClose={() => setSelectedFile(null)}
                    filePath={selectedFile.path}
                    fileName={selectedFile.fileName}
                    serverId={server.id}
                    volume={selectedFile.volume}
                    onSave={handleFileSave}
                />
            )}

            <PageSection>
                <Grid hasGutter>
                    {/* Server Status Cards */}
                    <Grid hasGutter className="pf-v6-u-mb-lg">
                        <GridItem span={12} md={6} lg={3}>
                            <Card isCompact>
                                <CardBody>
                                    <Flex alignItems={{default: 'alignItemsCenter'}}>
                                        <FlexItem spacer={{default: 'spacerSm'}}>
                                            <UsersIcon style={{
                                                fontSize: '20px',
                                                color: 'var(--pf-v6-global--info-color--100)'
                                            }}/>
                                        </FlexItem>
                                        <FlexItem>
                                            <div>
                                                <div className="pf-v6-u-font-size-sm">Players</div>
                                                <div
                                                    className="pf-v6-u-font-size-xl pf-v6-u-font-weight-bold pf-v6-u-mt-xs">
                                                    {server.players}/{server.maxPlayers}
                                                </div>
                                            </div>
                                        </FlexItem>
                                    </Flex>
                                </CardBody>
                            </Card>
                        </GridItem>

                        <GridItem span={12} md={6} lg={3}>
                            <Card isCompact>
                                <CardBody>
                                    <Flex alignItems={{default: 'alignItemsCenter'}}>
                                        <FlexItem spacer={{default: 'spacerSm'}}>
                                            <ClockIcon style={{
                                                fontSize: '20px',
                                                color: 'var(--pf-v6-global--success-color--100)'
                                            }}/>
                                        </FlexItem>
                                        <FlexItem>
                                            <div>
                                                <div className="pf-v6-u-font-size-sm">Uptime</div>
                                                <div
                                                    className="pf-v6-u-font-size-xl pf-v6-u-font-weight-bold pf-v6-u-mt-xs">{server.uptime}</div>
                                            </div>
                                        </FlexItem>
                                    </Flex>
                                </CardBody>
                            </Card>
                        </GridItem>

                        <GridItem span={12} md={6} lg={3}>
                            <Card isCompact>
                                <CardBody>
                                    <Flex alignItems={{default: 'alignItemsCenter'}}>
                                        <FlexItem spacer={{default: 'spacerSm'}}>
                                            <CubeIcon style={{
                                                fontSize: '20px',
                                                color: 'var(--pf-v6-global--palette--purple-500)'
                                            }}/>
                                        </FlexItem>
                                        <FlexItem>
                                            <div>
                                                <div className="pf-v6-u-font-size-sm">Plugins</div>
                                                <div
                                                    className="pf-v6-u-font-size-xl pf-v6-u-font-weight-bold pf-v6-u-mt-xs">
                                                    {registryPluginsForServer.length}
                                                </div>
                                            </div>
                                        </FlexItem>
                                    </Flex>
                                </CardBody>
                            </Card>
                        </GridItem>
                    </Grid>

                    {/* Tabs */}
                    <Tabs
                        activeKey={activeTab}
                        onSelect={(_event, tabIndex) => setActiveTab(tabIndex)}
                        isBox
                    >
                        <Tab
                            eventKey="overview"
                            title={
                                <>
                                    <TabTitleIcon><ChartLineIcon/></TabTitleIcon>
                                    <TabTitleText>Overview</TabTitleText>
                                </>
                            }
                        >
                            <OverviewTab server={server} stats={stats}
                                         isInitialLoading={isInitialLoading}/>
                        </Tab>

                        <Tab
                            eventKey="players"
                            title={
                                <>
                                    <TabTitleIcon><UsersIcon/></TabTitleIcon>
                                    <TabTitleText>Players</TabTitleText>
                                </>
                            }
                        >
                            <Card>
                                <CardBody>
                                    <Title headingLevel="h3" size="lg">
                                        Connected Players ({server.players}/{server.maxPlayers})
                                    </Title>
                                    {server.players === 0 ? (
                                        <EmptyState titleText="No players are currently connected"
                                                    variant={EmptyStateVariant.sm}
                                                    className="pf-v6-u-mt-md">
                                            <EmptyStateBody>No players are currently
                                                connected</EmptyStateBody>
                                        </EmptyState>
                                    ) : (
                                        <p className="pf-v6-u-mt-md">Player list would go here</p>
                                    )}
                                </CardBody>
                            </Card>
                        </Tab>

                        <Tab
                            eventKey="plugins"
                            title={
                                <>
                                    <TabTitleIcon><CubeIcon/></TabTitleIcon>
                                    <TabTitleText>Plugins</TabTitleText>
                                </>
                            }
                        >
                            <Card>
                                <CardBody>
                                    <Title headingLevel="h3" size="lg">
                                        Plugins for this server ({registryPluginsForServer.length})
                                    </Title>
                                    {registryPluginsForServer.length === 0 ? (
                                        <EmptyState
                                            titleText="No registry plugins apply to this server"
                                            variant={EmptyStateVariant.sm}
                                            className="pf-v6-u-mt-md">
                                            <EmptyStateBody>
                                                Assign plugins in the Plugins page or rely on
                                                system-managed
                                                plugins for this server type.
                                            </EmptyStateBody>
                                        </EmptyState>
                                    ) : (
                                        <Table aria-label="Plugins for server" variant="compact"
                                               className="pf-v6-u-mt-md">
                                            <Thead>
                                                <Tr>
                                                    <Th>Name</Th>
                                                    <Th>URL</Th>
                                                    <Th>Source</Th>
                                                </Tr>
                                            </Thead>
                                            <Tbody>
                                                {registryPluginsForServer.map((p) => (
                                                    <Tr key={p.id}>
                                                        <Td dataLabel="Name">{p.name}</Td>
                                                        <Td dataLabel="URL">
                                                            <span className="pf-v6-u-font-size-sm"
                                                                  title={p.url}>
                                                                {p.url.length > 80 ? `${p.url.slice(0, 80)}…` : p.url}
                                                            </span>
                                                        </Td>
                                                        <Td dataLabel="Source">
                                                            {p.systemManaged ? (
                                                                <Label
                                                                    color="purple">System-managed</Label>
                                                            ) : (
                                                                <Label color="green">User</Label>
                                                            )}
                                                        </Td>
                                                    </Tr>
                                                ))}
                                            </Tbody>
                                        </Table>
                                    )}
                                </CardBody>
                            </Card>
                        </Tab>

                        <Tab
                            eventKey="console"
                            title={
                                <>
                                    <TabTitleIcon><TerminalIcon/></TabTitleIcon>
                                    <TabTitleText>Console</TabTitleText>
                                </>
                            }
                        >
                            <ConsoleTab
                                containerId={server.id}
                                logsUrl={api.withSSEAuth(api.buildApiUrl(`/server/${server.id}/logs`))}
                                commandUrl={api.buildApiUrl(`/server/${server.id}/command`)}
                                isActive={activeTab === 'console'}
                                enableSendCommand={true}
                                wsClient={useWsTransport ? realtimeWsRef.current : null}
                            />
                        </Tab>

                        <Tab
                            eventKey="files"
                            title={
                                <>
                                    <TabTitleIcon><FileIcon/></TabTitleIcon>
                                    <TabTitleText>Files</TabTitleText>
                                </>
                            }
                        >
                            <Card>
                                <CardBody>
                                    <Title headingLevel="h3" size="lg">Server Files</Title>
                                    {fileNotice ? (
                                        <Alert
                                            variant="info"
                                            isInline
                                            title="File not editable"
                                            className="pf-v6-u-mt-md"
                                        >
                                            {fileNotice}
                                        </Alert>
                                    ) : null}

                                    {isLoadingFiles ? (
                                        <div style={{
                                            display: 'flex',
                                            justifyContent: 'center',
                                            padding: '3rem'
                                        }}>
                                            <Spinner size="xl"/>
                                        </div>
                                    ) : volumeFiles.length === 0 ? (
                                        <EmptyState titleText="No volumes"
                                                    variant={EmptyStateVariant.sm}
                                                    className="pf-v6-u-mt-md">
                                            <EmptyStateBody>No volumes found for this
                                                server</EmptyStateBody>
                                        </EmptyState>
                                    ) : (
                                        <div style={{marginTop: 'var(--pf-v6-global--spacer--md)'}}>
                                            {volumeFiles.map((volume, index) => (
                                                <FileBrowser
                                                    key={index}
                                                    files={volume.files}
                                                    containerPath={volume.containerPath}
                                                    onFileClick={(filePath) => handleFileClick(filePath, volume.containerPath)}
                                                />
                                            ))}
                                        </div>
                                    )}
                                </CardBody>
                            </Card>
                        </Tab>
                        {server.type === 'proxy' && (
                            <Tab
                                eventKey="motd"
                                title={
                                    <>
                                        <TabTitleIcon><NetworkIcon/></TabTitleIcon>
                                        <TabTitleText>MOTD</TabTitleText>
                                    </>
                                }
                            >
                                <ProxyMotdTab
                                    serverId={server.id}
                                    serverName={server.name}
                                    gitOpsEnabled={gitOpsStatus?.enabled === true}
                                />
                            </Tab>
                        )}
                    </Tabs>
                </Grid>
            </PageSection>
        </>
    );
};

export default ServerDetail;
