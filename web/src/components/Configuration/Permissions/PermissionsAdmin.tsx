import React, {useCallback, useEffect, useState} from 'react';
import {
    Alert,
    Button,
    Card,
    CardBody,
    Form,
    FormGroup,
    Modal,
    ModalBody,
    ModalFooter,
    ModalHeader,
    ModalVariant,
    PageSection,
    TextInput
} from '@patternfly/react-core';
import {ActionsColumn, IAction, Table, Tbody, Td, Th, Thead, Tr} from '@patternfly/react-table';
import {PlusIcon, TrashIcon} from '@patternfly/react-icons';
import PageHeader from '../../UI/PageHeader';
import * as api from '../../../service/apiService';

const PermissionsAdmin: React.FC = () => {
  const [rows, setRows] = useState<api.PermissionDTO[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [addOpen, setAddOpen] = useState(false);
  const [newKey, setNewKey] = useState('');
  const [newDesc, setNewDesc] = useState('');
  const [editRow, setEditRow] = useState<api.PermissionDTO | null>(null);
  const [editKey, setEditKey] = useState('');
  const [editDesc, setEditDesc] = useState('');
  const [deleteRow, setDeleteRow] = useState<api.PermissionDTO | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const { data } = await api.getPermissions();
      setRows(data);
    } catch (e: unknown) {
      setError((e as Error)?.message || 'Failed to load permissions');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const handleCreate = async () => {
    const key = newKey.trim();
    if (!key) return;
    setLoading(true);
    setError('');
    try {
      await api.createPermission({ key, description: newDesc.trim() });
      setAddOpen(false);
      setNewKey('');
      setNewDesc('');
      await load();
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } };
      setError(err?.response?.data?.error || (e as Error)?.message || 'Failed to create');
    } finally {
      setLoading(false);
    }
  };

  const handleSaveEdit = async () => {
    if (!editRow) return;
    const key = editKey.trim();
    if (!key) return;
    setLoading(true);
    setError('');
    try {
      await api.updatePermission(editRow.id, { key, description: editDesc.trim() });
      setEditRow(null);
      await load();
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } };
      setError(err?.response?.data?.error || (e as Error)?.message || 'Failed to save');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteRow) return;
    setLoading(true);
    setError('');
    try {
      await api.deletePermission(deleteRow.id);
      setDeleteRow(null);
      await load();
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } };
      setError(err?.response?.data?.error || (e as Error)?.message || 'Failed to delete');
    } finally {
      setLoading(false);
    }
  };

  const getActions = (p: api.PermissionDTO): IAction[] => [
    {
      title: 'Edit',
      onClick: () => {
        setEditRow(p);
        setEditKey(p.key);
        setEditDesc(p.description || '');
        setError('');
      }
    },
    {
      title: 'Delete',
      onClick: () => {
        setDeleteRow(p);
        setError('');
      }
    }
  ];

  return (
    <>
      <PageHeader
        title="Permissions management"
        description="Keys stored in the database. Admins can add, edit, or remove entries; you can also change data directly in the database."
        actions={
          <Button variant="primary" icon={<PlusIcon />} onClick={() => { setAddOpen(true); setError(''); }}>
            Add permission
          </Button>
        }
      />
      <PageSection>
        <Card>
          <CardBody>
            {error && !addOpen && !editRow && !deleteRow && (
              <Alert variant="danger" title={error} className="pf-v6-u-mb-md" />
            )}
            <Table aria-label="Permissions" variant="compact">
              <Thead>
                <Tr>
                  <Th>Key</Th>
                  <Th>Description</Th>
                  <Th />
                </Tr>
              </Thead>
              <Tbody>
                {rows.map((p) => (
                  <Tr key={p.id}>
                    <Td dataLabel="Key">
                      <code>{p.key}</code>
                    </Td>
                    <Td dataLabel="Description">{p.description || '—'}</Td>
                    <Td isActionCell>
                      <ActionsColumn items={getActions(p)} />
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
            {rows.length === 0 && !loading && (
              <p style={{ color: 'var(--pf-v6-global--Color--200)' }}>No permission rows yet.</p>
            )}
          </CardBody>
        </Card>
      </PageSection>

      <Modal
        variant={ModalVariant.small}
        isOpen={addOpen}
        onClose={() => { setAddOpen(false); setNewKey(''); setNewDesc(''); setError(''); }}
      >
        <ModalHeader title="Add permission" />
        <ModalBody>
          {error && addOpen && (
            <Alert variant="danger" title={error} className="pf-v6-u-mb-md" />
          )}
          <Form
            id="add-perm-form"
            onSubmit={(e) => {
              e.preventDefault();
              void handleCreate();
            }}
          >
            <FormGroup label="Key" isRequired fieldId="perm-key">
              <TextInput
                id="perm-key"
                value={newKey}
                onChange={(_e, v) => setNewKey(v)}
                placeholder="component.module.action"
              />
            </FormGroup>
            <FormGroup label="Description" fieldId="perm-desc">
              <TextInput
                id="perm-desc"
                value={newDesc}
                onChange={(_e, v) => setNewDesc(v)}
              />
            </FormGroup>
          </Form>
        </ModalBody>
        <ModalFooter>
          <Button
            variant="primary"
            type="submit"
            form="add-perm-form"
            isDisabled={loading || !newKey.trim()}
          >
            Add
          </Button>
          <Button variant="link" onClick={() => { setAddOpen(false); setError(''); }}>
            Cancel
          </Button>
        </ModalFooter>
      </Modal>

      {editRow && (
        <Modal
          variant={ModalVariant.small}
          isOpen={!!editRow}
          onClose={() => { setEditRow(null); setError(''); }}
        >
          <ModalHeader title="Edit permission" />
          <ModalBody>
            {error && (
              <Alert variant="danger" title={error} className="pf-v6-u-mb-md" />
            )}
            <Form
              id="edit-perm-form"
              onSubmit={(e) => {
                e.preventDefault();
                void handleSaveEdit();
              }}
            >
              <FormGroup label="Key" isRequired fieldId="edit-perm-key">
                <TextInput
                  id="edit-perm-key"
                  value={editKey}
                  onChange={(_e, v) => setEditKey(v)}
                />
              </FormGroup>
              <FormGroup label="Description" fieldId="edit-perm-desc">
                <TextInput
                  id="edit-perm-desc"
                  value={editDesc}
                  onChange={(_e, v) => setEditDesc(v)}
                />
              </FormGroup>
            </Form>
          </ModalBody>
          <ModalFooter>
            <Button
              variant="primary"
              type="submit"
              form="edit-perm-form"
              isDisabled={loading || !editKey.trim()}
            >
              Save
            </Button>
            <Button variant="link" onClick={() => setEditRow(null)}>
              Cancel
            </Button>
          </ModalFooter>
        </Modal>
      )}

      {deleteRow && (
        <Modal
          variant={ModalVariant.small}
          isOpen={!!deleteRow}
          onClose={() => { setDeleteRow(null); setError(''); }}
        >
          <ModalHeader title="Delete permission" titleIconVariant="warning" />
          <ModalBody>
            {error && (
              <Alert variant="danger" title={error} className="pf-v6-u-mb-md" />
            )}
            <p>
              Remove <code>{deleteRow.key}</code> from the database? Role and user assignments for this
              permission will be removed.
            </p>
          </ModalBody>
          <ModalFooter>
            <Button variant="danger" icon={<TrashIcon />} onClick={() => void handleDelete()} isDisabled={loading}>
              Delete
            </Button>
            <Button variant="link" onClick={() => setDeleteRow(null)}>
              Cancel
            </Button>
          </ModalFooter>
        </Modal>
      )}
    </>
  );
};

export default PermissionsAdmin;
