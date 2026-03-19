import { useEffect, useState } from 'react';
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
import { CpuIcon } from '@patternfly/react-icons';
import * as api from '../../service/apiService';
import { ServerStats } from '../../model/ServerStats';
import { redactEnvValue } from '../../utils/redactSensitiveEnv';
import OverviewTabSkeleton from '../Servers/ServerDetailTabs/OverviewTabSkeleton';

interface MemoryUsage {
  usedMemory: string;
  maxMemory: string;
  usagePercent: number;
}

interface InfrastructureOverviewTabProps {
  containerId: string;
  containerName: string;
  containerType: string;
  image: string;
  ports: string;
  stats: ServerStats | null;
  isInitialLoading: boolean;
}

function calculateCPUPercentage(previous: any, current: any): number {
  if (!previous || !current) return 0;
  const cpuUsage = current.cpu_usage || current.CPUUsage;
  const preCpuUsage = previous.cpu_usage || previous.CPUUsage;
  if (!cpuUsage || !preCpuUsage) return 0;

  const cpuDelta = Number(cpuUsage.total_usage ?? 0) - Number(preCpuUsage.total_usage ?? 0);
  const systemDelta =
    Number(current.system_cpu_usage ?? 0) - Number(previous.system_cpu_usage ?? 0);
  const numberCpus = Number(current.online_cpus ?? 1) || 1;

  if (systemDelta <= 0 || cpuDelta < 0) return 0;
  const cpuPercent = (cpuDelta / systemDelta) * numberCpus * 100;
  const result = Math.min(Number(cpuPercent.toFixed(2)), 100);
  return Number.isFinite(result) ? result : 0;
}

function formatBytes(bytes: number | undefined | null): string {
  if (bytes === undefined || bytes === null || isNaN(bytes)) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let i = 0;
  let value = bytes;
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024;
    i++;
  }
  return `${value.toFixed(2)} ${units[i]}`;
}

function getMemoryUsageInfo(memoryStats: any): MemoryUsage {
  if (!memoryStats || !memoryStats.usage || !memoryStats.limit) {
    return { usedMemory: '0 B', maxMemory: '0 B', usagePercent: 0 };
  }
  const used = memoryStats.usage;
  const max = memoryStats.limit;
  const usagePercent = max > 0 ? parseFloat(((used / max) * 100).toFixed(2)) : 0;
  return {
    usedMemory: formatBytes(used),
    maxMemory: formatBytes(max),
    usagePercent
  };
}

function parseEnvFromInspect(env: string[] | undefined): Record<string, string> {
  const result: Record<string, string> = {};
  if (!env || !Array.isArray(env)) return result;
  for (const item of env) {
    const eq = item.indexOf('=');
    if (eq > 0) {
      result[item.slice(0, eq)] = item.slice(eq + 1);
    }
  }
  return result;
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

const getTypeLabel = (type: string) => {
  switch (type) {
    case 'database':
      return 'Database';
    default:
      return type || 'Unknown';
  }
};

export const InfrastructureOverviewTab = ({
  containerId,
  containerName,
  containerType,
  image,
  ports,
  stats,
  isInitialLoading
}: InfrastructureOverviewTabProps) => {
  const [cpuUsage, setCpuUsage] = useState(0);
  const [memoryUsage, setMemoryUsage] = useState<MemoryUsage>({
    usedMemory: '0',
    maxMemory: '0',
    usagePercent: 0
  });
  const [envVars, setEnvVars] = useState<Record<string, string>>({});
  const [isLoadingEnv, setIsLoadingEnv] = useState(false);

  useEffect(() => {
    if (stats) {
      setCpuUsage(calculateCPUPercentage(stats.precpu_stats, stats.cpu_stats));
      setMemoryUsage(getMemoryUsageInfo(stats.memory_stats));
    }
  }, [stats]);

  useEffect(() => {
    if (containerId) {
      setIsLoadingEnv(true);
      api
        .getInfrastructureContainer(containerId)
        .then((res) => {
          const env = res.data.inspectData?.Config?.Env;
          setEnvVars(parseEnvFromInspect(env));
        })
        .catch((err) => console.error('Failed to load container details:', err))
        .finally(() => setIsLoadingEnv(false));
    }
  }, [containerId]);

  if (isInitialLoading) {
    return <OverviewTabSkeleton />;
  }

  return (
      <Grid hasGutter className="pf-v6-u-mb-lg">
        <GridItem span={12} md={6}>
          <Title headingLevel="h3" size="lg" className="pf-v6-u-mb-md">
            Container Information
          </Title>
          <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: 'var(--pf-v6-global--spacer--sm)'
              }}
          >
            <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
              <FlexItem className="pf-v6-u-color-200">Name:</FlexItem>
              <FlexItem>
                <strong>{containerName}</strong>
              </FlexItem>
            </Flex>
            <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
              <FlexItem className="pf-v6-u-color-200">Type:</FlexItem>
              <FlexItem>
                <strong>{getTypeLabel(containerType)}</strong>
              </FlexItem>
            </Flex>
            <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
              <FlexItem className="pf-v6-u-color-200">Image:</FlexItem>
              <FlexItem>
                <strong>{image}</strong>
              </FlexItem>
            </Flex>
            <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
              <FlexItem className="pf-v6-u-color-200">Ports:</FlexItem>
              <FlexItem>
                <strong>{ports || '-'}</strong>
              </FlexItem>
            </Flex>
          </div>
        </GridItem>

        <GridItem span={12} md={6}>
          <Title headingLevel="h3" size="lg" className="pf-v6-u-mb-md">
            Resource Usage
          </Title>
          <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: 'var(--pf-v6-global--spacer--md)'
              }}
          >
            <div>
              <Flex
                  justifyContent={{default: 'justifyContentSpaceBetween'}}
                  className="pf-v6-u-mb-sm"
              >
                <FlexItem>
                  <CpuIcon/> CPU Usage
                </FlexItem>
                <FlexItem>
                  <strong>{cpuUsage}%</strong>
                </FlexItem>
              </Flex>
              <Progress
                  value={Math.min(cpuUsage, 100)}
                  variant={getCPUVariant(cpuUsage)}
                  measureLocation={ProgressMeasureLocation.none}
                  aria-label="CPU usage"
              />
            </div>

            <div>
              <Flex
                  justifyContent={{default: 'justifyContentSpaceBetween'}}
                  className="pf-v6-u-mb-sm"
              >
                <FlexItem>Memory Usage</FlexItem>
                <FlexItem>
                  <strong>
                    {memoryUsage?.usedMemory} / {memoryUsage?.maxMemory}
                  </strong>
                </FlexItem>
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
          <Title headingLevel="h3" size="lg" className="pf-v6-u-mb-md">
            Environment Variables
          </Title>
          {isLoadingEnv ? (
              <div className="pf-v6-u-color-200">Loading environment variables...</div>
          ) : (
              <div
                  style={{
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 'var(--pf-v6-global--spacer--sm)',
                    maxHeight: '400px',
                    overflowY: 'auto',
                    padding: 'var(--pf-v6-global--spacer--md)',
                    backgroundColor: 'var(--pf-v6-global--BackgroundColor--dark-100)',
                    borderRadius: 'var(--pf-v6-global--BorderRadius--sm)',
                    border: '1px solid var(--pf-v6-global--BorderColor--100)'
                  }}
              >
                {Object.keys(envVars).length === 0 ? (
                    <div className="pf-v6-u-color-200">No environment variables configured</div>
                ) : (
                    Object.entries(envVars).map(([key, value]) => (
                        <Flex
                            key={key}
                            justifyContent={{default: 'justifyContentSpaceBetween'}}
                            style={{
                              fontFamily: 'monospace',
                              fontSize: '0.9em',
                              padding: 'var(--pf-v6-global--spacer--xs) var(--pf-v6-global--spacer--sm)',
                              backgroundColor: 'var(--pf-v6-global--BackgroundColor--dark-200)',
                              borderRadius: 'var(--pf-v6-global--BorderRadius--sm)'
                            }}
                        >
                          <FlexItem className="pf-v6-u-color-200" style={{fontWeight: 600}}>
                            {key}:
                          </FlexItem>
                          <FlexItem style={{wordBreak: 'break-all', textAlign: 'right'}}>
                            {redactEnvValue(key, value)}
                          </FlexItem>
                        </Flex>
                    ))
                )}
              </div>
          )}
        </GridItem>
      </Grid>
  );
};

export default InfrastructureOverviewTab;
