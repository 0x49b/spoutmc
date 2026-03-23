import React, {useEffect, useState} from 'react';
import {
    ActionList,
    ActionListItem,
    Alert,
    Button,
    Form,
    FormGroup,
    FormSelect,
    FormSelectOption,
    Grid,
    GridItem,
    Icon,
    Modal,
    ModalBody,
    ModalFooter,
    ModalHeader,
    ModalVariant,
    Radio,
    TextInput
} from '@patternfly/react-core';
import {PlusIcon, ServerIcon, TimesIcon} from '@patternfly/react-icons';
import {Server} from '../../../types';

interface EnvVar {
    key: string;
    value: string;
}

type ServerType = 'proxy' | 'lobby' | 'game';

interface AddServerModalProps {
    isOpen: boolean;
    onClose: () => void;
    servers: Server[];
    onAdd: (serverData: {
        name: string;
        image: string;
        port?: number;
        proxy?: boolean;
        lobby?: boolean;
        env: Record<string, string>;
    }) => Promise<void>;
}

const AddServerModal: React.FC<AddServerModalProps> = ({
                                                           isOpen,
                                                           onClose,
                                                           servers,
                                                           onAdd
                                                       }) => {
    const [loading, setLoading] = useState(false);
    const [serverType, setServerType] = useState<ServerType>('game');
    const [formData, setFormData] = useState({
        name: '',
        port: 25565
    });
    const [envVars, setEnvVars] = useState<EnvVar[]>([{key: '', value: ''}]);
    const [validationError, setValidationError] = useState<string>('');
    const [availableVersions, setAvailableVersions] = useState<string[]>([]);
    const [selectedVersion, setSelectedVersion] = useState<string>('');

    const getSystemManagedEnvVars = (type: ServerType): string[] => {
        switch (type) {
            case 'proxy':
                return ['TYPE'];
            case 'lobby':
            case 'game':
                return ['EULA', 'TYPE', 'ONLINE_MODE', 'GUI', 'CONSOLE', 'VERSION'];
            default:
                return [];
        }
    };

    const isSystemManaged = (key: string): boolean => {
        const managedVars = getSystemManagedEnvVars(serverType);
        return managedVars.includes(key.toUpperCase().trim());
    };

    const resetForm = () => {
        setFormData({name: '', port: 25565});
        setServerType('game');
        setEnvVars([{key: '', value: ''}]);
        setValidationError('');
        if (availableVersions.length > 0) {
            setSelectedVersion(availableVersions[0]);
        }
    };

    const handleClose = () => {
        resetForm();
        onClose();
    };

    const getDockerImage = (type: ServerType): string => {
        switch (type) {
            case 'proxy':
                return 'itzg/mc-proxy:latest';
            case 'lobby':
            case 'game':
                return 'itzg/minecraft-server:latest';
        }
    };

    const hasProxy = servers.some(server => server.location === 'Proxy');
    const hasLobby = servers.some(server => server.location === 'Lobby');

    useEffect(() => {
        const fetchVersions = async () => {
            try {
                // Fetch versions directly from PaperMC API
                const response = await fetch('https://api.papermc.io/v2/projects/paper');
                const data = await response.json();

                if (data.versions && Array.isArray(data.versions)) {
                    // Reverse to show newest versions first
                    const versions = [...data.versions].reverse();
                    setAvailableVersions(versions);
                    if (versions.length > 0) {
                        setSelectedVersion(versions[0]);
                    }
                }
            } catch (error) {
                console.error('Failed to fetch versions from PaperMC API:', error);
            }
        };

        fetchVersions();
    }, []);

    useEffect(() => {
        if ((serverType === 'proxy' && hasProxy) || (serverType === 'lobby' && hasLobby)) {
            setServerType('game');
            setValidationError('');
        }
    }, [serverType, hasProxy, hasLobby]);

    const validateServerType = (type: ServerType): string | null => {
        if (type === 'proxy' && hasProxy) {
            return 'A proxy server already exists in the network. Only one proxy is allowed.';
        }
        if (type === 'lobby' && hasLobby) {
            return 'A lobby server already exists in the network. Only one lobby is allowed.';
        }
        return null;
    };

    const handleSubmit = async () => {
        setValidationError('');

        const typeError = validateServerType(serverType);
        if (typeError) {
            setValidationError(typeError);
            return;
        }

        setLoading(true);

        try {
            const env: Record<string, string> = {};
            envVars.forEach(({key, value}) => {
                if (key.trim()) {
                    env[key.trim()] = value;
                }
            });

            if (serverType === 'lobby' || serverType === 'game') {
                if (selectedVersion) {
                    env['VERSION'] = selectedVersion;
                }
            }

            const serverData: any = {
                name: formData.name,
                image: getDockerImage(serverType),
                env
            };

            if (serverType === 'proxy') {
                serverData.proxy = true;
                serverData.port = Number(formData.port);
            } else if (serverType === 'lobby') {
                serverData.lobby = true;
            }

            await onAdd(serverData);
            handleClose();
        } catch (error) {
            console.error('Failed to add server:', error);
            alert('Failed to add server. Please try again.');
        } finally {
            setLoading(false);
        }
    };
    const handleEnvVarChange = (index: number, field: 'key' | 'value', value: string) => {
        const newEnvVars = [...envVars];
        newEnvVars[index][field] = value;
        setEnvVars(newEnvVars);
    };

    const addEnvVar = () => {
        setEnvVars([...envVars, {key: '', value: ''}]);
    };

    const removeEnvVar = (index: number) => {
        if (envVars.length > 1) {
            setEnvVars(envVars.filter((_, i) => i !== index));
        }
    };

    return (
        <Modal
            variant={ModalVariant.large}
            isOpen={isOpen}
            onClose={handleClose}
        >
            <ModalHeader title="Add New Server"/>
            <ModalBody>
                <Form>
                    {validationError && (
                        <Alert variant="danger" isInline title="Validation Error"
                               className="pf-v6-u-mb-md">
                            {validationError}
                        </Alert>
                    )}
                    <Alert
                        variant="warning"
                        isInline
                        title="Proxy restart notice"
                        className="pf-v6-u-mb-md"
                    >
                        Adding a non-proxy server updates <code>velocity.toml</code> and automatically restarts
                        the proxy. Connected players may be disconnected briefly.
                    </Alert>

                    <FormGroup label="Server Name" isRequired fieldId="server-name">
                        <TextInput
                            id="server-name"
                            name="name"
                            value={formData.name}
                            onChange={(_event, value) => setFormData(prev => ({
                                ...prev,
                                name: value
                            }))}
                            placeholder="Enter server name"
                            isRequired
                        />
                    </FormGroup>

                    <FormGroup label="Server Type" isRequired fieldId="server-type">
                        <Radio
                            id="proxy-type"
                            name="serverType"
                            label="Proxy Server"
                            description={hasProxy ? 'Already exists - only one proxy allowed' : 'Main entry point for the network (only one allowed)'}
                            isChecked={serverType === 'proxy'}
                            onChange={() => setServerType('proxy')}
                            isDisabled={hasProxy}
                        />
                        <Radio
                            id="lobby-type"
                            name="serverType"
                            label="Lobby Server"
                            description={hasLobby ? 'Already exists - only one lobby allowed' : 'Central hub where players spawn (only one allowed)'}
                            isChecked={serverType === 'lobby'}
                            onChange={() => setServerType('lobby')}
                            isDisabled={hasLobby}
                        />
                        <Radio
                            id="game-type"
                            name="serverType"
                            label="Game Server"
                            description="Regular game server (unlimited)"
                            isChecked={serverType === 'game'}
                            onChange={() => setServerType('game')}
                        />
                    </FormGroup>

                    {(serverType === 'lobby' || serverType === 'game') && (
                        <FormGroup label="Version" isRequired fieldId="version">
                            <FormSelect
                                id="version"
                                value={selectedVersion}
                                onChange={(_event, value) => setSelectedVersion(value as string)}
                                isRequired
                            >
                                {availableVersions.length === 0 ? (
                                    <FormSelectOption value="" label="No versions available"/>
                                ) : (
                                    availableVersions.map((version) => (
                                        <FormSelectOption key={version} value={version}
                                                          label={version}/>
                                    ))
                                )}
                            </FormSelect>
                            <p className="pf-v6-u-font-size-sm pf-v6-u-color-200 pf-v6-u-mt-xs">
                                Minecraft server version (automatically added as VERSION env var)
                            </p>
                        </FormGroup>
                    )}

                    {serverType === 'proxy' && (
                        <FormGroup label="Port" isRequired fieldId="port">
                            <TextInput
                                id="port"
                                name="port"
                                type="number"
                                value={formData.port}
                                onChange={(_event, value) => setFormData(prev => ({
                                    ...prev,
                                    port: parseInt(value) || 25565
                                }))}
                                min={1}
                                max={65535}
                                isRequired
                            />
                            <p className="pf-v6-u-font-size-sm pf-v6-u-color-200 pf-v6-u-mt-xs">
                                Port for external connections to the proxy server
                            </p>
                        </FormGroup>
                    )}

                    <FormGroup label="Environment Variables" fieldId="env-vars">
                        {envVars.map((envVar, index) => {
                            const isManagedVar = envVar.key && isSystemManaged(envVar.key);
                            return (
                                <div key={index} className="pf-v6-u-mb-sm">
                                    <Grid>


                                        <ActionList>
                                            <ActionListItem>
                                                <div style={{
                                                    display: 'flex',
                                                    gap: 'var(--pf-v6-global--spacer--sm)',
                                                    width: '100%'
                                                }}>
                                                    <GridItem>
                                                        <Grid hasGutter>
                                                            <GridItem span={4}>
                                                                <TextInput
                                                                    value={envVar.key}
                                                                    onChange={(_event, value) => handleEnvVarChange(index, 'key', value)}
                                                                    placeholder="KEY"
                                                                    style={{flex: 1}}
                                                                    validated={isManagedVar ? 'warning' : 'default'}
                                                                />
                                                            </GridItem>
                                                            <GridItem span={4}>
                                                                <TextInput
                                                                    value={envVar.value}
                                                                    onChange={(_event, value) => handleEnvVarChange(index, 'value', value)}
                                                                    placeholder="value"
                                                                    style={{flex: 1}}
                                                                    validated={isManagedVar ? 'warning' : 'default'}
                                                                />
                                                            </GridItem>
                                                            {envVars.length > 1 && (
                                                                <GridItem span={1}>
                                                                    <Button
                                                                        onClick={() => removeEnvVar(index)}
                                                                        variant="plain"
                                                                        icon={<Icon status="danger"><TimesIcon/></Icon>}/>
                                                                </GridItem>
                                                            )}
                                                        </Grid>
                                                    </GridItem>
                                                </div>
                                            </ActionListItem>
                                        </ActionList>
                                    </Grid>
                                    {isManagedVar && (
                                        <Alert variant="warning" isInline isPlain
                                               title="System-managed variable"
                                               className="pf-v6-u-mt-xs">
                                            This variable is system-managed. Your value will
                                            override the default.
                                        </Alert>
                                    )}
                                </div>
                            );
                        })}

                        <Button
                            variant="link"
                            icon={<PlusIcon/>}
                            onClick={addEnvVar}
                            className="pf-v6-u-mt-sm"
                        >
                            Add Environment Variable
                        </Button>

                        {getSystemManagedEnvVars(serverType).length > 0 && (
                            <Alert variant="info" isInline
                                   title={`System-Managed Variables for ${serverType === 'proxy' ? 'Proxy' : serverType === 'lobby' ? 'Lobby' : 'Game'} Server`}
                                   className="pf-v6-u-mt-md">
                                <p>{getSystemManagedEnvVars(serverType).join(', ')}</p>
                                <p className="pf-v6-u-mt-xs">These are automatically configured. Add
                                    them manually only if you need custom values.</p>
                            </Alert>
                        )}
                    </FormGroup>
                </Form>
            </ModalBody>
            <ModalFooter>
                <Button
                    key="add"
                    variant="primary"
                    onClick={handleSubmit}
                    isLoading={loading}
                    isDisabled={loading}
                    icon={<ServerIcon/>}
                >
                    {loading ? 'Adding...' : 'Add Server'}
                </Button>
                <Button key="cancel" variant="link" onClick={handleClose}>
                    Cancel
                </Button>
            </ModalFooter>
        </Modal>
    );
};

export default AddServerModal;
