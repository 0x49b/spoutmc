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
import {SyncAltIcon} from '@patternfly/react-icons';
import PageHeader from '../UI/PageHeader';
import LoadingSpinner from '../UI/LoadingSpinner';
import {useInfrastructureStore} from '../../store/infrastructureStore';
import {getContainerId} from '../../types';
import InfrastructureCard from './InfrastructureCard';

const InfrastructureList: React.FC = () => {
    const navigate = useNavigate();
    const {
        containers,
        loading,
        error,
        fetchInfrastructure,
        connectSSE,
        disconnectSSE
    } = useInfrastructureStore();
    const [searchTerm, setSearchTerm] = useState('');
    const [isLoading, setLoading] = useState(false);

    useEffect(() => {
        fetchInfrastructure();
        connectSSE();
        return () => disconnectSSE();
    }, [fetchInfrastructure, connectSSE, disconnectSSE]);

    const loadInfrastructure = () => {
        setLoading(true);
        fetchInfrastructure().then(() => setLoading(false));
    };

    const filteredContainers = containers.filter((container) => {
        const containerName = container.summary.Names?.[0]?.replace(/^\//, '') || 'Unnamed';
        return (
            containerName.toLowerCase().includes(searchTerm.toLowerCase()) ||
            container.summary.Image.toLowerCase().includes(searchTerm.toLowerCase())
        );
    });

    const handleContainerClick = (containerId: string) => {
        navigate(`/infrastructure/${containerId}`);
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
            <PageHeader
                title="Infrastructure"
                description="Manage infrastructure containers"
                actions={
                    <Button
                        variant="secondary"
                        icon={<SyncAltIcon className={loading ? 'pf-v6-u-animation-spin' : ''} />}
                        onClick={loadInfrastructure}
                        isDisabled={loading}
                    >
                        {isLoading ? 'Refreshing...' : 'Refresh'}
                    </Button>
                }
            />
            <PageSection>
                <div className="pf-v6-u-mb-md mb-2">
                    <SearchInput
                        placeholder="Search infrastructure by name or image..."
                        value={searchTerm}
                        onChange={(_event, value) => setSearchTerm(value)}
                        onClear={() => setSearchTerm('')}
                    />
                </div>

                {filteredContainers.length === 0 ? (
                    <Card>
                        <CardBody>
                            <EmptyState variant={EmptyStateVariant.lg}>
                                <EmptyStateBody>
                                    <strong>No infrastructure containers found</strong>
                                    <p>
                                        {searchTerm
                                            ? 'No infrastructure containers match your search criteria.'
                                            : 'There are no infrastructure containers to display.'}
                                    </p>
                                </EmptyStateBody>
                            </EmptyState>
                        </CardBody>
                    </Card>
                ) : (
                    <Gallery hasGutter minWidths={{default: '25%'}}>
                        {filteredContainers.map((container) => (
                            <InfrastructureCard
                                key={getContainerId(container.summary)}
                                container={container}
                                onClick={handleContainerClick}
                            />
                        ))}
                    </Gallery>
                )}
            </PageSection>
        </>
    );
};

export default InfrastructureList;
