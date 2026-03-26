import React from 'react';
import {
  Button,
  Card,
  CardBody,
  CardHeader,
  Checkbox,
  DatePicker,
  FormGroup,
  FormSelect,
  FormSelectOption,
  Grid,
  GridItem,
  TextInput,
  Title,
  TimePicker
} from '@patternfly/react-core';

import type {BanDurationOptionDTO, PlayerBanHistoryDTO, PlayerKickHistoryDTO, PlayerSummaryDTO} from '../../service/apiService';

interface PlayerDetailModerationTabProps {
  activeBan: PlayerSummaryDTO | null;

  banDurations: BanDurationOptionDTO[];
  banReason: string;
  banPermanent: boolean;
  banDurationKey: string;
  useCustomUntil: boolean;
  customDate: string;
  customTime: string;
  kickReason: string;

  bansHistory: PlayerBanHistoryDTO[];
  kicksHistory: PlayerKickHistoryDTO[];

  onBanReasonChange: (next: string) => void;
  onBanPermanentChange: (next: boolean) => void;
  onBanDurationKeyChange: (next: string) => void;
  onUseCustomUntilChange: (next: boolean) => void;
  onCustomDateChange: (next: string) => void;
  onCustomTimeChange: (next: string) => void;
  onKickReasonChange: (next: string) => void;

  onRequestConfirmBanOpen: () => void;
  onRequestConfirmUnbanOpen: () => void;
  onKick: () => Promise<void>;
}

const PlayerDetailModerationTab: React.FC<PlayerDetailModerationTabProps> = ({
  activeBan,
  banDurations,
  banReason,
  banPermanent,
  banDurationKey,
  useCustomUntil,
  customDate,
  customTime,
  kickReason,
  bansHistory,
  kicksHistory,
  onBanReasonChange,
  onBanPermanentChange,
  onBanDurationKeyChange,
  onUseCustomUntilChange,
  onCustomDateChange,
  onCustomTimeChange,
  onKickReasonChange,
  onRequestConfirmBanOpen,
  onRequestConfirmUnbanOpen,
  onKick
}) => {
  return (
    <Grid hasGutter className="pf-v6-u-mt-md">
      <GridItem span={6}>
        <Card isCompact>
          <CardHeader>
            <Title headingLevel="h2" size="lg">Ban / Kick</Title>
          </CardHeader>
          <CardBody>
            <FormGroup label="Ban reason" fieldId="ban-reason">
              <TextInput id="ban-reason" value={banReason} onChange={(_ev, value) => onBanReasonChange(value)} />
            </FormGroup>

            <Checkbox
              label="Permanent ban"
              id="ban-permanent"
              isChecked={banPermanent}
              onChange={(_ev, checked) => onBanPermanentChange(checked)}
            />

            <div style={{height: 12}} />

            <FormGroup label="Predefined duration" fieldId="ban-duration">
              <FormSelect
                value={banDurationKey}
                isDisabled={banPermanent}
                onChange={(_ev, value) => onBanDurationKeyChange(value as string)}
              >
                {banDurations.map((opt) => (
                  <FormSelectOption key={opt.key} value={opt.key} label={opt.label} />
                ))}
              </FormSelect>
            </FormGroup>

            <div style={{height: 12}} />

            <Checkbox
              label="Custom until"
              id="ban-custom-until"
              isDisabled={banPermanent}
              isChecked={useCustomUntil}
              onChange={(_ev, checked) => onUseCustomUntilChange(checked)}
            />

            {useCustomUntil && !banPermanent ? (
              <div style={{marginTop: 12}}>
                <FormGroup label="Until date" fieldId="ban-until-date">
                  <DatePicker
                    aria-label="Until date"
                    value={customDate}
                    placeholder="YYYY-MM-DD"
                    onChange={(_ev, value) => onCustomDateChange(value)}
                  />
                </FormGroup>
                <FormGroup label="Until time" fieldId="ban-until-time">
                  <TimePicker
                    aria-label="Until time"
                    time={customTime}
                    placeholder="hh:mm"
                    onChange={(_ev, value) => onCustomTimeChange(value)}
                  />
                </FormGroup>
              </div>
            ) : null}

            <div style={{marginTop: 12, display: 'flex', gap: 8}}>
              <Button
                variant="danger"
                onClick={onRequestConfirmBanOpen}
                isDisabled={!banReason.trim()}
              >
                Ban
              </Button>
              {activeBan ? (
                <Button variant="secondary" onClick={onRequestConfirmUnbanOpen}>
                  Unban
                </Button>
              ) : null}
            </div>

            <div style={{height: 20}} />

            <div>
              <Title headingLevel="h3" size="md" style={{marginBottom: 8}}>Kick</Title>
              <FormGroup label="Kick reason" fieldId="kick-reason">
                <TextInput
                  id="kick-reason"
                  value={kickReason}
                  onChange={(_ev, value) => onKickReasonChange(value)}
                />
              </FormGroup>
              <Button
                variant="warning"
                onClick={() => void onKick()}
                isDisabled={!kickReason.trim()}
              >
                Kick
              </Button>
            </div>
          </CardBody>
        </Card>
      </GridItem>

      <GridItem span={6}>
        <Card isCompact>
          <CardHeader>
            <Title headingLevel="h2" size="lg">History</Title>
          </CardHeader>
          <CardBody>
            <div>
              <Title headingLevel="h3" size="md">Bans</Title>
              <div className="playerDetail__historyList">
                {bansHistory.length ? bansHistory.map((b, i) => (
                  <div key={`${b.staffUserId}-${i}`} className="playerDetail__historyRow">
                    <div className="playerDetail__historyMain">
                      <strong>{b.staffDisplayName}</strong>: {b.reason}
                    </div>
                    <div className="playerDetail__historySub">
                      {b.permanent ? 'Permanent' : `Until: ${b.untilAt ?? '-'}`} {b.liftedAt ? `(lifted: ${new Date(b.liftedAt).toLocaleString()})` : ''}
                    </div>
                  </div>
                )) : <div style={{opacity: 0.8}}>No bans yet.</div>}
              </div>
            </div>

            <div style={{height: 12}} />

            <div>
              <Title headingLevel="h3" size="md">Kicks</Title>
              <div className="playerDetail__historyList">
                {kicksHistory.length ? kicksHistory.map((k, i) => (
                  <div key={`${k.staffUserId}-${i}`} className="playerDetail__historyRow">
                    <div className="playerDetail__historyMain">
                      <strong>{k.staffDisplayName}</strong>: {k.reason}
                    </div>
                    <div className="playerDetail__historySub">
                      {new Date(k.occurredAt).toLocaleString()}
                    </div>
                  </div>
                )) : <div style={{opacity: 0.8}}>No kicks yet.</div>}
              </div>
            </div>
          </CardBody>
        </Card>
      </GridItem>
    </Grid>
  );
};

export default PlayerDetailModerationTab;

