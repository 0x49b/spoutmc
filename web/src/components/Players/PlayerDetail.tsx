import React, {useEffect, useMemo, useRef, useState} from 'react';
import {useNavigate, useParams} from 'react-router-dom';
import {
  Button,
  Card,
  CardBody,
  CardHeader,
  Checkbox,
  EmptyState,
  EmptyStateBody,
  EmptyStateVariant,
  FormGroup,
  FormSelect,
  FormSelectOption,
  Grid,
  GridItem,
  Modal,
  ModalBody,
  ModalFooter,
  ModalVariant,
  PageSection,
  Tab,
  TabTitleText,
  Tabs,
  TextInput,
  Title,
  Toolbar,
  ToolbarContent,
  ToolbarItem,
  DatePicker,
  TimePicker
} from '@patternfly/react-core';

import * as api from '../../service/apiService';
import {useAuthStore} from '../../store/authStore';
import {
  BanDurationOptionDTO,
  PlayerBanHistoryDTO,
  PlayerChatMessageDTO,
  PlayerConversationDTO,
  PlayerKickHistoryDTO,
  PlayerSummaryDTO
} from '../../service/apiService';

import './PlayerDetail.css';

const PlayerDetail: React.FC = () => {
  const {playerUuid} = useParams<{playerUuid: string}>();
  const navigate = useNavigate();
  const currentUser = useAuthStore((s) => s.user);

  const currentStaffUserID = useMemo(() => {
    if (!currentUser) return null;
    const n = Number(currentUser.id);
    return Number.isFinite(n) ? n : null;
  }, [currentUser]);

  // Effective UUID avoids repeatedly resolving name->UUID on every poll tick.
  const [effectivePlayerUuid, setEffectivePlayerUuid] = useState<string>(playerUuid ?? '');

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [summary, setSummary] = useState<PlayerSummaryDTO | null>(null);
  const [aliases, setAliases] = useState<string[]>([]);
  const [conversations, setConversations] = useState<PlayerConversationDTO[]>([]);
  const [hasOtherConversations, setHasOtherConversations] = useState(false);

  const hasViewAllConversations = useAuthStore((s) => s.hasPermission('player.conversations.view_all'));

  const [selectedConversationId, setSelectedConversationId] = useState<number | null>(null);
  const [messages, setMessages] = useState<PlayerChatMessageDTO[]>([]);
  /** Draft thread: empty view until first send; first message notifies the player in-game. */
  const [pendingNewConversation, setPendingNewConversation] = useState(false);

  type TabKey = 'overview' | 'moderation' | 'messages';
  const [activeTab, setActiveTab] = useState<TabKey>('overview');

  const [banDurations, setBanDurations] = useState<BanDurationOptionDTO[]>([]);
  const [banReason, setBanReason] = useState('');
  const [banPermanent, setBanPermanent] = useState(false);
  const [banDurationKey, setBanDurationKey] = useState<string>('1h');
  const [useCustomUntil, setUseCustomUntil] = useState(false);
  const [customDate, setCustomDate] = useState<string>('');
  const [customTime, setCustomTime] = useState<string>('');

  const [kickReason, setKickReason] = useState('');

  const [bansHistory, setBansHistory] = useState<PlayerBanHistoryDTO[]>([]);
  const [kicksHistory, setKicksHistory] = useState<PlayerKickHistoryDTO[]>([]);

  const [confirmBanOpen, setConfirmBanOpen] = useState(false);
  const [confirmUnbanOpen, setConfirmUnbanOpen] = useState(false);

  const messagesEndRef = useRef<HTMLDivElement | null>(null);
  const lastKnownMinecraftNameRef = useRef<string | null>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({behavior: 'smooth', block: 'end'});
  };

  const loadBanDurations = async () => {
    const res = await api.getBanDurations();
    setBanDurations(res.data.options);
    if (res.data.options[0]?.key) {
      setBanDurationKey(res.data.options[0].key);
    }
  };

  const loadPlayerDetail = async (uuidToLoad: string) => {
    if (!uuidToLoad) return;

    const summaryRes = await api.getPlayerSummary(uuidToLoad);
    const resolvedUuid = summaryRes.data.minecraftUuid ?? uuidToLoad;

    if (resolvedUuid && resolvedUuid !== effectivePlayerUuid) {
      setEffectivePlayerUuid(resolvedUuid);
    }

    const [conversationsRes, bansRes, kicksRes, aliasesRes] = await Promise.all([
      api.getPlayerConversations(resolvedUuid),
      api.getPlayerBans(resolvedUuid),
      api.getPlayerKicks(resolvedUuid),
      api.getPlayerAliases(resolvedUuid)
    ]);

    setSummary(summaryRes.data);
    lastKnownMinecraftNameRef.current = summaryRes.data.minecraftName ?? null;
    setAliases(aliasesRes.data.aliases);
    setConversations(conversationsRes.data.conversations);
    setHasOtherConversations(conversationsRes.data.hasOtherConversations);
    setBansHistory(bansRes.data);
    setKicksHistory(kicksRes.data);
  };

  const loadMessages = async (conversationId: number) => {
    if (!effectivePlayerUuid) return;
    const res = await api.getConversationMessages(effectivePlayerUuid, conversationId);
    setMessages(res.data);
  };

  const refreshAll = async () => {
    setLoading(true);
    setError(null);
    try {
      await loadBanDurations();
      await loadPlayerDetail(effectivePlayerUuid);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load player detail');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    setEffectivePlayerUuid(playerUuid ?? '');
    setPendingNewConversation(false);
    void refreshAll();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [playerUuid]);

  // Default selection when conversations load.
  useEffect(() => {
    if (pendingNewConversation) return;
    if (selectedConversationId != null) return;
    if (!conversations.length) return;

    setSelectedConversationId(conversations[0].id);
  }, [conversations, selectedConversationId, pendingNewConversation]);

  useEffect(() => {
    if (!effectivePlayerUuid) return;

    const pollMs = 2500;
    let cancelled = false;

    const poll = async () => {
      try {
        // Keep Overview (online/offline/server) and Moderation (active ban) in sync.
        const summaryRes = await api.getPlayerSummary(effectivePlayerUuid);
        if (!cancelled) {
          setSummary(summaryRes.data);

          const nextName = summaryRes.data.minecraftName ?? null;
          if (nextName && nextName !== lastKnownMinecraftNameRef.current) {
            lastKnownMinecraftNameRef.current = nextName;
            try {
              const aliasesRes = await api.getPlayerAliases(effectivePlayerUuid);
              if (!cancelled) setAliases(aliasesRes.data.aliases);
            } catch {
              // Ignore alias refresh failures; next tick may succeed.
            }
          }
        }
      } catch {
        // Keep UI usable even if summary load fails.
      }

      if (!pendingNewConversation && selectedConversationId != null) {
        try {
          await loadMessages(selectedConversationId);
        } catch {
          // Keep UI usable even if message load fails.
        }
      }

      try {
        const conversationsRes = await api.getPlayerConversations(effectivePlayerUuid);
        if (!cancelled) {
          setConversations(conversationsRes.data.conversations);
          setHasOtherConversations(conversationsRes.data.hasOtherConversations);
        }
      } catch {
        // Ignore conversation poll failures; next tick may succeed.
      }
    };

    void poll();
    const intervalId = window.setInterval(() => void poll(), pollMs);

    return () => {
      cancelled = true;
      window.clearInterval(intervalId);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [effectivePlayerUuid, selectedConversationId, pendingNewConversation]);

  useEffect(() => {
    scrollToBottom();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [messages]);

  const selectedConversation = useMemo(() => {
    if (selectedConversationId == null) return null;
    return conversations.find((c) => c.id === selectedConversationId) ?? null;
  }, [conversations, selectedConversationId]);

  const canUseMessageComposer = useMemo(() => {
    if (currentStaffUserID == null) return false;
    if (pendingNewConversation) return true;
    if (selectedConversationId == null) return false;
    const c = conversations.find((x) => x.id === selectedConversationId);
    if (!c || c.closed) return false;
    return c.staffUserId === currentStaffUserID;
  }, [currentStaffUserID, pendingNewConversation, selectedConversationId, conversations]);

  const canCloseSelectedConversation = useMemo(() => {
    if (!selectedConversation || selectedConversation.closed) return false;
    if (currentStaffUserID == null) return false;
    if (selectedConversation.staffUserId === currentStaffUserID) return true;
    return hasViewAllConversations;
  }, [selectedConversation, currentStaffUserID, hasViewAllConversations]);

  const [composerText, setComposerText] = useState('');
  const sendMessage = async () => {
    if (!effectivePlayerUuid) return;
    const text = composerText.trim();
    if (!text) return;
    const isFirstInPendingThread = pendingNewConversation;
    const res = await api.sendPlayerMessage(effectivePlayerUuid, text, {
      newConversation: isFirstInPendingThread,
      conversationId: !isFirstInPendingThread && selectedConversationId != null ? selectedConversationId : undefined
    });
    setComposerText('');
    setPendingNewConversation(false);

    const cid = res.data.conversationId;
    if (typeof cid === 'number') {
      setSelectedConversationId(cid);
      await loadMessages(cid);
    } else if (selectedConversationId != null) {
      await loadMessages(selectedConversationId);
    }
    await loadPlayerDetail(effectivePlayerUuid);
  };

  const startNewConversation = () => {
    if (currentStaffUserID == null) return;
    setPendingNewConversation(true);
    setSelectedConversationId(null);
    setMessages([]);
    setComposerText('');
  };

  const closeConversation = async () => {
    if (!effectivePlayerUuid || selectedConversationId == null) return;
    await api.closePlayerConversation(effectivePlayerUuid, selectedConversationId);
    await loadPlayerDetail(effectivePlayerUuid);
    try {
      await loadMessages(selectedConversationId);
    } catch {
      // ignore
    }
  };

  const submitBan = async () => {
    if (!effectivePlayerUuid) return;
    const reason = banReason.trim();
    if (!reason) return;

    if (banPermanent) {
      await api.banPlayer(effectivePlayerUuid, reason, {permanent: true});
    } else {
      // Timed ban: either use custom until or predefined duration.
      let untilAtISO: string | null = null;

      if (useCustomUntil && customDate && customTime) {
        // Interpret as local datetime, then convert to ISO.
        untilAtISO = new Date(`${customDate}T${customTime}`).toISOString();
      } else {
        const opt = banDurations.find((d) => d.key === banDurationKey);
        if (opt) {
          untilAtISO = new Date(Date.now() + opt.durationSeconds * 1000).toISOString();
        }
      }

      if (!untilAtISO) return;
      await api.banPlayer(effectivePlayerUuid, reason, {untilAt: untilAtISO, permanent: false});
    }

    await loadPlayerDetail(effectivePlayerUuid);
    setBanReason('');
  };

  const submitKick = async () => {
    if (!effectivePlayerUuid) return;
    const reason = kickReason.trim();
    if (!reason) return;
    await api.kickPlayer(effectivePlayerUuid, reason);
    await loadPlayerDetail(effectivePlayerUuid);
    setKickReason('');
  };

  const submitUnban = async () => {
    if (!effectivePlayerUuid) return;
    await api.unbanPlayer(effectivePlayerUuid);
    await loadPlayerDetail(effectivePlayerUuid);
  };

  const computeBanUntilPreview = (): string => {
    if (banPermanent) return 'Permanent';
    if (useCustomUntil && customDate && customTime) {
      return `${customDate} ${customTime} (local)`;
    }
    const opt = banDurations.find((d) => d.key === banDurationKey);
    if (!opt) return 'Timed';
    const until = new Date(Date.now() + opt.durationSeconds * 1000);
    return until.toLocaleString();
  };

  if (!effectivePlayerUuid) {
    return <EmptyState variant={EmptyStateVariant.full} titleText="No player selected"/>;
  }

  if (loading) {
    return <PageSection isFilled style={{minHeight: 240}} />;
  }

  if (error) {
    return (
      <EmptyState variant={EmptyStateVariant.lg} titleText="Failed to load player details" headingLevel="h2">
        <EmptyStateBody>{error}</EmptyStateBody>
        <Button variant="primary" onClick={() => navigate('/players')}>Back to players</Button>
      </EmptyState>
    );
  }

  const activeBan = summary?.banned ? summary : null;
  const currentNameLower = summary?.minecraftName?.toLowerCase() ?? null;
  const aliasList = currentNameLower ? aliases.filter((a) => a.toLowerCase() !== currentNameLower) : aliases;

  return (
    <PageSection>
      <Toolbar className="pf-v6-u-mb-md">
        <ToolbarContent>
          <ToolbarItem>
            <Button variant="secondary" onClick={() => navigate(-1)}>Back</Button>
          </ToolbarItem>
          <ToolbarItem>
            <Title headingLevel="h1" size="2xl">Player details</Title>
          </ToolbarItem>
        </ToolbarContent>
      </Toolbar>

      <Tabs
        activeKey={activeTab}
        onSelect={(_event, tabKey) => {
          if (typeof tabKey === 'number') {
            if (tabKey === 0) setActiveTab('overview');
            else if (tabKey === 1) setActiveTab('moderation');
            else setActiveTab('messages');
            return;
          }
          setActiveTab(tabKey as TabKey);
        }}
        isBox
      >
        <Tab
          eventKey="overview"
          title={<TabTitleText>Overview</TabTitleText>}
        >
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
                          <div className="pf-v6-u-font-size-xl pf-v6-u-font-weight-bold pf-v6-u-mt-xs" style={{color: summary?.status === 'online' ? 'var(--pf-v6-global--success-color--100)' : undefined}}>
                            {summary?.status === 'online' ? 'Online' : summary?.status === 'banned' ? 'Banned' : 'Offline'}
                          </div>
                        </div>

                        <div style={{gridColumn: '1 / -1'}}>
                          <div className="pf-v6-u-font-size-sm" style={{opacity: 0.8}}>Aliases</div>
                          {aliasList.length ? (
                            <div style={{marginTop: 8, display: 'flex', flexWrap: 'wrap', gap: 8}}>
                              {aliasList.map((a) => (
                                <span key={a} className="pf-v6-u-mt-xs pf-v6-u-px-md pf-v6-u-py-xs" style={{background: 'var(--pf-v6-global--BackgroundColor--100)', borderRadius: 999}}>
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
                      </div>
                    </CardBody>
                  </Card>
                </GridItem>
              </Grid>
            </CardBody>
          </Card>
        </Tab>

        <Tab
          eventKey="moderation"
          title={<TabTitleText>Moderation</TabTitleText>}
        >
          <Grid hasGutter className="pf-v6-u-mt-md">
            <GridItem span={6}>
              <Card isCompact>
                <CardHeader>
                  <Title headingLevel="h2" size="lg">Ban / Kick</Title>
                </CardHeader>
                <CardBody>
                  <FormGroup label="Ban reason" fieldId="ban-reason">
                    <TextInput id="ban-reason" value={banReason} onChange={(_ev, value) => setBanReason(value)} />
                  </FormGroup>

                  <Checkbox
                    label="Permanent ban"
                    id="ban-permanent"
                    isChecked={banPermanent}
                    onChange={(_ev, checked) => {
                      setBanPermanent(checked);
                      if (checked) setUseCustomUntil(false);
                    }}
                  />

                  <div style={{height: 12}} />

                  <FormGroup label="Predefined duration" fieldId="ban-duration">
                    <FormSelect
                      value={banDurationKey}
                      isDisabled={banPermanent}
                      onChange={(_ev, value) => setBanDurationKey(value as string)}
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
                    onChange={(_ev, checked) => setUseCustomUntil(checked)}
                  />

                  {useCustomUntil && !banPermanent ? (
                    <div style={{marginTop: 12}}>
                      <FormGroup label="Until date" fieldId="ban-until-date">
                        <DatePicker
                          aria-label="Until date"
                          value={customDate}
                          placeholder="YYYY-MM-DD"
                          onChange={(_ev, value) => setCustomDate(value)}
                        />
                      </FormGroup>
                      <FormGroup label="Until time" fieldId="ban-until-time">
                        <TimePicker
                          aria-label="Until time"
                          time={customTime}
                          placeholder="hh:mm"
                          onChange={(_ev, value) => setCustomTime(value)}
                        />
                      </FormGroup>
                    </div>
                  ) : null}

                  <div style={{marginTop: 12, display: 'flex', gap: 8}}>
                    <Button
                      variant="danger"
                      onClick={() => setConfirmBanOpen(true)}
                      isDisabled={!banReason.trim()}
                    >
                      Ban
                    </Button>
                    {activeBan ? (
                      <Button variant="secondary" onClick={() => setConfirmUnbanOpen(true)}>
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
                        onChange={(_ev, value) => setKickReason(value)}
                      />
                    </FormGroup>
                    <Button
                      variant="warning"
                      onClick={() => void submitKick()}
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
        </Tab>

        <Tab
          eventKey="messages"
          title={<TabTitleText>Messages</TabTitleText>}
        >
          <Grid hasGutter className="pf-v6-u-mt-md">
            <GridItem span={4}>
              <Card>
                <CardHeader>
                  <div style={{display: 'flex', flexWrap: 'wrap', alignItems: 'flex-start', justifyContent: 'space-between', gap: 8}}>
                    <div>
                      <Title headingLevel="h2" size="lg">Conversations</Title>
                      {hasOtherConversations ? (
                        <div style={{fontSize: 12, opacity: 0.8, marginTop: 4}}>
                          Other conversations are hidden (insufficient permission).
                        </div>
                      ) : null}
                    </div>
                    <Button
                      variant="secondary"
                      size="sm"
                      isDisabled={currentStaffUserID == null}
                      onClick={() => startNewConversation()}
                    >
                      Start new conversation
                    </Button>
                  </div>
                </CardHeader>
                <CardBody>
                  {!conversations.length ? (
                    <EmptyState variant={EmptyStateVariant.sm} titleText="No conversations yet" />
                  ) : (
                    <div className="playerDetail__conversationList">
                      {conversations.map((conv) => (
                        <Button
                          key={conv.id}
                          variant={conv.id === selectedConversationId ? 'primary' : 'link'}
                          isInline
                          className="playerDetail__conversationButton"
                          onClick={() => {
                            setPendingNewConversation(false);
                            setSelectedConversationId(conv.id);
                          }}
                        >
                          <div style={{textAlign: 'left'}}>
                            <div className="playerDetail__conversationName">
                              {conv.staffDisplayName}
                              {conv.closed ? (
                                <span style={{marginLeft: 8, fontSize: 11, opacity: 0.75}}>(closed)</span>
                              ) : null}
                            </div>
                            <div className="playerDetail__conversationPreview">
                              {conv.lastMessage ? conv.lastMessage.slice(0, 60) : ''}
                            </div>
                          </div>
                        </Button>
                      ))}
                    </div>
                  )}
                </CardBody>
              </Card>
            </GridItem>

            <GridItem span={8}>
              <Card>
                <CardHeader>
                  <div style={{display: 'flex', flexWrap: 'wrap', alignItems: 'center', justifyContent: 'space-between', gap: 8}}>
                    <Title headingLevel="h2" size="lg">Messages</Title>
                    {canCloseSelectedConversation ? (
                      <Button variant="secondary" size="sm" onClick={() => void closeConversation()}>
                        Close conversation
                      </Button>
                    ) : null}
                  </div>
                </CardHeader>
                <CardBody>
                  {pendingNewConversation ? (
                    <div className="pf-v6-u-font-size-sm pf-v6-u-mb-md" style={{opacity: 0.85}}>
                      You are starting a new conversation with this player. Your first message will show them the support-chat notice in-game, then deliver your text.
                    </div>
                  ) : null}
                  {selectedConversation?.closed ? (
                    <div className="pf-v6-u-font-size-sm pf-v6-u-mb-md" style={{opacity: 0.85}}>
                      This conversation is closed. No more messages can be sent here.
                    </div>
                  ) : null}
                  <div className="playerDetail__messages">
                    {messages.length ? (
                      messages.map((m, idx) => (
                        <div
                          key={`${m.timestamp}-${idx}`}
                          className={m.direction === 'incoming' ? 'playerDetail__bubble playerDetail__bubble--incoming' : 'playerDetail__bubble playerDetail__bubble--outgoing'}
                        >
                          <div className="playerDetail__bubbleMeta">
                            {m.direction === 'outgoing' && m.role ? `${m.role} ` : ''}
                            {m.direction === 'outgoing' && m.sender ? m.sender : null}
                            <span className="playerDetail__bubbleTime">{new Date(m.timestamp).toLocaleTimeString()}</span>
                          </div>
                          <div className="playerDetail__bubbleText">{m.message}</div>
                        </div>
                      ))
                    ) : (
                      <EmptyState
                        variant={EmptyStateVariant.sm}
                        titleText={pendingNewConversation ? 'New conversation' : 'No messages yet'}
                      />
                    )}
                    <div ref={messagesEndRef} />
                  </div>

                  <form
                    onSubmit={(e) => {
                      e.preventDefault();
                      void sendMessage();
                    }}
                    style={{marginTop: 12, display: 'flex', gap: 8}}
                  >
                    <FormGroup style={{flex: 1, marginBottom: 0}}>
                      <TextInput
                        aria-label="Message"
                        placeholder={
                          !canUseMessageComposer
                            ? (selectedConversation?.closed
                              ? 'This conversation is closed'
                              : 'Select your own open conversation to send')
                            : pendingNewConversation
                              ? 'Write your first message…'
                              : 'Write a message…'
                        }
                        isDisabled={!canUseMessageComposer}
                        value={composerText}
                        onChange={(_ev, value) => setComposerText(value)}
                      />
                    </FormGroup>
                    <Button isDisabled={!canUseMessageComposer} type="submit" variant="primary">
                      Send
                    </Button>
                  </form>
                </CardBody>
              </Card>
            </GridItem>
          </Grid>
        </Tab>
      </Tabs>

      <Modal
        variant={ModalVariant.small}
        title={`Confirm ban${banReason ? `: ${banReason}` : ''}`}
        isOpen={confirmBanOpen}
        onClose={() => setConfirmBanOpen(false)}
      >
        <ModalBody>
          <p>
            You are about to ban <strong>{summary?.minecraftName || playerUuid}</strong>.
          </p>
          <p>
            <strong>Reason:</strong> {banReason || '-'}
          </p>
          <p>
            <strong>Duration:</strong> {computeBanUntilPreview()}
          </p>
        </ModalBody>
        <ModalFooter>
          <Button
            key="confirm-ban"
            variant="danger"
            onClick={() => {
              void submitBan().finally(() => setConfirmBanOpen(false));
            }}
          >
            Confirm Ban
          </Button>
          <Button key="cancel-ban" variant="link" onClick={() => setConfirmBanOpen(false)}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>

      <Modal
        variant={ModalVariant.small}
        title="Confirm unban"
        isOpen={confirmUnbanOpen}
        onClose={() => setConfirmUnbanOpen(false)}
      >
        <ModalBody>
          <p>
            You are about to unban <strong>{summary?.minecraftName || playerUuid}</strong>.
          </p>
        </ModalBody>
        <ModalFooter>
          <Button
            key="confirm-unban"
            variant="secondary"
            onClick={() => {
              void submitUnban().finally(() => setConfirmUnbanOpen(false));
            }}
          >
            Confirm Unban
          </Button>
          <Button key="cancel-unban" variant="link" onClick={() => setConfirmUnbanOpen(false)}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>
    </PageSection>
  );
};

export default PlayerDetail;

