import React, {useEffect, useMemo, useState} from 'react';
import {
    Avatar,
    Button,
    Card,
    CardBody,
    EmptyState,
    EmptyStateBody,
    EmptyStateVariant,
    Modal,
    ModalBody,
    ModalFooter,
    ModalVariant,
    PageSection,
    Toolbar,
    ToolbarContent,
    ToolbarItem
} from '@patternfly/react-core';
import {ActionsColumn, IAction, Table, Tbody, Td, Th, Thead, Tr} from '@patternfly/react-table';
import PageHeader from '../UI/PageHeader';
import {usePlayerStore} from '../../store/playerStore';
import StatusBadge from '../UI/StatusBadge';

const BannedPlayersList: React.FC = () => {
  const {
    getBannedPlayers,
    fetchPlayers,
    connectSSE,
    disconnectSSE,
    unbanPlayer,
    actionInProgressByPlayer
  } = usePlayerStore();
  const [selectedPlayerForUnban, setSelectedPlayerForUnban] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const bannedPlayers = getBannedPlayers();
  const sortedBannedPlayers = useMemo(
    () => [...bannedPlayers].sort((a, b) => a.username.localeCompare(b.username)),
    [bannedPlayers]
  );

  useEffect(() => {
    fetchPlayers();
    connectSSE();
    return () => disconnectSSE();
  }, [fetchPlayers, connectSSE, disconnectSSE]);

  const formatDateTime = (value?: string) => {
    if (!value) return '-';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '-';
    return date.toLocaleString();
  };

  const resetUnbanState = () => {
    setSelectedPlayerForUnban(null);
    setActionError(null);
  };

  const handleConfirmUnban = async () => {
    if (!selectedPlayerForUnban) return;
    setActionError(null);
    try {
      await unbanPlayer(selectedPlayerForUnban);
      resetUnbanState();
    } catch (error) {
      setActionError(error instanceof Error ? error.message : 'Failed to unban player');
    }
  };

  const getActions = (playerName: string): IAction[] => [
    {
      title: 'Unban',
      onClick: () => {
        setActionError(null);
        setSelectedPlayerForUnban(playerName);
      }
    }
  ];

  return (
    <>
      <PageHeader title="Banned Players" description="View banned players" />
      <PageSection>
        <Toolbar className="pf-v6-u-mb-md">
          <ToolbarContent>
            <ToolbarItem>
              <Button variant="secondary" onClick={fetchPlayers}>
                Refresh
              </Button>
            </ToolbarItem>
          </ToolbarContent>
        </Toolbar>

        <Card>
          <CardBody>
            {sortedBannedPlayers.length === 0 ? (
              <EmptyState variant={EmptyStateVariant.sm} titleText="No banned players">
                <EmptyStateBody>No players are currently banned.</EmptyStateBody>
              </EmptyState>
            ) : (
              <Table aria-label="Banned players table" variant="compact">
                <Thead>
                  <Tr>
                    <Th>Player</Th>
                    <Th>Reason</Th>
                    <Th>Last login</Th>
                    <Th>Last logout</Th>
                    <Th>Status</Th>
                    <Th />
                  </Tr>
                </Thead>
                <Tbody>
                  {sortedBannedPlayers.map(player => (
                    <Tr key={player.id}>
                      <Td dataLabel="Player">
                        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                          <Avatar src={player.avatarDataUrl} alt={`${player.username} avatar`} size="sm" />
                          <span>{player.username}</span>
                        </div>
                      </Td>
                      <Td dataLabel="Reason">{player.banReason || '-'}</Td>
                      <Td dataLabel="Last login">{formatDateTime(player.lastLoggedInAt)}</Td>
                      <Td dataLabel="Last logout">{formatDateTime(player.lastLoggedOutAt)}</Td>
                      <Td dataLabel="Status">
                        <StatusBadge status="banned" />
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
            )}
          </CardBody>
        </Card>
      </PageSection>

      <Modal
        variant={ModalVariant.small}
        title={`Unban player${selectedPlayerForUnban ? ` ${selectedPlayerForUnban}` : ''}`}
        isOpen={Boolean(selectedPlayerForUnban)}
        onClose={resetUnbanState}
      >
        <ModalBody>
          {selectedPlayerForUnban ? (
            <p>
              This will remove the ban for <strong>{selectedPlayerForUnban}</strong> and allow them to join again.
            </p>
          ) : null}
          {actionError ? <div className="pf-v6-u-danger-color-100">{actionError}</div> : null}
        </ModalBody>
        <ModalFooter>
          <Button
            key="confirm-unban"
            variant="primary"
            type="button"
            isLoading={selectedPlayerForUnban ? Boolean(actionInProgressByPlayer[selectedPlayerForUnban]) : false}
            onClick={() => void handleConfirmUnban()}
          >
            Unban
          </Button>
          <Button key="cancel-unban" variant="link" type="button" onClick={resetUnbanState}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>
    </>
  );
};

export default BannedPlayersList;
