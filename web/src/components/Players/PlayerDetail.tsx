import React, {useEffect, useMemo, useRef, useState} from 'react';
import {useNavigate, useParams} from 'react-router-dom';
import {
    Button,
    EmptyState,
    EmptyStateBody,
    EmptyStateVariant,
    Modal,
    ModalBody,
    ModalFooter,
    ModalVariant,
    PageSection,
    Tab,
    Tabs,
    TabTitleText,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem
} from '@patternfly/react-core';

import * as api from '../../service/apiService';
import {
    BanDurationOptionDTO,
    PlayerBanHistoryDTO,
    PlayerChatMessageDTO,
    PlayerConversationDTO,
    PlayerJournalEntryDTO,
    PlayerKickHistoryDTO,
    PlayerSummaryDTO
} from '../../service/apiService';
import {useAuthStore} from '../../store/authStore';

import './PlayerDetail.css';
import PlayerDetailMessagesTab from './PlayerDetailMessagesTab';
import PlayerDetailModerationTab from './PlayerDetailModerationTab';
import PlayerDetailOverviewTab from './PlayerDetailOverviewTab';
import PlayerDetailJournalTab from './PlayerDetailJournalTab';

type TabKey = 'overview' | 'moderation' | 'messages' | 'journal';

const PlayerDetail: React.FC = () => {

    const navigate = useNavigate();
    const {playerUuid} = useParams<{ playerUuid: string }>();
    const currentUser = useAuthStore((s) => s.user);
    const hasViewAllConversations = useAuthStore((s) => s.hasPermission('player.conversations.view_all'));


    const currentStaffUserID = useMemo(() => {
        if (!currentUser) return null;
        const n = Number(currentUser.id);
        return Number.isFinite(n) ? n : null;
    }, [currentUser]);


    const [effectivePlayerUuid, setEffectivePlayerUuid] = useState<string>(playerUuid ?? '');
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [summary, setSummary] = useState<PlayerSummaryDTO | null>(null);
    const [aliases, setAliases] = useState<string[]>([]);
    const [conversations, setConversations] = useState<PlayerConversationDTO[]>([]);
    const [hasOtherConversations, setHasOtherConversations] = useState(false);
    const [selectedConversationId, setSelectedConversationId] = useState<number | null>(null);
    const [messages, setMessages] = useState<PlayerChatMessageDTO[]>([]);
    const [pendingNewConversation, setPendingNewConversation] = useState(false);
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
    const [journalEntries, setJournalEntries] = useState<PlayerJournalEntryDTO[]>([]);
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

        const [conversationsRes, bansRes, kicksRes, aliasesRes, journalRes] = await Promise.all([
            api.getPlayerConversations(resolvedUuid),
            api.getPlayerBans(resolvedUuid),
            api.getPlayerKicks(resolvedUuid),
            api.getPlayerAliases(resolvedUuid),
            api.getPlayerJournal(resolvedUuid)
        ]);

        setSummary(summaryRes.data);
        lastKnownMinecraftNameRef.current = summaryRes.data.minecraftName ?? null;
        setAliases(aliasesRes.data.aliases);
        setConversations(conversationsRes.data.conversations);
        setHasOtherConversations(conversationsRes.data.hasOtherConversations);
        setBansHistory(bansRes.data);
        setKicksHistory(kicksRes.data);
        setJournalEntries(journalRes.data);
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
        const intervalId = globalThis.setInterval(() => void poll(), pollMs);

        return () => {
            cancelled = true;
            globalThis.clearInterval(intervalId);
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
    const [journalEntryText, setJournalEntryText] = useState('');
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

    const saveJournalEntry = async () => {
        if (!effectivePlayerUuid) return;
        const entry = journalEntryText.trim();
        if (!entry) return;
        await api.addPlayerJournalEntry(effectivePlayerUuid, entry);
        setJournalEntryText('');
        const journalRes = await api.getPlayerJournal(effectivePlayerUuid);
        setJournalEntries(journalRes.data);
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
            await api.banPlayer(effectivePlayerUuid, reason, {
                untilAt: untilAtISO,
                permanent: false
            });
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
        return <PageSection isFilled style={{minHeight: 240}}/>;
    }

    if (error) {
        return (
            <EmptyState variant={EmptyStateVariant.lg} titleText="Failed to load player details"
                        headingLevel="h2">
                <EmptyStateBody>{error}</EmptyStateBody>
                <Button variant="primary" onClick={() => navigate('/players')}>Back to
                    players</Button>
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
                        else if (tabKey === 2) setActiveTab('messages');
                        else setActiveTab('journal');
                        return;
                    }
                    setActiveTab(tabKey as TabKey);
                }}
                isBox
            >
                <Tab eventKey="overview" title={<TabTitleText>Overview</TabTitleText>}>
                    <PlayerDetailOverviewTab
                        summary={summary}
                        effectivePlayerUuid={effectivePlayerUuid}
                        aliasList={aliasList}
                    />
                </Tab>

                <Tab eventKey="moderation" title={<TabTitleText>Moderation</TabTitleText>}>
                    <PlayerDetailModerationTab
                        activeBan={activeBan}
                        banDurations={banDurations}
                        banReason={banReason}
                        banPermanent={banPermanent}
                        banDurationKey={banDurationKey}
                        useCustomUntil={useCustomUntil}
                        customDate={customDate}
                        customTime={customTime}
                        kickReason={kickReason}
                        bansHistory={bansHistory}
                        kicksHistory={kicksHistory}
                        onBanReasonChange={setBanReason}
                        onBanPermanentChange={(checked) => {
                            setBanPermanent(checked);
                            if (checked) setUseCustomUntil(false);
                        }}
                        onBanDurationKeyChange={setBanDurationKey}
                        onUseCustomUntilChange={setUseCustomUntil}
                        onCustomDateChange={setCustomDate}
                        onCustomTimeChange={setCustomTime}
                        onKickReasonChange={setKickReason}
                        onRequestConfirmBanOpen={() => setConfirmBanOpen(true)}
                        onRequestConfirmUnbanOpen={() => setConfirmUnbanOpen(true)}
                        onKick={submitKick}
                    />
                </Tab>

                <Tab eventKey="messages" title={<TabTitleText>Messages</TabTitleText>}>
                    <PlayerDetailMessagesTab
                        currentStaffUserID={currentStaffUserID}
                        hasOtherConversations={hasOtherConversations}
                        conversations={conversations}
                        selectedConversationId={selectedConversationId}
                        selectedConversation={selectedConversation}
                        pendingNewConversation={pendingNewConversation}
                        messages={messages}
                        composerText={composerText}
                        onComposerTextChange={setComposerText}
                        canUseMessageComposer={canUseMessageComposer}
                        canCloseSelectedConversation={canCloseSelectedConversation}
                        messagesEndRef={messagesEndRef}
                        onStartNewConversation={startNewConversation}
                        onSelectConversation={(conversationId) => {
                            setPendingNewConversation(false);
                            setSelectedConversationId(conversationId);
                        }}
                        onCloseConversation={closeConversation}
                        onSendMessage={sendMessage}
                    />
                </Tab>

                <Tab eventKey="journal" title={<TabTitleText>Journal</TabTitleText>}>
                    <PlayerDetailJournalTab
                        entries={journalEntries}
                        entryText={journalEntryText}
                        onEntryTextChange={setJournalEntryText}
                        onSaveEntry={saveJournalEntry}
                    />
                </Tab>
            </Tabs>

            <Modal
                variant={ModalVariant.small}
                title={`Confirm ban ${banReason ? `: ${banReason}` : ''}`}
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
                        }}>
                        Confirm Ban
                    </Button>
                    <Button key="cancel-ban" variant="link"
                            onClick={() => setConfirmBanOpen(false)}>
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
                        You are about to
                        unban <strong>{summary?.minecraftName || playerUuid}</strong>.
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
                    <Button key="cancel-unban" variant="link"
                            onClick={() => setConfirmUnbanOpen(false)}>
                        Cancel
                    </Button>
                </ModalFooter>
            </Modal>
        </PageSection>
    );
};

export default PlayerDetail;

