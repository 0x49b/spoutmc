import React, { useEffect, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
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
  PageSection,
  Spinner,
  Tab,
  Tabs,
  TabTitleIcon,
  TabTitleText
} from '@patternfly/react-core';
import {
  ArrowLeftIcon,
  ChartLineIcon,
  ClockIcon,
  DatabaseIcon,
  EditIcon,
  PowerOffIcon,
  SyncAltIcon,
  TerminalIcon
} from '@patternfly/react-icons';
import { useInfrastructureStore } from '../../store/infrastructureStore';
import { getContainerId } from '../../types';
import * as api from '../../service/apiService';
import PageHeader from '../UI/PageHeader';
import StatusBadge from '../UI/StatusBadge';
import { ConsoleTab } from '../Servers/ServerDetailTabs/ConsoleTab';
import { InfrastructureOverviewTab } from './InfrastructureOverviewTab';
import { ServerStats } from '../../model/ServerStats';

const API_BASE_URL = 'http://localhost:3000/api/v1'; // TODO: use env/config

const getTypeLabel = (type: string) => {
  switch (type) {
    case 'database':
      return 'Database';
    default:
      return type || 'Unknown';
  }
};

const normalizeStatus = (state: string): 'online' | 'offline' | 'restarting' => {
  switch (state) {
    case 'running':
      return 'online';
    case 'restarting':
    case 'paused':
      return 'restarting';
    case 'exited':
    case 'dead':
    default:
      return 'offline';
  }
};

function calculateUptime(startedAt: string | undefined): string {
  if (!startedAt) return '-';
  const start = new Date(startedAt).getTime();
  const now = Date.now();
  const uptimeSeconds = Math.floor((now - start) / 1000);
  if (uptimeSeconds < 0) return '0s';

  const days = Math.floor(uptimeSeconds / 86400);
  const hours = Math.floor((uptimeSeconds % 86400) / 3600);
  const minutes = Math.floor((uptimeSeconds % 3600) / 60);
  const seconds = Math.floor(uptimeSeconds % 60);

  const parts: string[] = [];
  if (days > 0) parts.push(`${days}d`);
  if (hours > 0 || days > 0) parts.push(`${hours}h`);
  if (minutes > 0 || hours > 0 || days > 0) parts.push(`${minutes}m`);
  if (seconds > 0 || uptimeSeconds < 60) parts.push(`${seconds}s`);

  return parts.join(' ');
}

const InfrastructureDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const {
    loading: storeLoading,
    getContainerById,
    fetchInfrastructure,
    restartContainer,
    stopContainer,
    startContainer,
    connectSSE,
    disconnectSSE
  } = useInfrastructureStore();

  const [activeTab, setActiveTab] = useState<string | number>('overview');
  const [stats, setStats] = useState<ServerStats | null>(null);
  const [isInitialLoading, setIsInitialLoading] = useState(true);
  const [isRestarting, setIsRestarting] = useState(false);
  const [isPowerActionLoading, setIsPowerActionLoading] = useState(false);
  const [inspectData, setInspectData] = useState<{ State?: { StartedAt?: string } } | null>(null);
  const statsEventSourceRef = useRef<EventSource | null>(null);

  const container = getContainerById(id || '');
  const containerId = id || (container ? getContainerId(container.summary) : '');

  // Ensure we have container data (fetch if not in store yet)
  useEffect(() => {
    if (id && !container) {
      fetchInfrastructure();
    }
  }, [id, container, fetchInfrastructure]);

  useEffect(() => {
    connectSSE();
    if (containerId) {
      statsEventSourceRef.current = new EventSource(
        api.withSSEAuth(`${API_BASE_URL}/infrastructure/${containerId}/stats`)
      );
      statsEventSourceRef.current.onmessage = (event: MessageEvent) => {
        try {
          const parsed = JSON.parse(event.data);
          setStats(parsed);
          setIsInitialLoading(false);
        } catch {
          setIsInitialLoading(false);
        }
      };
      statsEventSourceRef.current.onerror = () => setIsInitialLoading(false);

      api
        .getInfrastructureContainer(containerId)
        .then((res) => setInspectData(res.data.inspectData))
        .catch(() => {});
    }

    return () => {
      disconnectSSE();
      if (statsEventSourceRef.current) {
        statsEventSourceRef.current.close();
        statsEventSourceRef.current = null;
      }
    };
  }, [containerId, connectSSE, disconnectSSE]);

  if (!container && id) {
    if (storeLoading) {
      return (
        <PageSection>
          <Button variant="secondary" icon={<ArrowLeftIcon />} onClick={() => navigate('/infrastructure')}>
            Back to Infrastructure
          </Button>
          <div className="pf-v6-u-mt-lg" style={{ display: 'flex', justifyContent: 'center', padding: '3rem' }}>
            <Spinner size="xl" />
          </div>
        </PageSection>
      );
    }
    return (
      <PageSection>
        <Button variant="secondary" icon={<ArrowLeftIcon />} onClick={() => navigate('/infrastructure')}>
          Back to Infrastructure
        </Button>
        <EmptyState
          variant={EmptyStateVariant.lg}
          titleText="Container not found"
          className="pf-v6-u-mt-lg"
        >
          <EmptyStateBody>
            The requested infrastructure container could not be found. It may have been removed or the ID may be invalid.
          </EmptyStateBody>
        </EmptyState>
      </PageSection>
    );
  }

  if (!id) {
    return (
      <PageSection>
        <Button variant="secondary" icon={<ArrowLeftIcon />} onClick={() => navigate('/infrastructure')}>
          Back to Infrastructure
        </Button>
        <EmptyState
          variant={EmptyStateVariant.lg}
          titleText="No container selected"
          className="pf-v6-u-mt-lg"
        >
          <EmptyStateBody>Select an infrastructure container from the list to view its details.</EmptyStateBody>
        </EmptyState>
      </PageSection>
    );
  }

  const containerName = container.summary.Names?.[0]?.replace(/^\//, '') || 'Unnamed';
  const ports =
    container.summary.Ports && container.summary.Ports.length > 0
      ? container.summary.Ports.map(
          (p: { PublicPort?: number; PrivatePort: number }) =>
            p.PublicPort ? `${p.PublicPort}:${p.PrivatePort}` : String(p.PrivatePort)
        ).join(', ')
      : '-';
  const status = normalizeStatus(container.summary.State);
  const isRunning = container.summary.State === 'running';
  const uptime = calculateUptime(inspectData?.State?.StartedAt);

  const handleRestart = async () => {
    setIsRestarting(true);
    try {
      await restartContainer(containerId);
    } finally {
      setTimeout(() => setIsRestarting(false), 3000);
    }
  };

  const handlePowerAction = async () => {
    setIsPowerActionLoading(true);
    try {
      if (isRunning) {
        await stopContainer(containerId);
      } else {
        await startContainer(containerId);
      }
    } finally {
      setTimeout(() => setIsPowerActionLoading(false), 1000);
    }
  };

  return (
    <>
      <PageHeader
        title={containerName}
        description={`Infrastructure container details for ${containerName}`}
        actions={
          <>
            <Button
              variant="secondary"
              icon={<ArrowLeftIcon />}
              onClick={() => navigate('/infrastructure')}
            >
              Back to Infrastructure
            </Button>
            <Button
              variant="secondary"
              icon={<EditIcon />}
              onClick={() => {}}
              isDisabled={isRestarting || isPowerActionLoading}
            >
              Edit
            </Button>
            {isRunning && (
              <Button
                variant="secondary"
                icon={
                  <SyncAltIcon
                    className={isRestarting ? 'pf-v6-u-animation-spin' : ''}
                  />
                }
                onClick={handleRestart}
                isDisabled={isRestarting || isPowerActionLoading}
              >
                {isRestarting ? 'Restarting...' : 'Restart'}
              </Button>
            )}
            <Button
              variant={isRunning ? 'danger' : 'success'}
              icon={isPowerActionLoading ? <Spinner size="md" /> : <PowerOffIcon />}
              onClick={handlePowerAction}
              isDisabled={isRestarting || isPowerActionLoading}
            >
              {isPowerActionLoading
                ? isRunning
                  ? 'Stopping...'
                  : 'Starting...'
                : isRunning
                  ? 'Stop'
                  : 'Start'}
            </Button>
          </>
        }
      />

      <PageSection>
        <Grid hasGutter>
          {/* Status and Uptime cards */}
          <Grid hasGutter className="pf-v6-u-mb-lg">
            <GridItem span={12} md={6} lg={3}>
              <Card isCompact>
                <CardBody>
                  <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    <FlexItem spacer={{ default: 'spacerSm' }}>
                      <ChartLineIcon
                        style={{
                          fontSize: '20px',
                          color: 'var(--pf-v6-global--primary-color--100)'
                        }}
                      />
                    </FlexItem>
                    <FlexItem>
                      <div>
                        <div className="pf-v6-u-font-size-sm">Status</div>
                        <div className="pf-v6-u-mt-xs">
                          <StatusBadge status={status} />
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
                  <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    <FlexItem spacer={{ default: 'spacerSm' }}>
                      <ClockIcon
                        style={{
                          fontSize: '20px',
                          color: 'var(--pf-v6-global--success-color--100)'
                        }}
                      />
                    </FlexItem>
                    <FlexItem>
                      <div>
                        <div className="pf-v6-u-font-size-sm">Uptime</div>
                        <div className="pf-v6-u-font-size-xl pf-v6-u-font-weight-bold pf-v6-u-mt-xs">
                          {uptime}
                        </div>
                      </div>
                    </FlexItem>
                  </Flex>
                </CardBody>
              </Card>
            </GridItem>
          </Grid>

          {/* Tabs */}
          <Tabs activeKey={activeTab} onSelect={(_event, tabIndex) => setActiveTab(tabIndex)} isBox>
            <Tab
              eventKey="overview"
              title={
                <>
                  <TabTitleIcon>
                    <ChartLineIcon />
                  </TabTitleIcon>
                  <TabTitleText>Overview</TabTitleText>
                </>
              }
            >
              <InfrastructureOverviewTab
                containerId={containerId}
                containerName={containerName}
                containerType={container.type}
                image={container.summary.Image}
                ports={ports}
                stats={stats}
                isInitialLoading={isInitialLoading}
              />
            </Tab>

            <Tab
              eventKey="console"
              title={
                <>
                  <TabTitleIcon>
                    <TerminalIcon />
                  </TabTitleIcon>
                  <TabTitleText>Console</TabTitleText>
                </>
              }
            >
              <ConsoleTab
                containerId={containerId}
                logsUrl={api.withSSEAuth(`${API_BASE_URL}/infrastructure/${containerId}/logs`)}
                isActive={activeTab === 'console'}
                enableSendCommand={false}
              />
            </Tab>
          </Tabs>
        </Grid>
      </PageSection>
    </>
  );
};

export default InfrastructureDetail;
