import React, {useEffect, useMemo, useRef, useState} from 'react';
import {useNavigate} from 'react-router-dom';
import {
    Avatar,
    Button,
    Card,
    CardBody,
    EmptyState,
    EmptyStateBody,
    EmptyStateVariant,
    Form,
    FormGroup,
    Modal,
    ModalBody,
    ModalFooter,
    ModalVariant,
    PageSection,
    TextInput,
    Toolbar,
    ToolbarContent,
    ToolbarItem
} from '@patternfly/react-core';
import {Table, Tbody, Td, Th, Thead, Tr} from '@patternfly/react-table';
import PageHeader from '../UI/PageHeader';
import {usePlayerStore} from '../../store/playerStore';
import {useAuthStore} from '../../store/authStore';
import StatusBadge from '../UI/StatusBadge';
import {PlayerChatMessageDTO} from '../../service/apiService';

const PlayersList: React.FC = () => {
  const {
    players,
    loading,
    error,
    fetchPlayers,
    connectSSE,
    disconnectSSE,
    sendMessage,
    getPlayerChat,
    kickPlayer,
    banPlayer
  } = usePlayerStore();
  const currentUser = useAuthStore(state => state.user);
  const navigate = useNavigate();

  const [messagePlayer, setMessagePlayer] = useState<string | null>(null);
  const [kickTarget, setKickTarget] = useState<string | null>(null);
  const [banTarget, setBanTarget] = useState<string | null>(null);
  const [messageText, setMessageText] = useState('');
  const [kickReason, setKickReason] = useState('');
  const [banReason, setBanReason] = useState('');
  const [actionError, setActionError] = useState<string | null>(null);
  const [chatMessages, setChatMessages] = useState<PlayerChatMessageDTO[]>([]);
  const [chatLoading, setChatLoading] = useState(false);
  const pollingRef = useRef<number | null>(null);
  const [confirmBanOpen, setConfirmBanOpen] = useState(false);

  useEffect(() => {
    fetchPlayers();
    connectSSE();
    return () => disconnectSSE();
  }, [fetchPlayers, connectSSE, disconnectSSE]);

  const sortedPlayers = useMemo(
    () =>
      [...players].sort((a, b) => {
        if (a.status === 'online' && b.status !== 'online') return -1;
        if (a.status !== 'online' && b.status === 'online') return 1;
        return a.username.localeCompare(b.username);
      }),
    [players]
  );

  const formatDateTime = (value?: string) => {
    if (!value) return '-';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '-';
    return date.toLocaleString();
  };

  const resetActionState = () => {
    setMessagePlayer(null);
    setKickTarget(null);
    setBanTarget(null);
    setMessageText('');
    setKickReason('');
    setBanReason('');
    setActionError(null);
    setChatMessages([]);
    setChatLoading(false);
    if (pollingRef.current) {
      window.clearInterval(pollingRef.current);
      pollingRef.current = null;
    }
  };

  const getPrimaryRole = () => {
    const roles = currentUser?.roles ?? [];
    if (roles.includes('admin')) return 'admin';
    if (roles.includes('moderator')) return 'moderator';
    if (roles.includes('viewer')) return 'viewer';
    return 'staff';
  };

  const getSenderDisplayName = () => {
    return currentUser?.displayName?.trim() || currentUser?.email?.trim() || 'SpoutMC';
  };

  const loadChat = async (playerName: string) => {
    setChatLoading(true);
    try {
      const messages = await getPlayerChat(playerName);
      setChatMessages(messages);
      setActionError(null);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to load chat');
    } finally {
      setChatLoading(false);
    }
  };

  useEffect(() => {
    if (!messagePlayer) {
      if (pollingRef.current) {
        window.clearInterval(pollingRef.current);
        pollingRef.current = null;
      }
      return;
    }

    void loadChat(messagePlayer);
    pollingRef.current = window.setInterval(() => {
      void loadChat(messagePlayer);
    }, 2500);

    return () => {
      if (pollingRef.current) {
        window.clearInterval(pollingRef.current);
        pollingRef.current = null;
      }
    };
  }, [messagePlayer]);

  const executeAction = async (action: () => Promise<void>) => {
    setActionError(null);
    try {
      await action();
      resetActionState();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Action failed');
    }
  };

  const submitMessageForm = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!messagePlayer || messageText.trim() === '') return;
    const sender = getSenderDisplayName();
    const role = getPrimaryRole();
    setActionError(null);
    void (async () => {
      try {
        await sendMessage(messagePlayer, messageText.trim(), sender, role);
        setMessageText('');
        await loadChat(messagePlayer);
      } catch (err) {
        setActionError(err instanceof Error ? err.message : 'Action failed');
      }
    })();
  };

  const submitKickForm = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!kickTarget) return;
    void executeAction(() => kickPlayer(kickTarget, kickReason.trim()));
  };

  const submitBanForm = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!banTarget) return;
    setConfirmBanOpen(true);
  };

  return (
    <>
      <PageHeader title="Players" description="Manage players" />
      <PageSection>
        <Toolbar className="pf-v6-u-mb-md">
          <ToolbarContent>
            <ToolbarItem>
              <Button variant="secondary" onClick={fetchPlayers} isDisabled={loading}>
                Refresh
              </Button>
            </ToolbarItem>
          </ToolbarContent>
        </Toolbar>

        {error ? (
          <EmptyState variant={EmptyStateVariant.lg} titleText="Unable to load players">
            <EmptyStateBody>{error}</EmptyStateBody>
          </EmptyState>
        ) : null}

        <Card>
          <CardBody>
            <Table aria-label="Players table" variant="compact">
              <Thead>
                <Tr>
                  <Th>Player</Th>
                  <Th>Last login</Th>
                  <Th>Last logout</Th>
                  <Th>Current server</Th>
                  <Th>Banned</Th>
                  <Th>Status</Th>
                </Tr>
              </Thead>
              <Tbody>
                {sortedPlayers.map(player => (
                  <Tr
                    key={player.id}
                    style={{cursor: 'pointer'}}
                    onClick={() => navigate(`/players/${player.id}`)}
                  >
                    <Td dataLabel="Player">
                      <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                        <Avatar src={player.avatarDataUrl} alt={`${player.username} avatar`} size="sm" />
                        <span>{player.username}</span>
                      </div>
                    </Td>
                    <Td dataLabel="Last login">{formatDateTime(player.lastLoggedInAt)}</Td>
                    <Td dataLabel="Last logout">{formatDateTime(player.lastLoggedOutAt)}</Td>
                    <Td dataLabel="Current server">{player.currentServer || '-'}</Td>
                    <Td dataLabel="Banned">
                      <StatusBadge status={player.banned ? 'banned' : 'offline'} />
                    </Td>
                    <Td dataLabel="Status">
                      <StatusBadge status={player.status} />
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>

            {sortedPlayers.length === 0 ? (
              <EmptyState variant={EmptyStateVariant.sm} titleText="No players discovered yet">
                <EmptyStateBody>Players appear here automatically once they join the network.</EmptyStateBody>
              </EmptyState>
            ) : null}
          </CardBody>
        </Card>
      </PageSection>

      <Modal
        variant={ModalVariant.medium}
        title={`Send private message${messagePlayer ? ` to ${messagePlayer}` : ''}`}
        isOpen={Boolean(messagePlayer)}
        onClose={resetActionState}
      >
        <ModalBody>
          {messagePlayer ? (
            <div
              style={{
                border: '1px solid var(--pf-t--global--border--color--default)',
                borderRadius: '8px',
                padding: '0.75rem',
                maxHeight: '280px',
                overflowY: 'auto',
                marginBottom: '1rem',
                background: 'var(--pf-t--global--background--color--primary--default)'
              }}
            >
              {chatLoading && chatMessages.length === 0 ? <div>Loading chat…</div> : null}
              {!chatLoading && chatMessages.length === 0 ? (
                <div className="pf-v6-u-color-200">No chat messages yet. Start a conversation below.</div>
              ) : null}
              {chatMessages.map((entry, index) => {
                const isOutgoing = entry.direction === 'outgoing';
                const senderLabel = isOutgoing
                  ? `[${(entry.role || 'staff').toUpperCase()}] ${entry.sender || 'SpoutMC'}`
                  : `${entry.player}`;
                return (
                  <div key={`${entry.timestamp}-${index}`} style={{ marginBottom: '0.6rem' }}>
                    <div
                      className="pf-v6-u-font-size-sm pf-v6-u-color-200"
                      style={{ display: 'flex', justifyContent: 'space-between', gap: '0.5rem' }}
                    >
                      <span>{senderLabel}</span>
                      <span>{new Date(entry.timestamp).toLocaleTimeString()}</span>
                    </div>
                    <div>{entry.message}</div>
                  </div>
                );
              })}
            </div>
          ) : null}
          <Form id="player-message-form" onSubmit={submitMessageForm}>
            <FormGroup label="Message" fieldId="player-message">
              <TextInput id="player-message" value={messageText} onChange={(_event, value) => setMessageText(value)} />
            </FormGroup>
            <div className="pf-v6-u-font-size-sm pf-v6-u-color-200 pf-v6-u-mb-sm">
              Sent as [{getPrimaryRole().toUpperCase()}] {getSenderDisplayName()}
            </div>
            {actionError ? <div className="pf-v6-u-danger-color-100">{actionError}</div> : null}
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button
            key="send"
            variant="primary"
            type="submit"
            form="player-message-form"
            isDisabled={!messagePlayer || messageText.trim() === ''}
          >
            Send
          </Button>
          <Button key="cancel" variant="link" type="button" onClick={resetActionState}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>

      <Modal
        variant={ModalVariant.small}
        title={`Kick player${kickTarget ? ` ${kickTarget}` : ''}`}
        isOpen={Boolean(kickTarget)}
        onClose={resetActionState}
      >
        <ModalBody>
          <Form id="player-kick-form" onSubmit={submitKickForm}>
            <FormGroup label="Reason" fieldId="kick-reason">
              <TextInput id="kick-reason" value={kickReason} onChange={(_event, value) => setKickReason(value)} />
            </FormGroup>
            {actionError ? <div className="pf-v6-u-danger-color-100">{actionError}</div> : null}
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button key="kick" variant="warning" type="submit" form="player-kick-form">
            Kick
          </Button>
          <Button key="cancel" variant="link" type="button" onClick={resetActionState}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>

      <Modal
        variant={ModalVariant.small}
        title={`Ban player${banTarget ? ` ${banTarget}` : ''}`}
        isOpen={Boolean(banTarget)}
        onClose={resetActionState}
      >
        <ModalBody>
          <Form id="player-ban-form" onSubmit={submitBanForm}>
            <FormGroup label="Reason" fieldId="ban-reason">
              <TextInput id="ban-reason" value={banReason} onChange={(_event, value) => setBanReason(value)} />
            </FormGroup>
            {actionError ? <div className="pf-v6-u-danger-color-100">{actionError}</div> : null}
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button key="ban" variant="danger" type="submit" form="player-ban-form">
            Ban
          </Button>
          <Button key="cancel" variant="link" type="button" onClick={resetActionState}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>

      <Modal
        variant={ModalVariant.small}
        title={`Confirm ban${banTarget ? ` ${banTarget}` : ''}`}
        isOpen={confirmBanOpen}
        onClose={() => setConfirmBanOpen(false)}
      >
        <ModalBody>
          <p>
            Are you sure you want to ban <strong>{banTarget}</strong>?
          </p>
          <p>
            <strong>Reason:</strong> {banReason || '-'}
          </p>
        </ModalBody>
        <ModalFooter>
          <Button
            key="confirm-ban"
            variant="danger"
            type="button"
            onClick={() => {
              if (!banTarget) return;
              void executeAction(() => banPlayer(banTarget, banReason.trim())).finally(() => {
                setConfirmBanOpen(false);
              });
            }}
          >
            Confirm Ban
          </Button>
          <Button key="cancel-confirm-ban" variant="link" type="button" onClick={() => setConfirmBanOpen(false)}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>
    </>
  );
};

export default PlayersList;
