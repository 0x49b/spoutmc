import React, {useEffect, useState} from 'react';
import {
    Alert,
    Button,
    Modal,
    ModalBody,
    ModalFooter,
    ModalHeader,
    ModalVariant,
    Spinner
} from '@patternfly/react-core';
import {SaveIcon} from '@patternfly/react-icons';
import {CodeEditor, Language} from '@patternfly/react-code-editor';
import {getAuthFetchHeaders} from '../../../service/apiService.ts';

interface FileEditorModalProps {
    isOpen: boolean;
    onClose: () => void;
    filePath: string;
    fileName: string;
    serverId: string;
    volume?: string;
    onSave: (content: string) => Promise<void>;
}

const getLanguageFromFileName = (fileName: string): Language => {
    const ext = fileName.split('.').pop()?.toLowerCase() || '';

    const languageMap: Record<string, Language> = {
        'json': Language.json,
        'yaml': Language.yaml,
        'yml': Language.yaml,
        'xml': Language.xml,
        'js': Language.javascript,
        'ts': Language.typescript,
        'java': Language.java,
        'py': Language.python,
        'sh': Language.shell,
        'html': Language.html,
        'css': Language.css,
        'md': Language.markdown,
        'sql': Language.sql,
        'txt': Language.plaintext,
        'log': Language.plaintext,
        'properties': Language.ini,
        'conf': Language.plaintext,
        'config': Language.plaintext,
        'ini': Language.ini,
        'toml': Language.plaintext,
    };

    return languageMap[ext] || Language.plaintext;
};

const FileEditorModal: React.FC<FileEditorModalProps> = ({
                                                             isOpen,
                                                             onClose,
                                                             filePath,
                                                             fileName,
                                                             serverId,
                                                             volume,
                                                             onSave,
                                                         }) => {
    const [content, setContent] = useState<string>('');
    const [originalContent, setOriginalContent] = useState<string>('');
    const [isLoading, setIsLoading] = useState(true);
    const [isSaving, setIsSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [hasChanges, setHasChanges] = useState(false);

    const language = getLanguageFromFileName(fileName);

    useEffect(() => {
        if (isOpen && serverId && filePath) {
            loadFileContent();
        }
    }, [isOpen, serverId, filePath]);

    const loadFileContent = async () => {
        setIsLoading(true);
        setError(null);

        try {
            const response = await fetch(
                `http://localhost:3000/api/v1/server/${serverId}/file?path=${encodeURIComponent(filePath)}${volume ? `&volume=${encodeURIComponent(volume)}` : ''}`,
                { headers: getAuthFetchHeaders() }
            );

            if (!response.ok) {
                throw new Error('Failed to load file');
            }

            const data = await response.json();
            setContent(data.content);
            setOriginalContent(data.content);
            setHasChanges(false);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load file');
        } finally {
            setIsLoading(false);
        }
    };

    const handleEditorChange = (value: string | undefined) => {
        const newContent = value || '';
        setContent(newContent);
        setHasChanges(newContent !== originalContent);
    };

    const handleSave = async () => {
        setIsSaving(true);
        setError(null);

        try {
            await onSave(content);
            setOriginalContent(content);
            setHasChanges(false);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to save file');
        } finally {
            setIsSaving(false);
        }
    };

    const handleClose = () => {
        if (hasChanges) {
            if (window.confirm('You have unsaved changes. Are you sure you want to close?')) {
                onClose();
            }
        } else {
            onClose();
        }
    };

    return (
        <Modal
            variant={ModalVariant.large}
            title={fileName}
            isOpen={isOpen}
            onClose={handleClose}
        >
            <ModalHeader title={`${filePath}${hasChanges ? ' • Modified' : ''}`} labelId="basic-modal-title"/>
            <ModalBody>
                {error && (
                    <Alert variant="danger" isInline title="Error" className="pf-v6-u-mb-md">
                        {error}
                    </Alert>
                )}

                {isLoading ? (
                    <div style={{
                        display: 'flex',
                        justifyContent: 'center',
                        alignItems: 'center',
                        minHeight: '400px'
                    }}>
                        <Spinner size="xl"/>
                    </div>
                ) : (
                    <>
                        <CodeEditor
                            code={content}
                            onChange={handleEditorChange}
                            language={language}
                            height={'400px'}
                            isLineNumbersVisible
                            isReadOnly={false}
                            isMinimapVisible
                            isLanguageLabelVisible
                            options={{
                                fontSize: 14,
                                scrollBeyondLastLine: false,
                                tabSize: 2,
                                wordWrap: 'on',
                            }}
                        />
                        <div className="pf-v6-u-mt-sm pf-v6-u-font-size-sm pf-v6-u-color-200">
                            Language: <strong>{language}</strong> | {content.split('\n').length} lines
                            | {content.length} characters
                        </div>
                    </>
                )}
            </ModalBody>
            <ModalFooter>
                <Button
                    key="save"
                    variant="primary"
                    onClick={handleSave}
                    isDisabled={!hasChanges || isSaving || isLoading}
                    isLoading={isSaving}
                    icon={<SaveIcon/>}
                >
                    {isSaving ? 'Saving...' : 'Save'}
                </Button>
                <Button key="close" variant="link" onClick={handleClose} isDisabled={isSaving}>
                    Close
                </Button>
            </ModalFooter>
        </Modal>
    );
};

export default FileEditorModal;
