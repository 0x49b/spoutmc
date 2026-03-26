import React, {useState} from 'react';
import {
    Button,
    Card,
    CardBody,
    CardHeader,
    EmptyState,
    EmptyStateVariant,
    FormGroup,
    Grid,
    GridItem, Label,
    TextInput,
    Title
} from '@patternfly/react-core';

import type {PlayerChatMessageDTO, PlayerConversationDTO} from '../../service/apiService';
import {LockIcon} from "@patternfly/react-icons";
import {Table, Tbody, Td, Th, Thead, Tr} from "@patternfly/react-table";

interface PlayerDetailMessagesTabProps {
    currentStaffUserID: number | null;

    hasOtherConversations: boolean;
    conversations: PlayerConversationDTO[];
    selectedConversationId: number | null;
    selectedConversation: PlayerConversationDTO | null;

    pendingNewConversation: boolean;
    messages: PlayerChatMessageDTO[];

    composerText: string;
    onComposerTextChange: (next: string) => void;

    canUseMessageComposer: boolean;
    canCloseSelectedConversation: boolean;

    messagesEndRef: React.RefObject<HTMLDivElement | null>;

    onStartNewConversation: () => void;
    onSelectConversation: (conversationId: number) => void;
    onCloseConversation: () => Promise<void>;
    onSendMessage: () => Promise<void>;
}

const sortConversations = (convs: PlayerConversationDTO[]): PlayerConversationDTO[] => {
    const open = convs.filter((c) => !c.closed);
    const closed = convs.filter((c) => c.closed);

    const byLastOccurredDesc = (a: PlayerConversationDTO, b: PlayerConversationDTO) =>
        new Date(b.lastOccurredAt).getTime() - new Date(a.lastOccurredAt).getTime();

    open.sort(byLastOccurredDesc);
    closed.sort(byLastOccurredDesc);

    return [...open, ...closed];
};

const formatLocalDateTime = (timestamp?: string | null): string => {
    if (!timestamp) return '';

    const parsed = new Date(timestamp);
    if (Number.isNaN(parsed.getTime())) {
        return timestamp;
    }

    return parsed.toLocaleString();
};


const PlayerDetailMessagesTab: React.FC<PlayerDetailMessagesTabProps> = ({
                                                                             currentStaffUserID,
                                                                             hasOtherConversations,
                                                                             conversations,
                                                                             selectedConversation,
                                                                             pendingNewConversation,
                                                                             messages,
                                                                             composerText,
                                                                             onComposerTextChange,
                                                                             canUseMessageComposer,
                                                                             canCloseSelectedConversation,
                                                                             messagesEndRef,
                                                                             onStartNewConversation,
                                                                             onSelectConversation,
                                                                             onCloseConversation,
                                                                             onSendMessage
                                                                         }) => {
    const [selectedDataListItemId, setSelectedDataListItemId] = useState<number>();

    const onSelectRow = (id: number) => {
        setSelectedDataListItemId(id);
        onSelectConversation(id)
    };
    return (
        <Grid hasGutter className="pf-v6-u-mt-md">
            <GridItem span={4}>
                <Card>
                    <CardHeader>
                        <div style={{
                            display: 'flex',
                            flexWrap: 'wrap',
                            alignItems: 'flex-start',
                            justifyContent: 'space-between',
                            gap: 8
                        }}>
                            <div>
                                <Title headingLevel="h2" size="lg">Conversations</Title>
                                {hasOtherConversations ? (
                                    <div style={{fontSize: 12, opacity: 0.8, marginTop: 4}}>
                                        Other conversations are hidden (insufficient permission).
                                    </div>
                                ) : null}
                            </div>
                            <Button
                                variant="secondary"
                                size="sm"
                                isDisabled={currentStaffUserID == null}
                                onClick={() => onStartNewConversation()}
                            >
                                Start new conversation
                            </Button>
                        </div>
                    </CardHeader>
                    <CardBody>
                        {!conversations.length ? (
                            <EmptyState variant={EmptyStateVariant.sm}
                                        titleText="No conversations yet"/>
                        ) : (


                            <Table variant="compact">
                                <Thead>
                                    <Tr>
                                        <Th>Staff Member</Th>
                                        <Th>Last Message</Th>
                                        <Th></Th>
                                    </Tr>
                                </Thead>
                                <Tbody>
                                    {sortConversations(conversations).map((conv) => (

                                        <Tr onRowClick={() => onSelectRow(conv.id)} key={conv.id}
                                            isSelectable isClickable
                                            isRowSelected={conv.id === selectedDataListItemId}>
                                            <Td>{conv.staffDisplayName}</Td>
                                            <Td>{formatLocalDateTime(conv.lastOccurredAt)}</Td>
                                            <Td>{conv.closed ? <LockIcon/> : null}</Td>
                                        </Tr>
                                    ))}

                                </Tbody>
                            </Table>
                        )}
                    </CardBody>
                </Card>
            </GridItem>

            <GridItem span={8}>
                <Card>
                    <CardHeader>
                        <div style={{
                            display: 'flex',
                            flexWrap: 'wrap',
                            alignItems: 'center',
                            justifyContent: 'space-between',
                            gap: 8
                        }}>
                            <Title headingLevel="h2" size="lg">Messages</Title>
                            {canCloseSelectedConversation ? (
                                <Button variant="secondary" size="sm"
                                        onClick={() => void onCloseConversation()}>
                                    Close conversation
                                </Button>
                            ) : messages.length === 0 ? null : <Label color={"red"}>Conversation is closed</Label> }
                        </div>
                    </CardHeader>
                    <CardBody>
                        {pendingNewConversation ? (
                            <div className="pf-v6-u-font-size-sm pf-v6-u-mb-md"
                                 style={{opacity: 0.85}}>
                                You are starting a new conversation with this player. Your
                                first
                                message will show them the support-chat notice in-game, then
                                deliver
                                your text.
                            </div>
                        ) : null}

                        <div className="playerDetail__messages">
                            {messages.length ? (
                                messages.map((m, idx) => (
                                    <div
                                        key={`${m.timestamp}-${idx}`}
                                        className={m.direction === 'incoming'
                                            ? 'playerDetail__bubble playerDetail__bubble--incoming'
                                            : 'playerDetail__bubble playerDetail__bubble--outgoing'}
                                    >
                                        <div className="playerDetail__bubbleMeta">
                                            {m.direction === 'outgoing' && m.role ? `[${m.role}] ` : ''}
                                            {m.direction === 'outgoing' && m.sender ? m.sender : null}
                                            <span
                                                className="playerDetail__bubbleTime">{new Date(m.timestamp).toLocaleTimeString()}</span>
                                        </div>
                                        <div
                                            className="playerDetail__bubbleText">{m.message}</div>
                                    </div>
                                ))
                            ) : (
                                <EmptyState
                                    variant={EmptyStateVariant.sm}
                                    titleText={pendingNewConversation ? 'New conversation' : 'No messages yet'}
                                />
                            )}
                            <div ref={messagesEndRef}/>
                        </div>


                        {canUseMessageComposer ?
                        <form
                            onSubmit={(e) => {
                                e.preventDefault();
                                void onSendMessage();
                            }}
                            style={{marginTop: 12, display: 'flex', gap: 8}}
                        >
                            <FormGroup style={{flex: 1, marginBottom: 0}}>
                                <TextInput
                                    aria-label="Message"
                                    placeholder={
                                        !canUseMessageComposer
                                            ? (selectedConversation?.closed
                                                ? 'This conversation is closed'
                                                : 'Select your own open conversation to send')
                                            : pendingNewConversation
                                                ? 'Write your first message…'
                                                : 'Write a message…'
                                    }
                                    isDisabled={!canUseMessageComposer}
                                    value={composerText}
                                    onChange={(_ev, value) => onComposerTextChange(value)}
                                />
                            </FormGroup>
                            <Button isDisabled={!canUseMessageComposer} type="submit"
                                    variant="primary">
                                Send
                            </Button>
                        </form>
                            : null}


                    </CardBody>
                </Card>
            </GridItem>
        </Grid>
    );
};

export default PlayerDetailMessagesTab;

