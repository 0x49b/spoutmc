import { useEffect, useRef, useState, useCallback } from 'react';
import { TextInput, Button, Flex, FlexItem, Alert } from '@patternfly/react-core';
import { PaperPlaneIcon } from '@patternfly/react-icons';
import { LogViewer, LogViewerSearch } from '@patternfly/react-log-viewer';
import AnsiToHtml from 'ansi-to-html';
import { Server } from '../../../types';
import ConsoleTabSkeleton from './ConsoleTabSkeleton';

interface ConsoleTabProps {
    server: Server;
    isActive: boolean;
}

export const ConsoleTab = ({ server, isActive }: ConsoleTabProps) => {
    const [command, setCommand] = useState('');
    const [logs, setLogs] = useState<string>('');
    const [hasEverLoaded, setHasEverLoaded] = useState(false);
    const [isConnected, setIsConnected] = useState(false);
    const [reconnectAttempts, setReconnectAttempts] = useState(0);
    const eventSourceRef = useRef<EventSource | null>(null);
    const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
    const ansiConverter = useRef(new AnsiToHtml());
    const maxReconnectAttempts = 10;
    const baseReconnectDelay = 1000; // Start with 1 second

    const connectToLogs = useCallback(() => {
        if (!isActive || !server.id) return;

        // Close existing connection if any
        if (eventSourceRef.current) {
            eventSourceRef.current.close();
            eventSourceRef.current = null;
        }

        const eventSource = new EventSource(
            `http://localhost:3000/api/v1/server/${server.id}/logs`
        );

        eventSource.onopen = () => {
            console.log('SSE connection opened');
            setIsConnected(true);
            setReconnectAttempts(0);
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
    }, [isActive, server.id, reconnectAttempts]);

    useEffect(() => {
        if (isActive && server.id) {
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
    }, [isActive, server.id, connectToLogs]);

    const handleCommand = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!command.trim()) return;

        try {
            const response = await fetch(
                `http://localhost:3000/api/v1/server/${server.id}/command`,
                {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ command: command.trim() }),
                }
            );

            if (!response.ok) {
                throw new Error('Failed to send command');
            }

            // Add command to logs
            const timestamp = new Date().toLocaleTimeString();
            const commandLog = `[${timestamp}] > ${command}`;
            setLogs(prev => prev ? `${prev}\n${commandLog}` : commandLog);
            setCommand('');
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
        <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--pf-v6-global--spacer--md)' }}>
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

            <form onSubmit={handleCommand}>
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <TextInput
                            type="text"
                            value={command}
                            onChange={(_event, value) => setCommand(value)}
                            placeholder="Type a command..."
                            aria-label="Console command input"
                        />
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
        </div>
    );
};

export default ConsoleTab;
