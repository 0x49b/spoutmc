import { useEffect, useRef, useState, useCallback } from 'react';
import { TextInput, Button, Flex, FlexItem, Alert, Menu, MenuContent, MenuList, MenuItem } from '@patternfly/react-core';
import { PaperPlaneIcon } from '@patternfly/react-icons';
import { LogViewer, LogViewerSearch } from '@patternfly/react-log-viewer';
import AnsiToHtml from 'ansi-to-html';
import ConsoleTabSkeleton from './ConsoleTabSkeleton';

interface ConsoleTabProps {
    /** Container/server ID */
    containerId: string;
    /** URL for SSE logs stream */
    logsUrl: string;
    /** URL for sending commands (omit or set enableSendCommand=false to hide command input) */
    commandUrl?: string;
    isActive: boolean;
    /** When false, hides the send command input (e.g. for infrastructure containers) */
    enableSendCommand?: boolean;
}

// Java Edition commands (without leading slash) for command-name completion.
const JAVA_EDITION_COMMANDS = [
    '?', 'advancement', 'attribute', 'ban', 'ban-ip', 'banlist', 'bossbar', 'clear',
    'clone', 'damage', 'data', 'datapack', 'debug', 'defaultgamemode', 'deop', 'dialog',
    'difficulty', 'effect', 'enchant', 'execute', 'experience', 'fetchprofile', 'fill',
    'fillbiome', 'forceload', 'function', 'gamemode', 'gamerule', 'give', 'help', 'item',
    'jfr', 'kick', 'kill', 'list', 'locate', 'loot', 'me', 'msg', 'op', 'pardon',
    'pardon-ip', 'particle', 'perf', 'place', 'playsound', 'publish', 'random', 'recipe',
    'reload', 'return', 'ride', 'rotate', 'save-all', 'save-off', 'save-on', 'say',
    'schedule', 'scoreboard', 'seed', 'setblock', 'setidletimeout', 'setworldspawn',
    'spawnpoint', 'spectate', 'spreadplayers', 'stop', 'stopsound', 'summon', 'swing',
    'tag', 'team', 'teammsg', 'teleport', 'tell', 'tellraw', 'test', 'tick', 'time',
    'title', 'tm', 'tp', 'transfer', 'trigger', 'version', 'w', 'waypoint', 'weather',
    'whitelist', 'worldborder', 'xp'
];

const getCommandToken = (rawInput: string) => {
    if (!rawInput.startsWith('/')) {
        return { base: '', remainder: '', hasRemainder: false };
    }
    const withoutSlash = rawInput.slice(1);
    const match = withoutSlash.match(/^([^\s]*)([\s\S]*)$/);
    const base = match?.[1] ?? '';
    const remainder = match?.[2] ?? '';
    return { base, remainder, hasRemainder: remainder.trim().length > 0 || remainder.startsWith(' ') };
};

export const ConsoleTab = ({
    containerId,
    logsUrl,
    commandUrl,
    isActive,
    enableSendCommand = true
}: ConsoleTabProps) => {
    const [command, setCommand] = useState('');
    const [logs, setLogs] = useState<string>('');
    const [hasEverLoaded, setHasEverLoaded] = useState(false);
    const [isConnected, setIsConnected] = useState(false);
    const [reconnectAttempts, setReconnectAttempts] = useState(0);
    const [showSuggestions, setShowSuggestions] = useState(false);
    const [filteredCommands, setFilteredCommands] = useState<string[]>([]);
    const [selectedSuggestionIndex, setSelectedSuggestionIndex] = useState(0);
    const eventSourceRef = useRef<EventSource | null>(null);
    const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const inputRef = useRef<HTMLInputElement>(null);
    const ansiConverter = useRef(new AnsiToHtml());
    const maxReconnectAttempts = 10;
    const baseReconnectDelay = 1000; // Start with 1 second

    const connectToLogs = useCallback(() => {
        if (!isActive || !containerId) return;

        // Close existing connection if any
        if (eventSourceRef.current) {
            eventSourceRef.current.close();
            eventSourceRef.current = null;
        }

        const eventSource = new EventSource(logsUrl);

        eventSource.onopen = () => {
            console.log('SSE connection opened');
            setIsConnected(true);
            setReconnectAttempts(0);
            // Show the console even when no log lines arrived yet (idle or quiet container).
            setHasEverLoaded(true);
        };

        eventSource.onmessage = (event) => {
            try {
                // Parse the SSE data which comes as plain text
                const logLine = event.data;

                // Skip empty lines and lines with only ">"
                if (!logLine || logLine.trim() === '>' || logLine.trim() === '' || logLine.trim() === '>....') {
                    return;
                }

                // Convert ANSI codes to HTML
                const convertedHtml = ansiConverter.current.toHtml(logLine);

                setLogs(prev => {
                    const newLog = prev ? `${prev}\n${convertedHtml}` : convertedHtml;
                    // Keep only last 10000 lines to prevent memory issues
                    const lines = newLog.split('\n');
                    if (lines.length > 10000) {
                        return lines.slice(-10000).join('\n');
                    }
                    return newLog;
                });

                setHasEverLoaded(true);
            } catch (error) {
                console.error('Error parsing log message:', error);
            }
        };

        eventSource.onerror = (error) => {
            console.error('SSE connection error:', error);
            setIsConnected(false);
            eventSource.close();

            // Attempt to reconnect with exponential backoff
            if (isActive && reconnectAttempts < maxReconnectAttempts) {
                const delay = Math.min(
                    baseReconnectDelay * Math.pow(2, reconnectAttempts),
                    30000 // Max 30 seconds
                );

                console.log(`Reconnecting in ${delay}ms (attempt ${reconnectAttempts + 1}/${maxReconnectAttempts})`);

                reconnectTimeoutRef.current = setTimeout(() => {
                    setReconnectAttempts(prev => prev + 1);
                    connectToLogs();
                }, delay);
            }
        };

        eventSourceRef.current = eventSource;
    }, [isActive, containerId, logsUrl, reconnectAttempts]);

    useEffect(() => {
        if (isActive && containerId) {
            connectToLogs();
        }

        return () => {
            if (eventSourceRef.current) {
                eventSourceRef.current.close();
                eventSourceRef.current = null;
            }
            if (reconnectTimeoutRef.current) {
                clearTimeout(reconnectTimeoutRef.current);
                reconnectTimeoutRef.current = null;
            }
        };
    }, [isActive, containerId, connectToLogs]);

    // Command-name completion for Java Edition commands.
    useEffect(() => {
        const { base, hasRemainder } = getCommandToken(command);

        // Suggest command names only while typing the command token itself.
        if (!command.startsWith('/') || hasRemainder) {
            setShowSuggestions(false);
            return;
        }

        const searchTerm = base.toLowerCase();
        if (searchTerm.length > 0) {
            const matches = JAVA_EDITION_COMMANDS.filter(cmdName =>
                cmdName.toLowerCase().startsWith(searchTerm)
            );
            setFilteredCommands(matches);
            setShowSuggestions(matches.length > 0);
            setSelectedSuggestionIndex(0);
        } else {
            setFilteredCommands(JAVA_EDITION_COMMANDS);
            setShowSuggestions(true);
            setSelectedSuggestionIndex(0);
        }
    }, [command]);

    const applySuggestion = useCallback((selectedCmd: string) => {
        const { remainder } = getCommandToken(command);
        // Keep already typed remainder if present, otherwise add a trailing space for next arg.
        const suffix = remainder.length > 0 ? remainder : ' ';
        setCommand('/' + selectedCmd + suffix);
        setShowSuggestions(false);
        inputRef.current?.focus();
    }, [command]);

    const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
        if (!showSuggestions || filteredCommands.length === 0) return;

        if (e.key === 'ArrowDown') {
            e.preventDefault();
            setSelectedSuggestionIndex(prev =>
                prev < filteredCommands.length - 1 ? prev + 1 : 0
            );
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            setSelectedSuggestionIndex(prev =>
                prev > 0 ? prev - 1 : filteredCommands.length - 1
            );
        } else if (e.key === 'Tab') {
            if (showSuggestions) {
                e.preventDefault();
                const selectedCmd = filteredCommands[selectedSuggestionIndex];
                if (selectedCmd) {
                    applySuggestion(selectedCmd);
                }
            }
        } else if (e.key === 'Escape') {
            setShowSuggestions(false);
        }
    };

    const acceptSuggestion = (cmd: string) => {
        applySuggestion(cmd);
    };

    const handleCommand = async (e: React.FormEvent) => {
        e.preventDefault();

        // If suggestions are showing and Enter is pressed, accept the suggestion instead
        if (showSuggestions && filteredCommands.length > 0) {
            const selectedCmd = filteredCommands[selectedSuggestionIndex];
            if (selectedCmd) {
                applySuggestion(selectedCmd);
                return;
            }
        }

        if (!command.trim() || !commandUrl) return;

        try {
            const response = await fetch(commandUrl, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ command: command.trim() }),
            });

            if (!response.ok) {
                throw new Error('Failed to send command');
            }

            // Add command to logs
            const timestamp = new Date().toLocaleTimeString();
            const commandLog = `[${timestamp}] > ${command}`;
            setLogs(prev => prev ? `${prev}\n${commandLog}` : commandLog);
            setCommand('');
            setShowSuggestions(false);
        } catch (error) {
            console.error('Error sending command:', error);
            const errorLog = `Error: Failed to send command "${command}"`;
            setLogs(prev => prev ? `${prev}\n${errorLog}` : errorLog);
        }
    };

    if (!hasEverLoaded) {
        return <ConsoleTabSkeleton />;
    }

    return (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            {!isConnected && reconnectAttempts > 0 && (
                <Alert
                    variant="warning"
                    isInline
                    title="Reconnecting to logs..."
                >
                    Attempting to reconnect ({reconnectAttempts}/{maxReconnectAttempts})
                </Alert>
            )}

            <LogViewer
                data={logs}
                height={600}
                theme="dark"
                isTextWrapped
                hasLineNumbers={false}
                toolbar={<LogViewerSearch placeholder="Search logs..." minSearchChars={3} />}
            />

            {enableSendCommand && commandUrl && (
            <form onSubmit={handleCommand} style={{ position: 'relative' }}>
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <div style={{ position: 'relative' }}>
                            <TextInput
                                ref={inputRef}
                                type="text"
                                value={command}
                                onChange={(_event, value) => setCommand(value)}
                                onKeyDown={handleKeyDown}
                                placeholder="Type a command (start with /)"
                                aria-label="Console command input"
                                autoComplete="off"
                            />
                            {showSuggestions && filteredCommands.length > 0 && (
                                <div
                                    style={{
                                        position: 'absolute',
                                        bottom: '100%',
                                        left: 0,
                                        right: 0,
                                        marginBottom: '4px',
                                        maxHeight: '200px',
                                        overflowY: 'auto',
                                        backgroundColor: 'var(--pf-v6-global--BackgroundColor--100)',
                                        border: '1px solid var(--pf-v6-global--BorderColor--100)',
                                        borderRadius: 'var(--pf-v6-global--BorderRadius--sm)',
                                        boxShadow: 'var(--pf-v6-global--BoxShadow--md)',
                                        zIndex: 1000
                                    }}
                                >
                                    <Menu isScrollable>
                                        <MenuContent>
                                            <MenuList>
                                                {filteredCommands.slice(0, 10).map((cmd, index) => (
                                                    <MenuItem
                                                        key={cmd}
                                                        isSelected={index === selectedSuggestionIndex}
                                                        onClick={() => acceptSuggestion(cmd)}
                                                    >
                                                        /{cmd}
                                                    </MenuItem>
                                                ))}
                                            </MenuList>
                                        </MenuContent>
                                    </Menu>
                                </div>
                            )}
                        </div>
                    </FlexItem>
                    <FlexItem>
                        <Button
                            type="submit"
                            variant="primary"
                            icon={<PaperPlaneIcon />}
                            isDisabled={!command.trim() || !isConnected}
                        >
                            Send
                        </Button>
                    </FlexItem>
                </Flex>
            </form>
            )}
        </div>
    );
};

export default ConsoleTab;
