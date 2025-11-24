import React, {useEffect, useState} from 'react';
import {
    Alert,
    Button,
    Form,
    FormGroup,
    Grid,
    GridItem,
    Icon,
    Modal,
    ModalBody,
    ModalFooter,
    ModalHeader,
    ModalVariant,
    Spinner,
    TextInput
} from '@patternfly/react-core';
import {PlusIcon, TimesIcon} from '@patternfly/react-icons';
import {Server} from '../../../types';
import * as api from '../../../service/apiService';

interface EditServerModalProps {
    isOpen: boolean;
    onClose: () => void;
    server: Server;
    onUpdate: (serverId: string, data: {
        name?: string;
        env?: Record<string, string>
    }) => Promise<void>;
}

const EditServerModal: React.FC<EditServerModalProps> = ({
                                                             isOpen,
                                                             onClose,
                                                             server,
                                                             onUpdate,
                                                         }) => {
    const [name, setName] = useState('');
    const [envVars, setEnvVars] = useState<Array<{ key: string; value: string }>>([]);
    const [isLoading, setIsLoading] = useState(false);
    const [isLoadingEnv, setIsLoadingEnv] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (isOpen && server) {
            setName(server.name);
            setError(null);
            loadServerEnv();
        }
    }, [isOpen, server]);

    const loadServerEnv = async () => {
        if (!server?.id) return;

        setIsLoadingEnv(true);
        try {
            const response = await api.getServerEnv(server.id);
            const envMap = response.data;

            const envArray = Object.entries(envMap).map(([key, value]) => ({
                key,
                value: String(value),
            }));

            setEnvVars(envArray);
        } catch (err) {
            console.error('Failed to load environment variables:', err);
            setEnvVars([]);
        } finally {
            setIsLoadingEnv(false);
        }
    };

    const handleAddEnvVar = () => {
        setEnvVars([...envVars, {key: '', value: ''}]);
    };

    const handleRemoveEnvVar = (index: number) => {
        const newEnvVars = envVars.filter((_, i) => i !== index);
        setEnvVars(newEnvVars);
    };

    const handleEnvVarChange = (index: number, field: 'key' | 'value', value: string) => {
        const newEnvVars = [...envVars];
        newEnvVars[index][field] = value;
        setEnvVars(newEnvVars);
    };

    const handleSubmit = async () => {
        setError(null);

        if (!name.trim()) {
            setError('Server name is required');
            return;
        }

        const keys = envVars.map(ev => ev.key.trim()).filter(k => k);
        if (keys.length !== new Set(keys).size) {
            setError('Duplicate environment variable keys are not allowed');
            return;
        }

        setIsLoading(true);

        try {
            const envObj: Record<string, string> = {};
            envVars.forEach(ev => {
                if (ev.key.trim()) {
                    envObj[ev.key.trim()] = ev.value;
                }
            });

            const updateData: { name?: string; env?: Record<string, string> } = {};

            if (name !== server.name) {
                updateData.name = name;
            }

            if (Object.keys(envObj).length > 0) {
                updateData.env = envObj;
            }

            await onUpdate(server.id, updateData);
            onClose();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to update server');
        } finally {
            setIsLoading(false);
        }
    };

    return (
        <Modal
            variant={ModalVariant.medium}
            title="Edit Server"
            isOpen={isOpen}
            onClose={onClose}
        >
            <ModalHeader title="Edit Server"></ModalHeader>
            <ModalBody>
                <Form>
                    {error && (
                        <Alert variant="danger" isInline title="Error" className="pf-v6-u-mb-md">
                            {error}
                        </Alert>
                    )}

                    <FormGroup label="Server Name" isRequired fieldId="server-name">
                        <TextInput
                            id="server-name"
                            value={name}
                            onChange={(_event, value) => setName(value)}
                            isRequired
                            isDisabled={isLoading}
                        />
                    </FormGroup>

                    <FormGroup
                        label="Environment Variables"
                        fieldId="env-vars"
                    >
                        {isLoadingEnv ? (
                            <div style={{
                                display: 'flex',
                                justifyContent: 'center',
                                padding: '2rem'
                            }}>
                                <Spinner size="lg"/>
                            </div>
                        ) : envVars.length === 0 ? (
                            <Alert variant="info" isInline title="No custom variables">
                                No custom environment variables. System-managed variables are
                                hidden.
                            </Alert>
                        ) : (

                            <Grid hasGutter component="ul">
                                {envVars.map((envVar, index) => (
                                    <Grid hasGutter>
                                        <GridItem span={4}>
                                            <TextInput
                                                value={envVar.key}
                                                onChange={(_event, value) => handleEnvVarChange(index, 'key', value)}
                                                placeholder="KEY"
                                                isDisabled={isLoading}
                                                style={{flex: 1}}
                                            />
                                        </GridItem>
                                        <GridItem span={4}>
                                            <TextInput
                                                value={envVar.value}
                                                onChange={(_event, value) => handleEnvVarChange(index, 'value', value)}
                                                placeholder="value"
                                                isDisabled={isLoading}
                                                style={{flex: 1}}
                                            />
                                        </GridItem>
                                        <GridItem span={1}>
                                            <Button
                                                onClick={() => handleRemoveEnvVar(index)}
                                                isDisabled={isLoading} variant="plain"
                                                aria-label="Action"
                                                icon={<Icon status="danger"><TimesIcon/></Icon>}/>
                                        </GridItem>
                                    </Grid>
                                ))}
                            </Grid>
                        )}

                        <Button
                            variant="link"
                            icon={<PlusIcon/>}
                            onClick={handleAddEnvVar}
                            isDisabled={isLoading || isLoadingEnv}
                            className="pf-v6-u-mt-sm"
                        >
                            Add Variable
                        </Button>

                        <Alert variant="info" isInline title="Note" className="pf-v6-u-mt-md">
                            System-managed variables (TYPE, EULA, ONLINE_MODE, etc.) are hidden and
                            managed automatically.
                            Changes will require a server restart.
                        </Alert>
                    </FormGroup>
                </Form>
            </ModalBody>
            <ModalFooter>
                <Button
                    key="update"
                    variant="primary"
                    onClick={handleSubmit}
                    isLoading={isLoading}
                    isDisabled={isLoading}
                >
                    Update Server
                </Button>
                <Button key="cancel" variant="link" onClick={onClose} isDisabled={isLoading}>
                    Cancel
                </Button>
            </ModalFooter>
        </Modal>
    );
};

export default EditServerModal;
