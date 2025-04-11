import React, { useEffect } from 'react';
import { useParams } from 'react-router';
import { Subscription, WsCommand, WsCommandType } from '@app/model/wsCommand';
import { CpuStats, PrecpuStats, ServerStats } from '@app/model/serverstats';
import { useSelector } from 'react-redux';
import { RootState } from '@app/store/store';
import { Card, CardBody, CardTitle, Flex, FlexItem, Skeleton } from '@patternfly/react-core';
import { registerSubscriptions, useSharedWebSocket } from '@app/connection/WebSocketContext';

export const ServerDetailsStats: React.FC = () => {
  const { serverId } = useParams<{ serverId: string }>();
  const { sendMessage } = useSharedWebSocket();
  const serverStats: ServerStats | undefined = useSelector((state: RootState) => state.server.serverStats);

  useEffect(() => {
    registerSubscriptions(sendMessage, [Subscription.SUB_STATS], serverId);
  }, [serverId]);

  useEffect(() => {
    return () => {
      if (serverId) {
        unsubscribeFromStats(serverId);
      }
    };
  }, [serverId]);


  const bytesToGb = (b: number) => {
    return Number(b / (Math.pow(1024, 3))).toFixed(2);
  };

  const bytesToMb = (b: number) => {
    return Number(b / (Math.pow(1024, 2))).toFixed(2);
  };

  const calcCPUPercent = (cpu: CpuStats | undefined, pre_cpu: PrecpuStats | undefined) => {
    if (!cpu || !pre_cpu) return 0;

    const delta_container_cpu = cpu.cpu_usage.total_usage - pre_cpu.cpu_usage.total_usage;
    const delta_system_cpu = cpu.system_cpu_usage - pre_cpu.system_cpu_usage;
    const online_cpus = cpu.online_cpus;

    return Number((delta_container_cpu / delta_system_cpu) * online_cpus * 100).toFixed(2);
  };

  const unsubscribeFromStats = (containerId: string) => {
    const commandMessage: WsCommand = {
      type: WsCommandType.UNSUBSCRIBE_CONTAINER_STATS,
      containerId: containerId
    };
    sendMessage(JSON.stringify(commandMessage));
  };

  return (
    <>
      <Flex>
        <FlexItem grow={{ default: 'grow' }}>
          <Card ouiaId="MemoryCard" style={{ height: '100%' }}>
            <CardTitle>Memory</CardTitle>
            <CardBody>
              {serverStats ?
                <>
                  <p>Current Usage: {bytesToGb(serverStats?.memory_stats.usage ?? 0)} GB</p>
                  <p>Limit: {bytesToGb(serverStats?.memory_stats.limit ?? 0)} GB</p>
                </> :
                <Skeleton width="25%" screenreaderText="Loading contents" />}
            </CardBody>
          </Card>
        </FlexItem>
        <FlexItem grow={{ default: 'grow' }}>
          <Card ouiaId="CPUCard" style={{ height: '100%' }}>
            <CardTitle>CPU</CardTitle>
            <CardBody>
              {serverStats ?
                <p>{calcCPUPercent(serverStats?.cpu_stats, serverStats?.precpu_stats)}%</p> :
                <Skeleton width="25%" screenreaderText="Loading contents" />}
              <p>&nbsp;</p>
            </CardBody>
          </Card>
        </FlexItem>
      </Flex>
    </>
  );
};
