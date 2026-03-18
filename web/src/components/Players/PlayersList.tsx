import React, { useEffect, useMemo, useState } from 'react';
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
import { ActionsColumn, IAction, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import PageHeader from '../UI/PageHeader';
import { usePlayerStore } from '../../store/playerStore';
import StatusBadge from '../UI/StatusBadge';

const PlayersList: React.FC = () => {
  const {
    players,
    loading,
    error,
    actionInProgressByPlayer,
    fetchPlayers,
    connectSSE,
    disconnectSSE,
    sendMessage,
    kickPlayer,
    banPlayer
  } = usePlayerStore();

  const [messagePlayer, setMessagePlayer] = useState<string | null>(null);
  const [kickTarget, setKickTarget] = useState<string | null>(null);
  const [banTarget, setBanTarget] = useState<string | null>(null);
  const [messageText, setMessageText] = useState('');
  const [kickReason, setKickReason] = useState('');
  const [banReason, setBanReason] = useState('');
  const [actionError, setActionError] = useState<string | null>(null);

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
  };

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
    void executeAction(() => sendMessage(messagePlayer, messageText.trim()));
  };

  const submitKickForm = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!kickTarget) return;
    void executeAction(() => kickPlayer(kickTarget, kickReason.trim()));
  };

  const submitBanForm = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!banTarget) return;
    void executeAction(() => banPlayer(banTarget, banReason.trim()));
  };

  const getActions = (playerName: string): IAction[] => [
    {
      title: 'Message',
      onClick: () => {
        setActionError(null);
        setMessagePlayer(playerName);
      }
    },
    {
      title: 'Kick',
      onClick: () => {
        setActionError(null);
        setKickTarget(playerName);
      }
    },
    {
      title: 'Ban',
      onClick: () => {
        setActionError(null);
        setBanTarget(playerName);
      }
    }
  ];

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
                  <Th />
                </Tr>
              </Thead>
              <Tbody>
                {sortedPlayers.map(player => (
                  <Tr key={player.id}>
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
                    <Td isActionCell>
                      <ActionsColumn
                        items={getActions(player.username)}
                        isDisabled={Boolean(actionInProgressByPlayer[player.username])}
                      />
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
        variant={ModalVariant.small}
        title={`Send private message${messagePlayer ? ` to ${messagePlayer}` : ''}`}
        isOpen={Boolean(messagePlayer)}
        onClose={resetActionState}
      >
        <ModalBody>
          <Form id="player-message-form" onSubmit={submitMessageForm}>
            <FormGroup label="Message" fieldId="player-message">
              <TextInput id="player-message" value={messageText} onChange={(_event, value) => setMessageText(value)} />
            </FormGroup>
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
    </>
  );
};

export default PlayersList;
