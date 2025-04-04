import React, {useEffect} from 'react';
import {useParams} from "react-router";
import {useServerWebSocket} from "../../../services/websocketService";
import {WsCommand, WsCommandType} from "@app/model/wsCommand";
import {CpuStats, PrecpuStats, ServerStats} from "@app/model/serverstats";
import {useSelector} from "react-redux";
import {RootState} from "@app/store/store";
import {Card, CardBody, CardTitle, Flex, FlexItem} from "@patternfly/react-core";

export const ServerDetailsStats: React.FC = () => {
  const {serverId} = useParams<{ serverId: string }>();
  const {sendMessage} = useServerWebSocket();
  const serverStats: ServerStats | undefined = useSelector((state: RootState) => state.server.serverStats);

  useEffect(() => {
    loadServerStats();
  }, [serverId]);

  const loadServerStats = () => {
    const commandMessage: WsCommand = {
      type: WsCommandType.CONTAINERSTATS,
      containerId: serverId,
    };
    sendMessage(JSON.stringify(commandMessage));
  };

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
              <p>Current Usage: {bytesToGb(serverStats?.memory_stats.usage ?? 0)} GB</p>
              <p>Limit: {bytesToGb(serverStats?.memory_stats.limit ?? 0)} GB</p>
            </CardBody>
          </Card>
        </FlexItem>
        <FlexItem grow={{ default: 'grow' }}>
          <Card ouiaId="CPUCard" style={{ height: '100%' }}>
            <CardTitle>CPU</CardTitle>
            <CardBody>
              <p>{calcCPUPercent(serverStats?.cpu_stats, serverStats?.precpu_stats)}%</p>
              <p>&nbsp;</p>
            </CardBody>
          </Card>
        </FlexItem>
      </Flex>
    </>
  );
};
