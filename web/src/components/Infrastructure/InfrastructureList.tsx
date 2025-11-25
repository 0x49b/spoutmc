import React, {useEffect, useState} from 'react';
import {useNavigate} from 'react-router-dom';
import {
    Button,
    EmptyState,
    EmptyStateBody,
    EmptyStateVariant,
    PageSection,
    Toolbar,
    ToolbarContent,
    ToolbarItem
} from '@patternfly/react-core';
import {ActionsColumn, IAction, Table, Tbody, Td, Th, Thead, Tr} from '@patternfly/react-table';
import {DatabaseIcon} from '@patternfly/react-icons';
import PageHeader from '../UI/PageHeader';
import StatusBadge from '../UI/StatusBadge';
import LoadingSpinner from '../UI/LoadingSpinner';
import {useInfrastructureStore} from '../../store/infrastructureStore';
import {InfrastructureContainer} from '../../types';

const InfrastructureList: React.FC = () => {
    const navigate = useNavigate();
    const {
        containers,
        loading,
        error,
        fetchInfrastructure,
        connectSSE,
        disconnectSSE,
        restartContainer,
        stopContainer
    } = useInfrastructureStore();
    const [actionInProgress, setActionInProgress] = useState<string | null>(null);

    useEffect(() => {
        // Initial fetch
        fetchInfrastructure();

        // Connect to SSE for real-time updates
        connectSSE();

        // Cleanup on unmount
        return () => {
            disconnectSSE();
        };
    }, [fetchInfrastructure, connectSSE, disconnectSSE]);

    /*const getStatusColor = (state: string): 'success' | 'danger' | 'warning' | 'info' => {
        switch (state) {
            case 'running':
                return 'success';
            case 'exited':
            case 'dead':
                return 'danger';
            case 'paused':
                return 'warning';
            default:
                return 'info';
        }
    };*/

    const getTypeLabel = (type: string) => {
        switch (type) {
            case 'database':
                return 'Database';
            default:
                return 'Unknown';
        }
    };

    const handleRestart = async (containerId: string) => {
        setActionInProgress(containerId);
        try {
            await restartContainer(containerId);
        } catch (error) {
            console.error('Failed to restart container:', error);
        } finally {
            setActionInProgress(null);
        }
    };

    const handleStop = async (containerId: string) => {
        setActionInProgress(containerId);
        try {
            await stopContainer(containerId);
        } catch (error) {
            console.error('Failed to stop container:', error);
        } finally {
            setActionInProgress(null);
        }
    };

    const getActions = (container: InfrastructureContainer): IAction[] => {
        const isRunning = container.summary.State === 'running';
        const isActionInProgress = actionInProgress === container.summary.ID;

        return [
            {
                title: 'View Details',
                onClick: () => navigate(`/infrastructure/${container.summary.ID}`)
            },
            {
                title: 'Restart',
                onClick: () => handleRestart(container.summary.ID),
                isDisabled: isActionInProgress
            },
            {
                title: 'Stop',
                onClick: () => handleStop(container.summary.ID),
                isDisabled: !isRunning || isActionInProgress
            }
        ];
    };

    const columnNames = {
        name: 'Name',
        type: 'Type',
        status: 'Status',
        image: 'Image',
        ports: 'Ports'
    };

    if (loading) {
        return (
            <>
                <PageHeader title="Infrastructure" description="Manage infrastructure containers"/>
                <PageSection>
                    <LoadingSpinner/>
                </PageSection>
            </>
        );
    }

    if (error) {
        return (
            <>
                <PageHeader title="Infrastructure" description="Manage infrastructure containers"/>
                <PageSection>
                    <EmptyState variant={EmptyStateVariant.lg}
                                titleText="Error loading infrastructure">
                        <EmptyStateBody>{error}</EmptyStateBody>
                        <Button variant="primary" onClick={fetchInfrastructure}>
                            Retry
                        </Button>
                    </EmptyState>
                </PageSection>
            </>
        );
    }

    if (containers.length === 0) {
        return (
            <>
                <PageHeader title="Infrastructure" description="Manage infrastructure containers"/>
                <PageSection>
                    <EmptyState variant={EmptyStateVariant.lg}
                                titleText="No infrastructure containers">
                        <EmptyStateBody>
                            No infrastructure containers are currently configured.
                        </EmptyStateBody>
                    </EmptyState>
                </PageSection>
            </>
        );
    }

    return (
        <>
            <PageHeader title="Infrastructure" description="Manage infrastructure containers"/>
            <PageSection>
                <Toolbar>
                    <ToolbarContent>
                        <ToolbarItem>
                            <Button variant="primary" onClick={fetchInfrastructure}>
                                Refresh
                            </Button>
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>

                <Table aria-label="Infrastructure containers table" variant="compact">
                    <Thead>
                        <Tr>
                            <Th>{columnNames.name}</Th>
                            <Th>{columnNames.type}</Th>
                            <Th>{columnNames.status}</Th>
                            <Th>{columnNames.image}</Th>
                            <Th>{columnNames.ports}</Th>
                            <Th></Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {containers.map((container) => {
                            const containerName = container.summary.Names?.[0]?.replace(/^\//, '') || 'Unnamed';
                            const ports = container.summary.Ports && container.summary.Ports.length > 0
                                ? container.summary.Ports.map((port: any) =>
                                    port.PublicPort ? `${port.PublicPort}:${port.PrivatePort}` : port.PrivatePort
                                ).join(', ')
                                : '-';

                            return (
                                <Tr key={container.summary.ID}>
                                    <Td dataLabel={columnNames.name}>
                                        <div style={{
                                            display: 'flex',
                                            alignItems: 'center',
                                            gap: '0.5rem'
                                        }}>
                                            <DatabaseIcon/>
                                            <span>{containerName}</span>
                                        </div>
                                    </Td>
                                    <Td dataLabel={columnNames.type}>
                                        {getTypeLabel(container.type)}
                                    </Td>
                                    <Td dataLabel={columnNames.status}>
                                        <StatusBadge
                                            status={container.summary.State}
                                            /*color={getStatusColor(container.summary.State)}*/
                                        />
                                    </Td>
                                    <Td dataLabel={columnNames.image}>
                                        {container.summary.Image}
                                    </Td>
                                    <Td dataLabel={columnNames.ports}>
                                        {ports}
                                    </Td>
                                    <Td isActionCell>
                                        <ActionsColumn items={getActions(container)}/>
                                    </Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                </Table>
            </PageSection>
        </>
    );
};

export default InfrastructureList;
