import React, {useEffect, useState} from 'react';
import {
    Alert,
    Bullseye,
    Button,
    Card,
    CardBody,
    EmptyState,
    EmptyStateBody,
    EmptyStateVariant,
    Label,
    Modal,
    ModalBody,
    ModalFooter,
    ModalHeader,
    ModalVariant,
    PageSection,
    Spinner
} from '@patternfly/react-core';
import {PlusIcon} from '@patternfly/react-icons';
import {ActionsColumn, IAction, Table, Tbody, Td, Th, Thead, Tr} from '@patternfly/react-table';
import PageHeader from '../UI/PageHeader';
import {usePluginStore} from '../../store/pluginStore';
import {useAuthStore} from '../../store/authStore';
import PluginFormModal from './PluginFormModal';
import type {RegistryPluginEntry} from '../../types';

const truncateUrl = (u: string, max = 56) => (u.length <= max ? u : `${u.slice(0, max)}…`);

const PluginsList: React.FC = () => {
  const { plugins, fetchPlugins, loading, createPlugin, updatePlugin, deletePlugin } = usePluginStore();
  const { hasRole, hasPermission } = useAuthStore();
  const canManage =
    hasRole('admin') || hasRole('manager') || hasPermission('plugins.manage');

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<RegistryPluginEntry | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<RegistryPluginEntry | null>(null);
  const [deleteError, setDeleteError] = useState('');
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    fetchPlugins();
  }, [fetchPlugins]);

  const openAdd = () => {
    setEditing(null);
    setFormOpen(true);
  };

  const openEdit = (p: RegistryPluginEntry) => {
    if (p.systemManaged) return;
    setEditing(p);
    setFormOpen(true);
  };

  const handleFormSubmit = async (values: {
    name: string;
    url: string;
    description: string;
    serverNames: string[];
  }) => {
    if (editing) {
      await updatePlugin(editing.id, values);
    } else {
      await createPlugin(values);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget || deleteTarget.systemManaged) return;
    setDeleting(true);
    setDeleteError('');
    try {
      await deletePlugin(deleteTarget.id);
      setDeleteTarget(null);
    } catch (e: unknown) {
      const ax = e as { response?: { status?: number; data?: { error?: string } } };
      if (ax?.response?.status === 409) {
        setDeleteError(ax.response?.data?.error || 'Remove all server assignments first.');
      } else {
        setDeleteError(ax?.response?.data?.error || (e as Error)?.message || 'Delete failed');
      }
    } finally {
      setDeleting(false);
    }
  };

  const getActions = (p: RegistryPluginEntry): IAction[] => {
    if (!canManage || p.systemManaged) return [];
    return [
      { title: 'Edit', onClick: () => openEdit(p) },
      {
        title: 'Delete',
        onClick: () => {
          setDeleteTarget(p);
          setDeleteError('');
        }
      }
    ];
  };

  return (
    <>
      <PageHeader
        title="Plugins"
        description="Registry of plugin JAR URLs applied on container start (itzg PLUGINS). System-managed entries are shipped with SpoutMC."
        actions={
          canManage && (
            <Button variant="primary" icon={<PlusIcon />} onClick={openAdd}>
              Add plugin
            </Button>
          )
        }
      />
      <PageSection>
        <Card>
          <CardBody>
            {loading && plugins.length === 0 ? (
              <Bullseye>
                <Spinner />
              </Bullseye>
            ) : plugins.length === 0 ? (
              <EmptyState variant={EmptyStateVariant.sm} titleText="No plugins in registry">
                <EmptyStateBody>
                  {canManage ? 'Add a plugin JAR URL and assign it to servers.' : 'Nothing to show yet.'}
                </EmptyStateBody>
              </EmptyState>
            ) : (
              <Table aria-label="Plugin registry" variant="compact">
                <Thead>
                  <Tr>
                    <Th>Name</Th>
                    <Th>URL</Th>
                    <Th>Servers</Th>
                    <Th>Source</Th>
                    {canManage && <Th />}
                  </Tr>
                </Thead>
                <Tbody>
                  {plugins.map((p) => (
                    <Tr key={p.id}>
                      <Td dataLabel="Name">
                        <strong>{p.name}</strong>
                        {p.description && (
                          <div className="pf-v6-u-font-size-sm pf-v6-u-color-200">{p.description}</div>
                        )}
                      </Td>
                      <Td dataLabel="URL">
                        <span title={p.url}>{truncateUrl(p.url)}</span>
                      </Td>
                      <Td dataLabel="Servers">
                        {p.serverNames.length === 0 ? (
                          <span className="pf-v6-u-color-200">—</span>
                        ) : (
                          p.serverNames.map((n) => (
                            <Label key={n} isCompact color="blue" className="pf-v6-u-mr-xs pf-v6-u-mb-xs">
                              {n}
                            </Label>
                          ))
                        )}
                      </Td>
                      <Td dataLabel="Source">
                        {p.systemManaged ? (
                          <Label color="purple">System-managed</Label>
                        ) : (
                          <Label color="green">User</Label>
                        )}
                        {p.systemManaged && p.kinds && p.kinds.length > 0 && (
                          <div className="pf-v6-u-font-size-sm pf-v6-u-color-200 pf-v6-u-mt-xs">
                            Kinds: {p.kinds.join(', ')}
                          </div>
                        )}
                      </Td>
                      {canManage && (
                        <Td isActionCell>
                          <ActionsColumn items={getActions(p)} />
                        </Td>
                      )}
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            )}
          </CardBody>
        </Card>
      </PageSection>

      <PluginFormModal
        isOpen={formOpen}
        onClose={() => setFormOpen(false)}
        plugin={editing}
        onSubmit={handleFormSubmit}
      />

      {deleteTarget && (
        <Modal
          variant={ModalVariant.small}
          isOpen={!!deleteTarget}
          onClose={() => {
            if (!deleting) setDeleteTarget(null);
          }}
        >
          <ModalHeader title="Delete plugin" titleIconVariant="warning" />
          <ModalBody>
            {deleteError && (
              <Alert variant="danger" title={deleteError} className="pf-v6-u-mb-md" />
            )}
            <p>
              Delete <strong>{deleteTarget.name}</strong>? Remove all server assignments first, or the
              delete will be rejected.
            </p>
          </ModalBody>
          <ModalFooter>
            <Button variant="danger" onClick={handleDelete} isLoading={deleting}>
              {deleting ? 'Deleting…' : 'Delete'}
            </Button>
            <Button
              variant="link"
              onClick={() => {
                setDeleteTarget(null);
                setDeleteError('');
              }}
              isDisabled={deleting}
            >
              Cancel
            </Button>
          </ModalFooter>
        </Modal>
      )}
    </>
  );
};

export default PluginsList;
