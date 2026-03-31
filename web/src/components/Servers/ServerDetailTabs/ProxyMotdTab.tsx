import {
    Alert,
    Button,
    Card,
    CardBody,
    ClipboardCopy,
    Flex,
    FlexItem,
    Grid,
    GridItem,
    Spinner,
    Title
} from '@patternfly/react-core';
import {SaveIcon, UploadIcon} from '@patternfly/react-icons';
import React, {useEffect, useMemo, useRef, useState} from 'react';
import motdDirtBackground from '../../../assets/motd-dirt-background.png';
import packPng from '../../../assets/packpng.svg';
import * as api from '../../../service/apiService.ts';
import {useNotificationStore} from '../../../store/notificationStore.ts';

interface ProxyMotdTabProps {
    serverId: string;
    serverName: string;
    gitOpsEnabled: boolean;
}

const COMMON_SYMBOLS = ['★', '✦', '✧', '▶', '◀', '◆', '»', '«', '|', '•', '✪', '⚔'];
const SERVER_ICON_SIZE = 64;

const escapeTomlBasicString = (value: string): string =>
    value
        .replace(/\\/g, '\\\\')
        .replace(/"/g, '\\"')
        .replace(/\n/g, '\\n')
        .replace(/\r/g, '\\r')
        .replace(/\t/g, '\\t');

const decodeTomlBasicString = (value: string): string =>
    value
        .replace(/\\n/g, '\n')
        .replace(/\\r/g, '\r')
        .replace(/\\t/g, '\t')
        .replace(/\\"/g, '"')
        .replace(/\\\\/g, '\\');

const escapeMiniMessageText = (value: string): string =>
    value
        .replace(/\\/g, '\\\\')
        .replace(/</g, '\\<')
        .replace(/>/g, '\\>');

const parseMotdFromToml = (tomlContent: string): string | null => {
    const motdMatch = tomlContent.match(/^\s*motd\s*=\s*"((?:\\.|[^"\\])*)"/m);
    if (!motdMatch) {
        return null;
    }
    return decodeTomlBasicString(motdMatch[1]);
};

const setTomlMotdValue = (tomlContent: string, motdValue: string): string => {
    const escaped = escapeTomlBasicString(motdValue);
    const motdLine = `motd = "${escaped}"`;
    const motdRegex = /^(\s*motd\s*=\s*)"((?:\\.|[^"\\])*)"/m;

    if (motdRegex.test(tomlContent)) {
        return tomlContent.replace(motdRegex, `${'$1'}"${escaped}"`);
    }

    const lines = tomlContent.split('\n');
    const bindLineIndex = lines.findIndex((line) => /^\s*bind\s*=/.test(line));
    if (bindLineIndex >= 0) {
        lines.splice(bindLineIndex + 1, 0, motdLine);
        return lines.join('\n');
    }

    const firstSectionIndex = lines.findIndex((line) => /^\s*\[/.test(line));
    if (firstSectionIndex >= 0) {
        lines.splice(firstSectionIndex, 0, motdLine, '');
        return lines.join('\n');
    }

    return `${tomlContent.trimEnd()}\n${motdLine}\n`;
};

const normalizeHexColor = (value: string): string | null => {
    const v = value.trim().toLowerCase();
    const hex6 = /^#([0-9a-f]{6})$/;
    const hex3 = /^#([0-9a-f]{3})$/;
    const rgb = /^rgb\(\s*([0-9]{1,3})\s*,\s*([0-9]{1,3})\s*,\s*([0-9]{1,3})\s*\)$/;

    if (hex6.test(v)) return v;
    const hex3Match = v.match(hex3);
    if (hex3Match) {
        const [a, b, c] = hex3Match[1].split('');
        return `#${a}${a}${b}${b}${c}${c}`;
    }
    const rgbMatch = v.match(rgb);
    if (rgbMatch) {
        const r = Math.min(255, Number(rgbMatch[1]));
        const g = Math.min(255, Number(rgbMatch[2]));
        const b = Math.min(255, Number(rgbMatch[3]));
        return `#${r.toString(16).padStart(2, '0')}${g.toString(16).padStart(2, '0')}${b.toString(16).padStart(2, '0')}`;
    }
    return null;
};

const readColorFromElement = (el: HTMLElement): string | null => {
    if (el.tagName.toLowerCase() === 'font') {
        const color = (el as HTMLFontElement).color;
        return color ? normalizeHexColor(color) : null;
    }
    if (el.style.color) {
        return normalizeHexColor(el.style.color);
    }
    return null;
};

const nodeToMiniMessage = (node: Node): string => {
    if (node.nodeType === Node.TEXT_NODE) {
        return escapeMiniMessageText(node.textContent ?? '');
    }
    if (node.nodeType !== Node.ELEMENT_NODE) {
        return '';
    }

    const el = node as HTMLElement;
    const tag = el.tagName.toLowerCase();
    if (tag === 'br') {
        return '\n';
    }

    const wrappers: Array<[string, string]> = [];

    if (tag === 'b' || tag === 'strong' || Number(el.style.fontWeight || 400) >= 600) {
        wrappers.push(['<bold>', '</bold>']);
    }
    if (tag === 'i' || tag === 'em' || el.style.fontStyle === 'italic') {
        wrappers.push(['<italic>', '</italic>']);
    }
    if (tag === 'u' || el.style.textDecoration.includes('underline')) {
        wrappers.push(['<underlined>', '</underlined>']);
    }
    if (tag === 's' || tag === 'strike' || tag === 'del' || el.style.textDecoration.includes('line-through')) {
        wrappers.push(['<strikethrough>', '</strikethrough>']);
    }

    if (el.dataset.mmObfuscated === 'true') {
        wrappers.push(['<obfuscated>', '</obfuscated>']);
    }

    const gradientStart = el.dataset.mmGradientStart;
    const gradientEnd = el.dataset.mmGradientEnd;
    if (gradientStart && gradientEnd) {
        wrappers.push([`<gradient:${gradientStart}:${gradientEnd}>`, '</gradient>']);
    }

    const color = readColorFromElement(el);
    if (color) {
        wrappers.push([`<${color}>`, `</${color}>`]);
    }

    let content = '';
    Array.from(el.childNodes).forEach((child) => {
        content += nodeToMiniMessage(child);
    });

    wrappers.forEach(([open, close]) => {
        content = `${open}${content}${close}`;
    });

    if (tag === 'div' || tag === 'p') {
        return `${content}\n`;
    }

    return content;
};

const htmlToMiniMessage = (html: string): string => {
    const parser = new DOMParser();
    const doc = parser.parseFromString(`<div>${html}</div>`, 'text/html');
    const root = doc.body.firstElementChild;
    if (!root) return '';

    let output = '';
    Array.from(root.childNodes).forEach((child) => {
        output += nodeToMiniMessage(child);
    });
    return output.replace(/\n+$/g, '');
};

const escapeHtml = (text: string): string =>
    text
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');

const normalizeImageToPngIcon = (file: File): Promise<{ dataUrl: string; base64: string }> =>
    new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onerror = () => reject(new Error('Failed to read image file'));
        reader.onload = () => {
            const sourceDataUrl = typeof reader.result === 'string' ? reader.result : '';
            if (!sourceDataUrl) {
                reject(new Error('Invalid image data'));
                return;
            }

            const img = new Image();
            img.onerror = () => reject(new Error('Failed to decode image'));
            img.onload = () => {
                const canvas = document.createElement('canvas');
                canvas.width = SERVER_ICON_SIZE;
                canvas.height = SERVER_ICON_SIZE;
                const ctx = canvas.getContext('2d');
                if (!ctx) {
                    reject(new Error('Canvas context unavailable'));
                    return;
                }

                const scale = Math.max(SERVER_ICON_SIZE / img.width, SERVER_ICON_SIZE / img.height);
                const drawWidth = img.width * scale;
                const drawHeight = img.height * scale;
                const drawX = (SERVER_ICON_SIZE - drawWidth) / 2;
                const drawY = (SERVER_ICON_SIZE - drawHeight) / 2;

                ctx.clearRect(0, 0, SERVER_ICON_SIZE, SERVER_ICON_SIZE);
                ctx.imageSmoothingEnabled = true;
                ctx.drawImage(img, drawX, drawY, drawWidth, drawHeight);

                const pngDataUrl = canvas.toDataURL('image/png');
                const base64 = pngDataUrl.split(',')[1] || '';
                if (!base64) {
                    reject(new Error('Failed to encode PNG image'));
                    return;
                }
                resolve({dataUrl: pngDataUrl, base64});
            };
            img.src = sourceDataUrl;
        };
        reader.readAsDataURL(file);
    });

type MiniMessageParseResult = {
    html: string;
    hadUnsupportedTag: boolean;
    hadMalformedTag: boolean;
};

const normalizeHexForMiniMessage = (value: string): string | null => {
    const normalized = normalizeHexColor(value);
    return normalized ? normalized.toLowerCase() : null;
};

const openTagToHtml = (rawTag: string): {
    key: string;
    openHtml: string;
    closeHtml: string
} | null => {
    const tag = rawTag.trim().toLowerCase();

    if (tag === 'bold') {
        return {key: 'bold', openHtml: '<strong>', closeHtml: '</strong>'};
    }
    if (tag === 'italic') {
        return {key: 'italic', openHtml: '<em>', closeHtml: '</em>'};
    }
    if (tag === 'underlined') {
        return {key: 'underlined', openHtml: '<u>', closeHtml: '</u>'};
    }
    if (tag === 'strikethrough') {
        return {key: 'strikethrough', openHtml: '<s>', closeHtml: '</s>'};
    }
    if (tag === 'obfuscated') {
        return {
            key: 'obfuscated',
            openHtml: '<span data-mm-obfuscated="true" style="letter-spacing: 0.14em;">',
            closeHtml: '</span>'
        };
    }

    const normalizedColor = normalizeHexForMiniMessage(tag);
    if (normalizedColor) {
        return {
            key: `color:${normalizedColor}`,
            openHtml: `<span style="color: ${normalizedColor};">`,
            closeHtml: '</span>'
        };
    }

    const gradientMatch = tag.match(/^gradient:([^:]+):([^:]+)$/);
    if (gradientMatch) {
        const start = normalizeHexForMiniMessage(gradientMatch[1]);
        const end = normalizeHexForMiniMessage(gradientMatch[2]);
        if (!start || !end) {
            return null;
        }
        return {
            key: `gradient:${start}:${end}`,
            openHtml: `<span data-mm-gradient-start="${start}" data-mm-gradient-end="${end}" style="color: transparent; background-image: linear-gradient(90deg, ${start}, ${end}); -webkit-background-clip: text; background-clip: text;">`,
            closeHtml: '</span>'
        };
    }

    return null;
};

const closeTagToKey = (rawTag: string): string | null => {
    const tag = rawTag.trim().toLowerCase();
    if (tag === 'bold' || tag === 'italic' || tag === 'underlined' || tag === 'strikethrough' || tag === 'obfuscated') {
        return tag;
    }

    const normalizedColor = normalizeHexForMiniMessage(tag);
    if (normalizedColor) {
        return `color:${normalizedColor}`;
    }

    if (tag === 'gradient') {
        return 'gradient';
    }

    return null;
};

const miniMessageToEditorHtml = (input: string): MiniMessageParseResult => {
    const stack: Array<{ key: string; closeHtml: string }> = [];
    let html = '';
    let hadUnsupportedTag = false;
    let hadMalformedTag = false;

    let i = 0;
    while (i < input.length) {
        const ch = input[i];

        if (ch === '\\' && i + 1 < input.length) {
            const next = input[i + 1];
            if (next === '<' || next === '>' || next === '\\') {
                html += escapeHtml(next);
                i += 2;
                continue;
            }
        }

        if (ch === '\n') {
            html += '<br>';
            i += 1;
            continue;
        }

        if (ch !== '<') {
            html += escapeHtml(ch);
            i += 1;
            continue;
        }

        const end = input.indexOf('>', i + 1);
        if (end === -1) {
            hadMalformedTag = true;
            html += '&lt;';
            i += 1;
            continue;
        }

        const tagContent = input.slice(i + 1, end).trim();
        const rawToken = input.slice(i, end + 1);

        if (tagContent.startsWith('/')) {
            const key = closeTagToKey(tagContent.slice(1));
            if (!key) {
                hadUnsupportedTag = true;
                html += escapeHtml(rawToken);
                i = end + 1;
                continue;
            }

            if (key === 'gradient') {
                const top = stack[stack.length - 1];
                if (top && top.key.startsWith('gradient:')) {
                    html += top.closeHtml;
                    stack.pop();
                } else {
                    hadMalformedTag = true;
                    html += escapeHtml(rawToken);
                }
                i = end + 1;
                continue;
            }

            const top = stack[stack.length - 1];
            if (top && top.key === key) {
                html += top.closeHtml;
                stack.pop();
            } else {
                hadMalformedTag = true;
                html += escapeHtml(rawToken);
            }
            i = end + 1;
            continue;
        }

        const parsed = openTagToHtml(tagContent);
        if (!parsed) {
            hadUnsupportedTag = true;
            html += escapeHtml(rawToken);
            i = end + 1;
            continue;
        }

        html += parsed.openHtml;
        stack.push({key: parsed.key, closeHtml: parsed.closeHtml});
        i = end + 1;
    }

    while (stack.length > 0) {
        const top = stack.pop();
        if (!top) break;
        hadMalformedTag = true;
        html += top.closeHtml;
    }

    return {html: html || '<br>', hadUnsupportedTag, hadMalformedTag};
};

const wrapSelectionWithSpan = (
    target: HTMLDivElement,
    attributes: Record<string, string>,
    styles?: Partial<CSSStyleDeclaration>
) => {
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0) return;
    const range = selection.getRangeAt(0);
    if (!target.contains(range.commonAncestorContainer) || range.collapsed) return;

    const span = document.createElement('span');
    Object.entries(attributes).forEach(([k, v]) => span.setAttribute(k, v));
    if (styles) {
        Object.entries(styles).forEach(([k, v]) => {
            if (v != null) {
                span.style.setProperty(k.replace(/[A-Z]/g, (m) => `-${m.toLowerCase()}`), String(v));
            }
        });
    }

    const extracted = range.extractContents();
    span.appendChild(extracted);
    range.insertNode(span);

    const newRange = document.createRange();
    newRange.selectNodeContents(span);
    selection.removeAllRanges();
    selection.addRange(newRange);
};

export const ProxyMotdTab: React.FC<ProxyMotdTabProps> = ({
                                                              serverId,
                                                              serverName,
                                                              gitOpsEnabled
                                                          }) => {
    const [editorHtml, setEditorHtml] = useState('');
    const [selectedColor, setSelectedColor] = useState('#09add3');

    const [tomlContent, setTomlContent] = useState('');
    const [isLoading, setIsLoading] = useState(true);
    const [isSaving, setIsSaving] = useState(false);
    const [isUploadingIcon, setIsUploadingIcon] = useState(false);
    const [loadError, setLoadError] = useState<string | null>(null);
    const [parseWarning, setParseWarning] = useState<string | null>(null);
    const [iconError, setIconError] = useState<string | null>(null);
    const [uploadedIconDataUrl, setUploadedIconDataUrl] = useState<string | null>(null);

    const editorRef = useRef<HTMLDivElement | null>(null);
    const iconFileInputRef = useRef<HTMLInputElement | null>(null);
    const pushToast = useNotificationStore((state) => state.pushToast);

    const generatedMotd = useMemo(() => {
        return htmlToMiniMessage(editorHtml).trim();
    }, [editorHtml]);

    const syncEditorState = () => {
        if (editorRef.current) setEditorHtml(editorRef.current.innerHTML);
    };

    const focusActiveEditor = () => {
        if (!editorRef.current) return;
        editorRef.current.focus();
    };

    const runCommand = (command: string, value?: string) => {
        focusActiveEditor();
        document.execCommand('styleWithCSS', false, 'true');
        document.execCommand(command, false, value);
        syncEditorState();
    };

    const applyObfuscated = () => {
        const editor = editorRef.current;
        if (!editor) return;
        focusActiveEditor();
        wrapSelectionWithSpan(editor, {'data-mm-obfuscated': 'true'}, {
            letterSpacing: '0.14em'
        });
        syncEditorState();
    };

    const insertSymbol = (symbol: string) => {
        focusActiveEditor();
        document.execCommand('insertText', false, symbol);
        syncEditorState();
    };

    const loadVelocityToml = async () => {
        setIsLoading(true);
        setLoadError(null);
        setParseWarning(null);

        try {
            const response = await api.getConfigFile(serverId, 'velocity.toml');
            const content = response.data.content;
            setTomlContent(content);

            const currentMotd = parseMotdFromToml(content);
            if (currentMotd === null) {
                setParseWarning('No existing MOTD key found in velocity.toml. A new one will be added on save.');
                setEditorHtml(`<span style="color: #ffffff;">${escapeHtml(serverName || 'Minecraft Server')}</span><br><span style="color: #8a8a8a;">A Minecraft Server</span>`);
                return;
            }

            const parsed = miniMessageToEditorHtml(currentMotd);

            if (parsed.hadUnsupportedTag || parsed.hadMalformedTag) {
                setParseWarning('Some MOTD formatting could not be mapped exactly. Unsupported or malformed tags were kept as text where possible.');
            }
            setEditorHtml(parsed.html || '<br>');
        } catch (error) {
            console.error('Failed to load velocity.toml', error);
            setLoadError('Failed to load velocity.toml for this proxy server.');
        } finally {
            setIsLoading(false);
        }
    };

    useEffect(() => {
        void loadVelocityToml();
    }, [serverId]);

    useEffect(() => {
        if (editorRef.current && editorRef.current.innerHTML !== editorHtml) {
            editorRef.current.innerHTML = editorHtml;
        }
    }, [editorHtml]);

    const handleSave = async () => {
        if (!tomlContent) {
            setLoadError('Cannot save because velocity.toml was not loaded.');
            return;
        }
        if (!generatedMotd.trim()) {
            setLoadError('Generated MOTD is empty. Please add text before saving.');
            return;
        }

        setIsSaving(true);
        setLoadError(null);
        try {
            const updatedToml = setTomlMotdValue(tomlContent, generatedMotd);
            await api.updateConfigFile(serverId, 'velocity.toml', updatedToml);
            setTomlContent(updatedToml);
            pushToast({
                variant: 'success',
                title: 'MOTD saved',
                description: 'Updated velocity.toml successfully.'
            });
        } catch (error) {
            console.error('Failed to save MOTD', error);
            setLoadError('Failed to save MOTD to velocity.toml.');
        } finally {
            setIsSaving(false);
        }
    };

    const handleIconPickClick = () => {
        if (isUploadingIcon) return;
        iconFileInputRef.current?.click();
    };

    const handleIconFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
        const file = event.target.files?.[0];
        event.target.value = '';
        if (!file) return;

        if (!file.type.startsWith('image/')) {
            setIconError('Please select a valid image file.');
            return;
        }

        setIconError(null);
        setIsUploadingIcon(true);

        try {
            const {dataUrl, base64} = await normalizeImageToPngIcon(file);
            await api.updateServerBinaryFile(serverId, 'server-icon.png', base64, '/server');
            setUploadedIconDataUrl(dataUrl);
            pushToast({
                variant: 'success',
                title: 'Proxy icon uploaded',
                description: 'Saved /server/server-icon.png successfully (proxy was not restarted).'
            });
        } catch (error: any) {
            console.error('Failed to upload proxy icon', error);
            if (error?.response?.status === 404) {
                setIconError('Icon upload endpoint is not available on the running API. Restart/update the backend, then try again.');
            } else {
                setIconError('Failed to upload icon. Please try another image.');
            }
        } finally {
            setIsUploadingIcon(false);
        }
    };

    return (
        <Card>
            <CardBody>
                <Title headingLevel="h3" size="lg" className="pf-v6-u-mb-md">Proxy MOTD
                    Generator</Title>
                <div className="pf-v6-u-color-200 pf-v6-u-mb-md">
                    {gitOpsEnabled && ' This file can be overwritten by GitSync. Ensure your repository includes the same MOTD (MOTD Env Var).'}
                </div>

                {parseWarning && (
                    <Alert variant="info" isInline title="MOTD parsing note"
                           className="pf-v6-u-mb-md">
                        {parseWarning}
                    </Alert>
                )}

                {loadError && (
                    <Alert variant="danger" isInline title="Error" className="pf-v6-u-mb-md">
                        {loadError}
                    </Alert>
                )}
                {iconError && (
                    <Alert variant="danger" isInline title="Icon upload error" className="pf-v6-u-mb-md">
                        {iconError}
                    </Alert>
                )}

                {isLoading ? (
                    <div style={{display: 'flex', justifyContent: 'center', padding: '2rem'}}>
                        <Spinner size="xl"/>
                    </div>
                ) : (
                    <>
                        <Grid hasGutter>
                            <GridItem span={12} lg={12}>
                                <Card isCompact>
                                    <CardBody>
                                        <Title headingLevel="h4" size="md"
                                               className="pf-v6-u-mb-sm">MOTD editor (live
                                            preview)</Title>
                                        <Flex spaceItems={{default: 'spaceItemsSm'}}
                                              flexWrap={{default: 'wrap'}}
                                              className="pf-v6-u-mb-sm">
                                            <FlexItem><Button size="sm" variant="control"
                                                              onClick={() => runCommand('bold')}><b>B</b></Button></FlexItem>
                                            <FlexItem><Button size="sm" variant="control"
                                                              onClick={() => runCommand('italic')}><i>I</i></Button></FlexItem>
                                            <FlexItem><Button size="sm" variant="control"
                                                              onClick={() => runCommand('underline')}><u>U</u></Button></FlexItem>
                                            <FlexItem><Button size="sm" variant="control"
                                                              onClick={() => runCommand('strikeThrough')}>
                                                <del>S</del>
                                            </Button></FlexItem>
                                            <FlexItem><Button size="sm" variant="control"
                                                              onClick={applyObfuscated}>Obfuscated</Button></FlexItem>
                                            <FlexItem>
                                                <input
                                                    aria-label="Selected text color"
                                                    type="color"
                                                    value={selectedColor}
                                                    onChange={(event) => {
                                                        const color = event.target.value;
                                                        setSelectedColor(color);
                                                        runCommand('foreColor', color);
                                                    }}
                                                />
                                            </FlexItem>

                                            {COMMON_SYMBOLS.map((symbol) => (
                                                <FlexItem key={symbol}>
                                                    <Button size="sm" variant="control"
                                                            onClick={() => insertSymbol(symbol)}>{symbol}</Button>
                                                </FlexItem>
                                            ))}
                                        </Flex>
                                        <div
                                            className="server-detail-motd-preview pf-v6-u-mb-sm"
                                            style={{backgroundImage: `url(${motdDirtBackground})`}}
                                        >
                                            <button
                                                type="button"
                                                className="server-detail-motd-icon-upload"
                                                onClick={handleIconPickClick}
                                                disabled={isUploadingIcon}
                                                title={isUploadingIcon ? 'Uploading icon...' : 'Upload server icon'}
                                                aria-label="Upload server icon"
                                            >
                                                <img
                                                    src={uploadedIconDataUrl || packPng}
                                                    alt="server icon"
                                                    className="server-detail-motd-default-image"
                                                />
                                                <span className="server-detail-motd-icon-upload-overlay" aria-hidden="true">
                                                    <UploadIcon/>
                                                </span>
                                            </button>
                                            <input
                                                ref={iconFileInputRef}
                                                type="file"
                                                accept="image/png,image/jpeg,image/webp,image/gif"
                                                onChange={handleIconFileChange}
                                                className="server-detail-motd-file-input"
                                            />
                                            <div
                                                className="server-detail-motd-content"
                                            >
                                                <div
                                                    ref={editorRef}
                                                    contentEditable
                                                    suppressContentEditableWarning
                                                    onInput={(event) => setEditorHtml((event.target as HTMLDivElement).innerHTML)}
                                                    onPaste={(event) => {
                                                        event.preventDefault();
                                                        const text = event.clipboardData.getData('text/plain');
                                                        document.execCommand('insertText', false, text);
                                                    }}
                                                    className="server-detail-motd-editor"
                                                />
                                            </div>
                                        </div>
                                        <ClipboardCopy isReadOnly hoverTip="Copy" clickTip="Copied">
                                            {generatedMotd || '<#09add3>A Velocity Server'}
                                        </ClipboardCopy>
                                    </CardBody>
                                </Card>
                            </GridItem>


                        </Grid>

                        <Flex className="pf-v6-u-mt-md" spaceItems={{default: 'spaceItemsSm'}}>
                            <FlexItem>
                                <Button
                                    variant="primary"
                                    icon={<SaveIcon/>}
                                    onClick={handleSave}
                                    isLoading={isSaving}
                                    isDisabled={isLoading || isSaving}
                                >
                                    Save to velocity.toml
                                </Button>
                            </FlexItem>
                        </Flex>
                    </>
                )}
            </CardBody>
        </Card>
    );
};

export default ProxyMotdTab;
