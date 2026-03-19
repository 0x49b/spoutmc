import React, {useEffect} from 'react';
import {useNavigate} from 'react-router-dom';
import {
    Button,
    Card,
    CardBody,
    CardTitle,
    Flex,
    FlexItem,
    Gallery,
    Grid,
    PageSection
} from '@patternfly/react-core';
import {
    ArrowRightIcon,
    CubeIcon,
    CubesIcon,
    ServerIcon,
    UsersIcon
} from '@patternfly/react-icons';
import {useServerStore} from '../../store/serverStore';
import {usePlayerStore} from '../../store/playerStore';
import {usePluginStore} from '../../store/pluginStore';
import PageHeader from '../UI/PageHeader';
import LoadingSpinner from '../UI/LoadingSpinner';

const Dashboard: React.FC = () => {
    const navigate = useNavigate();
    const {servers, fetchServers} = useServerStore();
    const {players, getBannedPlayers, fetchPlayers, loading: playersLoading} = usePlayerStore();
    const {plugins, getEnabledPlugins, fetchPlugins, loading: pluginsLoading} = usePluginStore();

    useEffect(() => {
        fetchServers();
        fetchPlayers();
        fetchPlugins();
    }, [fetchServers, fetchPlayers, fetchPlugins]);

    const onlineServers = servers.filter(server => server.status === 'online').length;
    const totalPlayers = players.length;
    const bannedPlayers = getBannedPlayers().length;
    const enabledPlugins = getEnabledPlugins().length;

    // Find servers with high resource usage (CPU or Memory > 80%)
    const highResourceServers = servers.filter(
        server => server.cpu > 80 || server.memory > 80
    );

    const isLoading = playersLoading || pluginsLoading;

    if (isLoading) {
        return <LoadingSpinner/>;
    }

    return (
        <>
            <PageHeader
                title="Dashboard"
                description="Overview of your server infrastructure"
            />

            <PageSection>
                <Grid hasGutter>


                    {/* Stats Cards */}
                    <Gallery hasGutter minWidths={{default: '50%', sm: '50%', lg: '25%'}}
                             className="pf-v6-u-mb-lg">
                        <Card isCompact>
                            <CardTitle>
                                <Flex alignItems={{default: 'alignItemsCenter'}}>
                                    <FlexItem>
                                        <ServerIcon style={{
                                            fontSize: '24px',
                                            color: 'var(--pf-v6-global--primary-color--100)'
                                        }}/>
                                    </FlexItem>
                                    <FlexItem>
                                        <div>
                                            <div
                                                className="pf-v6-u-font-size-sm pf-v6-u-color-200">Servers
                                            </div>
                                            <div
                                                className="pf-v6-u-font-size-2xl pf-v6-u-font-weight-bold">{onlineServers}/{servers.length}</div>
                                        </div>
                                    </FlexItem>
                                </Flex>
                            </CardTitle>
                            <CardBody>
                                <Button variant="link" isInline component="a"
                                        onClick={() => navigate('/servers')}>
                                    View all servers <ArrowRightIcon/>
                                </Button>
                            </CardBody>
                        </Card>

                        <Card isCompact>
                            <CardTitle>
                                <Flex alignItems={{default: 'alignItemsCenter'}}>
                                    <FlexItem>
                                        <UsersIcon style={{
                                            fontSize: '24px',
                                            color: 'var(--pf-v6-global--info-color--100)'
                                        }}/>
                                    </FlexItem>
                                    <FlexItem>
                                        <div>
                                            <div
                                                className="pf-v6-u-font-size-sm pf-v6-u-color-200">Players
                                            </div>
                                            <div
                                                className="pf-v6-u-font-size-2xl pf-v6-u-font-weight-bold">{totalPlayers}</div>
                                        </div>
                                    </FlexItem>
                                </Flex>
                            </CardTitle>
                            <CardBody>
                                <div className="pf-v6-u-mb-sm">
                                    <span className="pf-v6-u-font-size-sm pf-v6-u-color-200">
                                        Banned players: {bannedPlayers}
                                    </span>
                                </div>
                                <Button variant="link" isInline component="a"
                                        onClick={() => navigate('/players')}>
                                    View all players <ArrowRightIcon/>
                                </Button>
                                <br />
                                <Button variant="link" isInline component="a"
                                        onClick={() => navigate('/players/banned')}>
                                    View banned players <ArrowRightIcon/>
                                </Button>
                            </CardBody>
                        </Card>

                        <Card isCompact>
                            <CardTitle>
                                <Flex alignItems={{default: 'alignItemsCenter'}}>
                                    <FlexItem>
                                        <CubeIcon style={{
                                            fontSize: '24px',
                                            color: 'var(--pf-v6-global--success-color--100)'
                                        }}/>
                                    </FlexItem>
                                    <FlexItem>
                                        <div>
                                            <div
                                                className="pf-v6-u-font-size-sm pf-v6-u-color-200">Plugins
                                            </div>
                                            <div
                                                className="pf-v6-u-font-size-2xl pf-v6-u-font-weight-bold">{enabledPlugins}/{plugins.length}</div>
                                        </div>
                                    </FlexItem>
                                </Flex>
                            </CardTitle>
                            <CardBody>
                                <Button variant="link" isInline component="a"
                                        onClick={() => navigate('/plugins')}>
                                    View all plugins <ArrowRightIcon/>
                                </Button>
                            </CardBody>
                        </Card>

                        <Card isCompact>
                            <CardTitle>
                                <Flex alignItems={{default: 'alignItemsCenter'}}>
                                    <FlexItem>
                                        <CubesIcon style={{
                                            fontSize: '24px',
                                            color: 'var(--pf-v6-global--success-color--100)'
                                        }}/>
                                    </FlexItem>
                                    <FlexItem>
                                        <div>
                                            <div
                                                className="pf-v6-u-font-size-sm pf-v6-u-color-200">Resource
                                                Alerts
                                            </div>
                                        </div>
                                    </FlexItem>
                                </Flex>
                            </CardTitle>
                            <CardBody>
                                {highResourceServers.length === 0 ? (
                                    <span>No Resource Alerts</span>
                                ) : (
                                    <Button variant="link" isInline component="a"
                                            onClick={() => navigate('/alerts')}>
                                        View all plugins <ArrowRightIcon/>
                                    </Button>
                                )}
                            </CardBody>
                        </Card>
                    </Gallery>
                </Grid>
            </PageSection>
        </>
    );
};

export default Dashboard;
