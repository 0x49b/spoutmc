import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router';
import { CpuStats, PrecpuStats, ServerStats } from '@app/model/serverstats';
import { useDispatch, useSelector } from 'react-redux';
import { RootState } from '@app/store/store';
import { Card, CardBody, CardTitle, Flex, FlexItem, Skeleton } from '@patternfly/react-core';
import { useMqtt } from '@app/connection/MqttContext';
import { setServerStats } from '@app/store/serverSlice';

export const ServerDetailsStats: React.FC = () => {
  const { serverId } = useParams<{ serverId: string }>();
  //const { sendMessage, readyState } = useSharedWebSocket();
  const { subscribe, publish, isConnected, unsubscribe } = useMqtt();
  const dispatch = useDispatch();
  const [topic, setTopic] = useState('');
  const serverStats: ServerStats | undefined = useSelector((state: RootState) => state.server.serverStats);


  /*useEffect(() => {
    if (readyState === ReadyState.OPEN) {
      registerSubscriptions(sendMessage, [Subscription.SUB_STATS], serverId);
    }
  }, [readyState, serverId]);*/


  useEffect(() => {
    if (!isConnected) return;

    setTopic(`server/${serverId}/stats`);

    const handleMsg = (msg: string) => {
      const messageJSON: ServerStats = JSON.parse(msg);
      dispatch(setServerStats(messageJSON));
    };

    subscribe(topic, handleMsg);
  }, [isConnected]);

  useEffect(() => {
    const handleUnload = () => {
      if (isConnected) {
        unsubscribe(topic);
        setTopic('');
      }
    };

    window.addEventListener('beforeunload', handleUnload);
    return () => {
      window.removeEventListener('beforeunload', handleUnload);
    };
  }, [isConnected]);


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
