import React, {useMemo, useState} from 'react';
import {
    Button,
    Card,
    CardBody,
    CardHeader,
    EmptyState,
    EmptyStateVariant,
    FormGroup,
    Grid,
    GridItem,
    TextInput,
    Title
} from '@patternfly/react-core';

import type {PlayerJournalEntryDTO} from '../../service/apiService';

interface PlayerDetailJournalTabProps {
    entries: PlayerJournalEntryDTO[];
    entryText: string;
    onEntryTextChange: (next: string) => void;
    onSaveEntry: () => Promise<void>;
}

const PlayerDetailJournalTab: React.FC<PlayerDetailJournalTabProps> = ({
                                                                       entries,
                                                                       entryText,
                                                                       onEntryTextChange,
                                                                       onSaveEntry
                                                                   }) => {
    const [isSubmitting, setIsSubmitting] = useState(false);

    const sortedEntries = useMemo(
        () => [...entries].sort((a, b) => new Date(b.occurredAt).getTime() - new Date(a.occurredAt).getTime()),
        [entries]
    );

    const submit = async () => {
        if (!entryText.trim() || isSubmitting) return;
        setIsSubmitting(true);
        try {
            await onSaveEntry();
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        <Grid hasGutter className="pf-v6-u-mt-md">
            <GridItem span={6}>
                <Card isCompact>
                    <CardHeader>
                        <Title headingLevel="h2" size="lg">Add journal entry</Title>
                    </CardHeader>
                    <CardBody>
                        <form
                            onSubmit={(e) => {
                                e.preventDefault();
                                void submit();
                            }}
                        >
                            <FormGroup label="Entry" fieldId="journal-entry">
                                <TextInput
                                    id="journal-entry"
                                    aria-label="Journal entry"
                                    value={entryText}
                                    onChange={(_ev, value) => onEntryTextChange(value)}
                                    placeholder="Write an internal staff note..."
                                />
                            </FormGroup>
                            <Button type="submit" variant="primary" isDisabled={!entryText.trim() || isSubmitting}>
                                Save
                            </Button>
                        </form>
                    </CardBody>
                </Card>
            </GridItem>

            <GridItem span={6}>
                <Card isCompact>
                    <CardHeader>
                        <Title headingLevel="h2" size="lg">Journal entries</Title>
                    </CardHeader>
                    <CardBody>
                        {!sortedEntries.length ? (
                            <EmptyState variant={EmptyStateVariant.sm} titleText="No journal entries yet"/>
                        ) : (
                            <div className="playerDetail__historyList">
                                {sortedEntries.map((entry, idx) => (
                                    <div key={`${entry.staffUserId}-${entry.occurredAt}-${idx}`}
                                         className="playerDetail__historyRow">
                                        <div className="playerDetail__historyMain">
                                            <strong>{entry.staffDisplayName}</strong>: {entry.entry}
                                        </div>
                                        <div className="playerDetail__historySub">
                                            {new Date(entry.occurredAt).toLocaleString()}
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}
                    </CardBody>
                </Card>
            </GridItem>
        </Grid>
    );
};

export default PlayerDetailJournalTab;
