import {useEffect, useState} from 'react';
import {
    Flex,
    FlexItem,
    Grid,
    GridItem,
    Progress,
    ProgressMeasureLocation,
    ProgressVariant,
    Title
} from '@patternfly/react-core';
import {CpuIcon} from '@patternfly/react-icons';
import {ServerStats} from '../../../model/ServerStats.ts';
import {Server} from '../../../types';
import OverviewTabSkeleton from './OverviewTabSkeleton';
import RestartConfirmationModal from '../RestartConfirmationModal';
import * as api from '../../../service/apiService';
import {useServerStore} from '../../../store/serverStore';

interface MemoryUsage {
    usedMemory: string,
    maxMemory: string,
    usagePercent: number
}

interface OverviewTabProps {
    server: Server;
    stats: ServerStats | null;
    isInitialLoading: boolean;
}

export const OverviewTab = ({
                                server,
                                stats,
                                isInitialLoading
                            }: OverviewTabProps) => {
    const {restartServer} = useServerStore();
    const [cpuUsage, setCpuUsage] = useState(0);
    const [memoryUsage, setMemoryUsage] = useState<MemoryUsage>({
        usedMemory: '0',
        maxMemory: '0',
        usagePercent: 0
    });
    const [configFiles, setConfigFiles] = useState<string[]>([]);
    const [isLoadingFiles, setIsLoadingFiles] = useState(false);
    const [isEditorOpen, setIsEditorOpen] = useState(false);
    const [isRestartModalOpen, setIsRestartModalOpen] = useState(false);
    const [selectedFile, setSelectedFile] = useState<string>('');
    const [isRestarting, setIsRestarting] = useState(false);
    const [envVars, setEnvVars] = useState<Record<string, string>>({});
    const [isLoadingEnv, setIsLoadingEnv] = useState(false);

    useEffect(() => {
        if (stats) {
            setCpuUsage(calculateCPUPercentage(stats.precpu_stats, stats.cpu_stats));
            setMemoryUsage(getMemoryUsageInfo(stats.memory_stats));
        }
    }, [stats]);

    useEffect(() => {
        if (server && server.id) {
            loadConfigFiles();
            loadEnvironmentVariables();
        }
    }, [server?.id]);

    const loadConfigFiles = async () => {
        setIsLoadingFiles(true);
        try {
            const response = await api.listConfigFiles(server.id);
            setConfigFiles(response.data.files);
        } catch (err) {
            console.error('Failed to load config files:', err);
        } finally {
            setIsLoadingFiles(false);
        }
    };

    const loadEnvironmentVariables = async () => {
        setIsLoadingEnv(true);
        try {
            const response = await api.getServerEnv(server.id);
            setEnvVars(response.data);
        } catch (err) {
            console.error('Failed to load environment variables:', err);
        } finally {
            setIsLoadingEnv(false);
        }
    };

    const handleOpenFile = (filename: string) => {
        setSelectedFile(filename);
        setIsEditorOpen(true);
    };

    const handleSaveSuccess = () => {
        setIsEditorOpen(false);
        setIsRestartModalOpen(true);
    };

    const handleRestartNow = async () => {
        setIsRestarting(true);
        try {
            await restartServer(server.id);
            setTimeout(() => {
                setIsRestarting(false);
                setIsRestartModalOpen(false);
            }, 2000);
        } catch (err) {
            console.error('Failed to restart server:', err);
            setIsRestarting(false);
        }
    };

    function calculateCPUPercentage(previous: any, current: any): number {
        if (!previous || !current) return 0;

        const cpuDelta = current.cpu_usage.total_usage - previous.cpu_usage.total_usage;
        const systemDelta = current.system_cpu_usage - previous.system_cpu_usage;

        if (systemDelta <= 0 || cpuDelta <= 0) {
            return 0;
        }

        const cpuPercent = (cpuDelta / systemDelta) * current.online_cpus * 100;
        return Math.min(Number(cpuPercent.toFixed(2)), 100);
    }

    function formatBytes(bytes: number | undefined | null): string {
        if (bytes === undefined || bytes === null || isNaN(bytes)) {
            return '0 B';
        }

        const units = ['B', 'KB', 'MB', 'GB', 'TB'];
        let i = 0;
        let value = bytes;

        while (value >= 1024 && i < units.length - 1) {
            value /= 1024;
            i++;
        }

        return `${value.toFixed(2)} ${units[i]}`;
    }

    function getMemoryUsageInfo(memory_stats: any): MemoryUsage {
        if (!memory_stats || !memory_stats.usage || !memory_stats.limit) {
            return {
                usedMemory: '0 B',
                maxMemory: '0 B',
                usagePercent: 0
            };
        }

        const used = memory_stats.usage;
        const max = memory_stats.limit;

        const usedFormatted = formatBytes(used);
        const maxFormatted = formatBytes(max);

        const usagePercent = max > 0 ? parseFloat(((used / max) * 100).toFixed(2)) : 0;

        return {
            usedMemory: usedFormatted,
            maxMemory: maxFormatted,
            usagePercent
        };
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

    if (isInitialLoading) {
        return <OverviewTabSkeleton/>;
    }

    return (
        <>
            <Grid hasGutter className="pf-v6-u-mb-lg">
                <GridItem span={12} md={6}>
                    <Title headingLevel="h3" size="lg" className="pf-v6-u-mb-md">Server
                        Information</Title>
                    <div style={{
                        display: 'flex',
                        flexDirection: 'column',
                        gap: 'var(--pf-v6-global--spacer--sm)'
                    }}>
                        <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
                            <FlexItem className="pf-v6-u-color-200">Version:</FlexItem>
                            <FlexItem><strong>{server.version}</strong></FlexItem>
                        </Flex>
                        <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
                            <FlexItem className="pf-v6-u-color-200">Address:</FlexItem>
                            <FlexItem><strong>{server.ip}:{server.port}</strong></FlexItem>
                        </Flex>
                        <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
                            <FlexItem className="pf-v6-u-color-200">Location:</FlexItem>
                            <FlexItem><strong>{server.location || 'Not specified'}</strong></FlexItem>
                        </Flex>
                    </div>
                </GridItem>

                <GridItem span={12} md={6}>
                    <Title headingLevel="h3" size="lg" className="pf-v6-u-mb-md">Resource
                        Usage</Title>
                    <div style={{
                        display: 'flex',
                        flexDirection: 'column',
                        gap: 'var(--pf-v6-global--spacer--md)'
                    }}>
                        <div>
                            <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}
                                  className="pf-v6-u-mb-sm">
                                <FlexItem>
                                    <CpuIcon/> CPU Usage
                                </FlexItem>
                                <FlexItem><strong>{cpuUsage}%</strong></FlexItem>
                            </Flex>
                            <Progress
                                value={Math.min(cpuUsage, 100)}
                                variant={getCPUVariant(cpuUsage)}
                                measureLocation={ProgressMeasureLocation.none}
                                aria-label="CPU usage"
                            />
                        </div>

                        <div>
                            <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}
                                  className="pf-v6-u-mb-sm">
                                <FlexItem>Memory Usage</FlexItem>
                                <FlexItem><strong>{memoryUsage?.usedMemory} / {memoryUsage?.maxMemory}</strong></FlexItem>
                            </Flex>
                            <Progress
                                value={Math.min(memoryUsage?.usagePercent || 0, 100)}
                                variant={getMemoryVariant(memoryUsage?.usagePercent || 0)}
                                measureLocation={ProgressMeasureLocation.none}
                                aria-label="Memory usage"
                            />
                        </div>
                    </div>
                </GridItem>

                <GridItem span={12}>
                    <Title headingLevel="h3" size="lg" className="pf-v6-u-mb-md">Environment Variables</Title>
                    {isLoadingEnv ? (
                        <div className="pf-v6-u-color-200">Loading environment variables...</div>
                    ) : (
                        <div style={{
                            display: 'flex',
                            flexDirection: 'column',
                            gap: 'var(--pf-v6-global--spacer--sm)',
                            maxHeight: '400px',
                            overflowY: 'auto',
                            padding: 'var(--pf-v6-global--spacer--md)',
                            backgroundColor: 'var(--pf-v6-global--BackgroundColor--dark-100)',
                            borderRadius: 'var(--pf-v6-global--BorderRadius--sm)',
                            border: '1px solid var(--pf-v6-global--BorderColor--100)'
                        }}>
                            {Object.keys(envVars).length === 0 ? (
                                <div className="pf-v6-u-color-200">No environment variables configured</div>
                            ) : (
                                Object.entries(envVars).map(([key, value]) => (
                                    <Flex key={key} justifyContent={{default: 'justifyContentSpaceBetween'}}
                                          style={{
                                              fontFamily: 'monospace',
                                              fontSize: '0.9em',
                                              padding: 'var(--pf-v6-global--spacer--xs) var(--pf-v6-global--spacer--sm)',
                                              backgroundColor: 'var(--pf-v6-global--BackgroundColor--dark-200)',
                                              borderRadius: 'var(--pf-v6-global--BorderRadius--sm)'
                                          }}>
                                        <FlexItem className="pf-v6-u-color-200" style={{fontWeight: 600}}>
                                            {key}:
                                        </FlexItem>
                                        <FlexItem style={{wordBreak: 'break-all', textAlign: 'right'}}>
                                            {value}
                                        </FlexItem>
                                    </Flex>
                                ))
                            )}
                        </div>
                    )}
                </GridItem>
            </Grid>
            <RestartConfirmationModal
                isOpen={isRestartModalOpen}
                onClose={() => setIsRestartModalOpen(false)}
                onRestartNow={handleRestartNow}
                serverName={server.name}
                isRestarting={isRestarting}
            />
        </>

    );
};

export default OverviewTab;
