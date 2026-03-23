import React, { useEffect, useState } from 'react';
import {
  Button,
  Form,
  FormGroup,
  Modal,
  ModalVariant,
  TextArea,
  TextInput,
  Checkbox,
  FormHelperText
} from '@patternfly/react-core';
import { useServerStore } from '../../store/serverStore';
import type { RegistryPluginEntry } from '../../types';

export interface PluginFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  /** Set for edit mode; omit for add. System-managed entries must not be passed here. */
  plugin: RegistryPluginEntry | null;
  onSubmit: (values: {
    name: string;
    url: string;
    description: string;
    serverNames: string[];
  }) => Promise<void>;
}

const PluginFormModal: React.FC<PluginFormModalProps> = ({ isOpen, onClose, plugin, onSubmit }) => {
  const { servers, fetchServers } = useServerStore();
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [description, setDescription] = useState('');
  const [selectedServers, setSelectedServers] = useState<Set<string>>(new Set());
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState('');

  useEffect(() => {
    if (isOpen) {
      fetchServers();
    }
  }, [isOpen, fetchServers]);

  useEffect(() => {
    if (!isOpen) return;
    if (plugin) {
      setName(plugin.name);
      setUrl(plugin.url);
      setDescription(plugin.description ?? '');
      setSelectedServers(new Set(plugin.serverNames));
    } else {
      setName('');
      setUrl('');
      setDescription('');
      setSelectedServers(new Set());
    }
    setFormError('');
  }, [isOpen, plugin]);

  const toggleServer = (serverName: string, checked: boolean) => {
    setSelectedServers((prev) => {
      const next = new Set(prev);
      if (checked) next.add(serverName);
      else next.delete(serverName);
      return next;
    });
  };

  const runSubmit = async () => {
    setFormError('');
    if (!name.trim() || !url.trim()) {
      setFormError('Name and URL are required.');
      return;
    }
    setSubmitting(true);
    try {
      await onSubmit({
        name: name.trim(),
        url: url.trim(),
        description: description.trim(),
        serverNames: Array.from(selectedServers)
      });
      onClose();
    } catch (err: unknown) {
      const ax = err as { response?: { data?: { error?: string } } };
      setFormError(ax?.response?.data?.error || (err as Error)?.message || 'Request failed');
    } finally {
      setSubmitting(false);
    }
  };

  const title = plugin ? 'Edit plugin' : 'Add plugin';

  return (
    <Modal
      variant={ModalVariant.medium}
      title={title}
      isOpen={isOpen}
      onClose={onClose}
      actions={[
        <Button key="cancel" variant="link" onClick={onClose}>
          Cancel
        </Button>,
        <Button key="submit" variant="primary" isLoading={submitting} onClick={runSubmit}>
          {plugin ? 'Save' : 'Create'}
        </Button>
      ]}
    >
      <Form
        onSubmit={(e) => {
          e.preventDefault();
          runSubmit();
        }}
      >
        {formError && (
          <FormHelperText>
            <span style={{ color: 'var(--pf-v6-global--danger-color--100)' }}>{formError}</span>
          </FormHelperText>
        )}
        <FormGroup label="Name" isRequired fieldId="plugin-name">
          <TextInput
            id="plugin-name"
            value={name}
            onChange={(_e, v) => setName(v)}
            isRequired
          />
        </FormGroup>
        <FormGroup
          label="JAR URL"
          isRequired
          fieldId="plugin-url"
          helperText="HTTPS URL to a plugin .jar (itzg PLUGINS)."
        >
          <TextInput
            id="plugin-url"
            type="url"
            value={url}
            onChange={(_e, v) => setUrl(v)}
            isRequired
          />
        </FormGroup>
        <FormGroup label="Description" fieldId="plugin-desc">
          <TextArea id="plugin-desc" value={description} onChange={(_e, v) => setDescription(v)} rows={2} />
        </FormGroup>
        <FormGroup label="Apply to servers" fieldId="plugin-servers">
          {servers.length === 0 ? (
            <FormHelperText>No servers in configuration. Add a server first.</FormHelperText>
          ) : (
            servers.map((s) => (
              <Checkbox
                key={s.id}
                id={`srv-${s.id}`}
                label={`${s.name} (${s.type})`}
                isChecked={selectedServers.has(s.name)}
                onChange={(_e, checked) => toggleServer(s.name, Boolean(checked))}
              />
            ))
          )}
        </FormGroup>
      </Form>
    </Modal>
  );
};

export default PluginFormModal;
