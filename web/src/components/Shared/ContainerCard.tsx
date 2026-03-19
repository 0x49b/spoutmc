import React from 'react';
import {
  Card,
  CardBody,
  CardTitle,
  Divider,
  Flex,
  FlexItem,
  Progress,
  ProgressMeasureLocation,
  ProgressVariant
} from '@patternfly/react-core';
import StatusBadge from '../UI/StatusBadge';

export interface ContainerCardDetail {
  label: string;
  value: string;
}

export interface ContainerCardProps {
  id: string;
  name: string;
  status: 'online' | 'offline' | 'restarting';
  icon: React.ReactNode;
  typeLabel: string;
  details: ContainerCardDetail[];
  cpu: number;
  memory: number;
  onClick: () => void;
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

const safeNum = (n: number): number =>
  Number.isFinite(n) && !Number.isNaN(n) ? n : 0;

const ContainerCard: React.FC<ContainerCardProps> = ({
  name,
  status,
  icon,
  typeLabel,
  details,
  cpu,
  memory,
  onClick
}) => {
  const safeCpu = safeNum(cpu);
  const safeMemory = safeNum(memory);

  return (
    <Card
      onClick={onClick}
      isSelectable
      style={{
        border: '1px solid var(--pf-v6-global--BorderColor--100, #d2d2d2)',
        borderRadius: 'var(--pf-v6-global--BorderRadius--sm, 4px)'
      }}
    >
      <CardTitle>
        <Flex justifyContent={{default: 'justifyContentSpaceBetween'}} alignItems={{default: 'alignItemsFlexStart'}}>
          <FlexItem>
            {icon} <strong>{name}</strong>
          </FlexItem>
          <FlexItem>
            <StatusBadge status={status}/>
          </FlexItem>
        </Flex>
      </CardTitle>
      <CardBody>
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 'var(--pf-v6-global--spacer--sm)'
        }}>
          <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
            <FlexItem>Type:</FlexItem>
            <FlexItem><strong>{typeLabel}</strong></FlexItem>
          </Flex>
          {details.map(({label, value}) => (
            <Flex key={label} justifyContent={{default: 'justifyContentSpaceBetween'}}>
              <FlexItem>{label}:</FlexItem>
              <FlexItem><strong>{value}</strong></FlexItem>
            </Flex>
          ))}
        </div>

        <Divider className="pf-v6-u-my-md"/>

        <div style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 'var(--pf-v6-global--spacer--md)'
        }}>
          <div>
            <div className="pf-v6-u-mb-xs">
              <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
                <FlexItem className="pf-v6-u-font-size-sm pf-v6-u-color-200">
                  CPU Usage
                </FlexItem>
                <FlexItem className="pf-v6-u-font-size-sm">
                  <strong>{safeCpu}%</strong>
                </FlexItem>
              </Flex>
            </div>
            <Progress
              value={Math.min(safeCpu, 100)}
              variant={getCPUVariant(safeCpu)}
              measureLocation={ProgressMeasureLocation.none}
              aria-label="CPU usage"
            />
          </div>

          <div>
            <div className="pf-v6-u-mb-xs">
              <Flex justifyContent={{default: 'justifyContentSpaceBetween'}}>
                <FlexItem className="pf-v6-u-font-size-sm pf-v6-u-color-200">
                  Memory Usage
                </FlexItem>
                <FlexItem className="pf-v6-u-font-size-sm">
                  <strong>{safeMemory}%</strong>
                </FlexItem>
              </Flex>
            </div>
            <Progress
              value={Math.min(safeMemory, 100)}
              variant={getMemoryVariant(safeMemory)}
              measureLocation={ProgressMeasureLocation.none}
              aria-label="Memory usage"
            />
          </div>
        </div>
      </CardBody>
    </Card>
  );
};

export default ContainerCard;
