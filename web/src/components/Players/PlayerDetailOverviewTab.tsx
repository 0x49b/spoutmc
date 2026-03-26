import React from 'react';
import {Card, CardBody, CardHeader, EmptyState, EmptyStateVariant, Grid, GridItem, Title} from '@patternfly/react-core';

import type {PlayerSummaryDTO} from '../../service/apiService';

interface PlayerDetailOverviewTabProps {
  summary: PlayerSummaryDTO | null;
  effectivePlayerUuid: string;
  aliasList: string[];
}

const PlayerDetailOverviewTab: React.FC<PlayerDetailOverviewTabProps> = ({summary, effectivePlayerUuid, aliasList}) => {
  return (
    <Card className="pf-v6-u-mt-md">
      <CardBody>
        <Grid hasGutter>
          <GridItem span={4}>
            <Card>
              <CardHeader>
                <Title headingLevel="h2" size="md">Skin</Title>
              </CardHeader>
              <CardBody>
                {summary?.avatarDataUrl ? (
                  <img
                    src={summary.avatarDataUrl}
                    alt={`${summary.minecraftName || 'player'} skin`}
                    style={{width: '100%', maxWidth: 160, borderRadius: 10}}
                  />
                ) : (
                  <EmptyState variant={EmptyStateVariant.sm} titleText="No avatar available" />
                )}
              </CardBody>
            </Card>
          </GridItem>

          <GridItem span={8}>
            <Card>
              <CardHeader>
                <Title headingLevel="h2" size="md">Player</Title>
              </CardHeader>
              <CardBody>
                <div style={{display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12}}>
                  <div>
                    <div className="pf-v6-u-font-size-sm" style={{opacity: 0.8}}>Current username</div>
                    <div className="pf-v6-u-font-size-xl pf-v6-u-font-weight-bold pf-v6-u-mt-xs">
                      {summary?.minecraftName || effectivePlayerUuid}
                    </div>
                  </div>

                  <div>
                    <div className="pf-v6-u-font-size-sm" style={{opacity: 0.8}}>Online</div>
                    <div
                      className="pf-v6-u-font-size-xl pf-v6-u-font-weight-bold pf-v6-u-mt-xs"
                      style={{
                        color: summary?.status === 'online' ? 'var(--pf-v6-global--success-color--100)' : undefined
                      }}
                    >
                      {summary?.status === 'online' ? 'Online' : summary?.status === 'banned' ? 'Banned' : 'Offline'}
                    </div>
                  </div>

                  <div style={{gridColumn: '1 / -1'}}>
                    <div className="pf-v6-u-font-size-sm" style={{opacity: 0.8}}>Aliases</div>
                    {aliasList.length ? (
                      <div style={{marginTop: 8, display: 'flex', flexWrap: 'wrap', gap: 8}}>
                        {aliasList.map((a) => (
                          <span
                            key={a}
                            className="pf-v6-u-mt-xs pf-v6-u-px-md pf-v6-u-py-xs"
                            style={{background: 'var(--pf-v6-global--BackgroundColor--100)', borderRadius: 999}}
                          >
                            {a}
                          </span>
                        ))}
                      </div>
                    ) : (
                      <div style={{marginTop: 8, opacity: 0.8}}>No previous names recorded.</div>
                    )}
                  </div>

                  <div>
                    <div className="pf-v6-u-font-size-sm" style={{opacity: 0.8}}>Last logged in</div>
                    <div className="pf-v6-u-font-size-lg pf-v6-u-font-weight-bold pf-v6-u-mt-xs">
                      {summary?.lastLoggedInAt ? new Date(summary.lastLoggedInAt).toLocaleString() : '-'}
                    </div>
                  </div>

                  <div>
                    <div className="pf-v6-u-font-size-sm" style={{opacity: 0.8}}>Last logged out</div>
                    <div className="pf-v6-u-font-size-lg pf-v6-u-font-weight-bold pf-v6-u-mt-xs">
                      {summary?.lastLoggedOutAt ? new Date(summary.lastLoggedOutAt).toLocaleString() : '-'}
                    </div>
                  </div>

                  <div style={{gridColumn: '1 / -1'}}>
                    <div className="pf-v6-u-font-size-sm" style={{opacity: 0.8}}>Current server</div>
                    <div className="pf-v6-u-font-size-lg pf-v6-u-font-weight-bold pf-v6-u-mt-xs">
                      {summary?.currentServer ? summary.currentServer : '-'}
                    </div>
                  </div>

                  <div style={{gridColumn: '1 / -1'}}>
                    <div className="pf-v6-u-font-size-sm" style={{opacity: 0.8}}>Client brand</div>
                    <div className="pf-v6-u-font-size-lg pf-v6-u-font-weight-bold pf-v6-u-mt-xs">
                      {summary?.clientBrand?.trim() ? summary.clientBrand : '-'}
                    </div>
                  </div>

                  <div style={{gridColumn: '1 / -1'}}>
                    <div className="pf-v6-u-font-size-sm" style={{opacity: 0.8}}>Detected mods</div>
                    {summary?.clientMods?.length ? (
                      <div style={{marginTop: 8, display: 'flex', flexWrap: 'wrap', gap: 8}}>
                        {summary.clientMods.map((mod) => (
                          <span
                            key={mod}
                            className="pf-v6-u-mt-xs pf-v6-u-px-md pf-v6-u-py-xs"
                            style={{background: 'var(--pf-v6-global--BackgroundColor--100)', borderRadius: 999}}
                          >
                            {mod}
                          </span>
                        ))}
                      </div>
                    ) : (
                      <div style={{marginTop: 8, opacity: 0.8}}>No mods detected.</div>
                    )}
                  </div>
                </div>
              </CardBody>
            </Card>
          </GridItem>
        </Grid>
      </CardBody>
    </Card>
  );
};

export default PlayerDetailOverviewTab;

