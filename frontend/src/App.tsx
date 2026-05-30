import {ChangeEvent, DragEvent, KeyboardEvent, useState} from 'react';
import './App.css';
import {CopyResultJSON, Decode, DecodeFile, ExportResult, OpenInputFile} from '../wailsjs/go/main/App';
import {main} from '../wailsjs/go/models';

type DecodeRequest = main.DecodeRequest;
type DecodeOptions = main.DecodeOptions;
type DecodeResult = main.DecodeResult;
type OpenFileResult = main.OpenFileResult;
type Part = main.Part;
type SaveFileResult = main.SaveFileResult;
type ValueVariant = main.ValueVariant;
type DroppedFile = File & {path?: string};

type PartRecord = {
    path: string;
    part: Part;
    level: number;
    globalRange: [number, number];
    hasChildren: boolean;
};

const sampleInput = '0a03666f6f';
const defaultSelectedFile = 'No file selected';
const idleStatus = 'Paste payload, tune limits, or drop file to start decoding.';
const largeInputPromptBytes = 5 * 1024 * 1024;

function splitHexBytes(value: string): string[] {
    const bytes: string[] = [];
    for (let index = 0; index+1 < value.length; index += 2) {
        bytes.push(value.slice(index, index + 2));
    }

    return bytes;
}

function applyHexBytes(target: string[], start: number, hexValue: string) {
    const bytes = splitHexBytes(hexValue);
    for (let index = 0; index < bytes.length; index += 1) {
        const targetIndex = start + index;
        if (targetIndex < 0 || targetIndex >= target.length) {
            continue;
        }

        target[targetIndex] = bytes[index];
    }
}

function findBytesHexVariant(part: Part): string {
    return part.value.find((variant) => variant.candidateType === 'bytes.hex')?.displayValue ?? '';
}

function resolveChildBaseOffset(part: Part, globalRange: [number, number]): number {
    const payloadHex = findBytesHexVariant(part);
    const payloadLength = payloadHex.length / 2;
    return globalRange[1] - payloadLength;
}

function buildPartRecords(parts: Part[], parentBaseOffset: number | null = null, level = 0, pathPrefix = ''): PartRecord[] {
    const records: PartRecord[] = [];

    parts.forEach((part, index) => {
        const children = part.children ?? [];
        const path = pathPrefix === '' ? String(index) : `${pathPrefix}.${index}`;
        const globalRange: [number, number] = parentBaseOffset === null
            ? [part.byteRange[0], part.byteRange[1]]
            : [parentBaseOffset + part.byteRange[0], parentBaseOffset + part.byteRange[1]];

        records.push({
            path,
            part,
            level,
            globalRange,
            hasChildren: children.length > 0,
        });

        if (children.length > 0) {
            records.push(...buildPartRecords(children, resolveChildBaseOffset(part, globalRange), level + 1, path));
        }
    });

    return records;
}

function buildRawHexBytes(result: DecodeResult): string[] {
    const bytes = Array.from({length: result.inputSize}, () => '..');

    result.parts.forEach((part) => {
        applyHexBytes(bytes, part.byteRange[0], part.rawHex);
    });

    if (result.leftover !== '') {
        const leftoverBytes = splitHexBytes(result.leftover);
        const start = Math.max(0, result.inputSize - leftoverBytes.length);
        applyHexBytes(bytes, start, result.leftover);
    }

    return bytes;
}

function isRecordVisible(path: string, expandedPaths: Record<string, boolean>): boolean {
    const segments = path.split('.');
    if (segments.length === 1) {
        return true;
    }

    for (let index = 1; index < segments.length; index += 1) {
        const ancestor = segments.slice(0, index).join('.');
        if (!expandedPaths[ancestor]) {
            return false;
        }
    }

    return true;
}

function findFirstPath(parts: Part[]): string | null {
    return parts.length > 0 ? '0' : null;
}

function summarizePart(part: Part): string {
    if (part.value.length === 0) {
        return 'no candidates';
    }

    const lead = part.value[0];
    return `${lead.candidateType}: ${lead.displayValue}`;
}

function formatByteRange(range: [number, number]): string {
    return `[${range[0]}, ${range[1]})`;
}

function formatOffset(value: number): string {
    return value.toString(16).padStart(4, '0');
}

function exportLabel(format: 'json' | 'text'): string {
    return format === 'text' ? 'text report' : 'JSON';
}

function formatByteSize(value: number): string {
    if (value < 1024) {
        return `${value} B`;
    }

    if (value < 1024 * 1024) {
        return `${(value / 1024).toFixed(1)} KiB`;
    }

    return `${(value / (1024 * 1024)).toFixed(1)} MiB`;
}

function confirmLargeInput(sourceLabel: string, size: number, maxBytesLimit: number): boolean {
    if (size <= 0) {
        return true;
    }

    if (size > maxBytesLimit) {
        return window.confirm(
            `${sourceLabel} is ${formatByteSize(size)}, above current MaxBytes ${formatByteSize(maxBytesLimit)}. Decode will fail unless you raise MaxBytes. Continue anyway?`,
        );
    }

    if (size >= largeInputPromptBytes) {
        return window.confirm(
            `${sourceLabel} is ${formatByteSize(size)}. Large inputs can take longer to decode. Continue with current MaxBytes ${formatByteSize(maxBytesLimit)}?`,
        );
    }

    return true;
}

function buildLimitGuidance(messages: string[]): string[] {
    const guidance = new Set<string>();

    messages.forEach((message) => {
        const normalized = message.toLowerCase();

        if (/max_bytes_exceeded|exceeds maxbytes/.test(normalized)) {
            guidance.add('MaxBytes limit hit. Raise MaxBytes for trusted inputs or choose smaller payload.');
        }

        if (/max_fields_exceeded|decoded fields exceeded maxfields/.test(normalized)) {
            guidance.add('MaxFields limit hit. Raise MaxFields only for trusted payloads that are expected to be wide.');
        }

        if (/maxdepth .* reached|nested decode skipped .* maxdepth/.test(normalized)) {
            guidance.add('MaxDepth limit hit. Raise MaxDepth only for trusted payloads with deeper nesting.');
        }
    });

    return Array.from(guidance);
}

const defaultDecodeRequest: DecodeRequest = {
    input: sampleInput,
    inputEncoding: 'auto',
    parseDelimited: false,
    maxDepth: 4,
    maxFields: 256,
    maxBytes: 10 * 1024 * 1024,
};

function App() {
    const [input, setInput] = useState(defaultDecodeRequest.input);
    const [inputEncoding, setInputEncoding] = useState(defaultDecodeRequest.inputEncoding);
    const [parseDelimited, setParseDelimited] = useState(defaultDecodeRequest.parseDelimited);
    const [maxDepth, setMaxDepth] = useState(String(defaultDecodeRequest.maxDepth));
    const [maxFields, setMaxFields] = useState(String(defaultDecodeRequest.maxFields));
    const [maxBytes, setMaxBytes] = useState(String(defaultDecodeRequest.maxBytes));
    const [result, setResult] = useState<DecodeResult | null>(null);
    const [selectedFile, setSelectedFile] = useState(defaultSelectedFile);
    const [errorMessage, setErrorMessage] = useState('');
    const [statusMessage, setStatusMessage] = useState(idleStatus);
    const [isBusy, setIsBusy] = useState(false);
    const [isDragActive, setIsDragActive] = useState(false);
    const [selectedPartPath, setSelectedPartPath] = useState<string | null>(null);
    const [expandedPaths, setExpandedPaths] = useState<Record<string, boolean>>({});

    const partRecords = result ? buildPartRecords(result.parts) : [];
    const selectedRecord = selectedPartPath ? partRecords.find((record) => record.path === selectedPartPath) ?? null : null;
    const rawHexBytes = result ? buildRawHexBytes(result) : [];
    const resultWarnings = result?.warnings ?? [];
    const limitGuidance = buildLimitGuidance([
        errorMessage,
        result?.error ?? '',
        ...resultWarnings,
    ].filter((message) => message !== ''));

    function setDecodeResult(decodeResult: DecodeResult) {
        setResult(decodeResult);
        setSelectedPartPath(findFirstPath(decodeResult.parts));
        setExpandedPaths({});
    }

    function parseLimit(value: string, fallback: number): number {
        const parsedValue = Number.parseInt(value, 10);
        if (!Number.isFinite(parsedValue) || parsedValue <= 0) {
            return fallback;
        }

        return parsedValue;
    }

    function currentDecodeOptions(): DecodeOptions {
        return {
            parseDelimited,
            maxDepth: parseLimit(maxDepth, defaultDecodeRequest.maxDepth),
            maxFields: parseLimit(maxFields, defaultDecodeRequest.maxFields),
            maxBytes: parseLimit(maxBytes, defaultDecodeRequest.maxBytes),
        };
    }

    async function decodeFileAtPath(path: string, sourceLabel: string) {
        try {
            const decodeResult = await DecodeFile(path, currentDecodeOptions());
            setDecodeResult(decodeResult);
            setSelectedFile(path);
            setStatusMessage(`${sourceLabel} decoded (${decodeResult.inputSize} bytes).`);
        } catch (error) {
            const message = error instanceof Error ? error.message : String(error);
            setResult(null);
            setSelectedPartPath(null);
            setExpandedPaths({});
            setErrorMessage(message);
            setStatusMessage(`${sourceLabel} failed.`);
        }
    }

    async function handleDecode() {
        setIsBusy(true);
        setErrorMessage('');
        setSelectedFile(defaultSelectedFile);
        setStatusMessage('Decoding text input...');

        try {
            const decodeRequest: DecodeRequest = {
                input,
                inputEncoding,
                ...currentDecodeOptions(),
            };

            const decodeResult = await Decode(decodeRequest);
            setDecodeResult(decodeResult);
            setStatusMessage(`Decoded text input (${decodeResult.inputSize} bytes).`);
        } catch (error) {
            const message = error instanceof Error ? error.message : String(error);
            setResult(null);
            setSelectedPartPath(null);
            setExpandedPaths({});
            setErrorMessage(message);
            setStatusMessage('Text decode failed.');
        } finally {
            setIsBusy(false);
        }
    }

    async function handleOpenFile() {
        setIsBusy(true);
        setErrorMessage('');
        setStatusMessage('Waiting for file selection...');

        try {
            const dialogResult: OpenFileResult = await OpenInputFile();
            if (dialogResult.cancelled) {
                setSelectedFile('File selection cancelled');
                setStatusMessage('File selection cancelled.');
                return;
            }

            const maxBytesLimit = currentDecodeOptions().maxBytes;
            if (!confirmLargeInput('Selected file', dialogResult.size, maxBytesLimit)) {
                setSelectedFile(dialogResult.path);
                setStatusMessage('Large file decode cancelled. Adjust MaxBytes, then retry if needed.');
                return;
            }

            await decodeFileAtPath(dialogResult.path, 'File');
        } finally {
            setIsBusy(false);
        }
    }

    async function handleCopyJSON() {
        if (!result) {
            return;
        }

        setIsBusy(true);
        setErrorMessage('');
        setStatusMessage('Copying pretty JSON to clipboard...');

        try {
            await CopyResultJSON(result);
            setStatusMessage('Copied pretty JSON to clipboard.');
        } catch (error) {
            const message = error instanceof Error ? error.message : String(error);
            setErrorMessage(message);
            setStatusMessage('Copy JSON failed.');
        } finally {
            setIsBusy(false);
        }
    }

    async function handleExport(format: 'json' | 'text') {
        if (!result) {
            return;
        }

        setIsBusy(true);
        setErrorMessage('');
        setStatusMessage(`Waiting for ${exportLabel(format)} export path...`);

        try {
            const exportResult: SaveFileResult = await ExportResult(result, format);
            if (exportResult.cancelled) {
                setStatusMessage(`${exportLabel(format)} export cancelled.`);
                return;
            }

            setStatusMessage(`Saved ${exportLabel(format)} export to ${exportResult.path}.`);
        } catch (error) {
            const message = error instanceof Error ? error.message : String(error);
            setErrorMessage(message);
            setStatusMessage(`${exportLabel(format)} export failed.`);
        } finally {
            setIsBusy(false);
        }
    }

    function handleInputChange(event: ChangeEvent<HTMLTextAreaElement>) {
        setInput(event.target.value);
    }

    function handleLimitChange(setter: (value: string) => void) {
        return (event: ChangeEvent<HTMLInputElement>) => {
            setter(event.target.value);
        };
    }

    function handleLoadSample() {
        setInput(sampleInput);
        setInputEncoding(defaultDecodeRequest.inputEncoding);
        setParseDelimited(defaultDecodeRequest.parseDelimited);
        setMaxDepth(String(defaultDecodeRequest.maxDepth));
        setMaxFields(String(defaultDecodeRequest.maxFields));
        setMaxBytes(String(defaultDecodeRequest.maxBytes));
        setResult(null);
        setSelectedPartPath(null);
        setExpandedPaths({});
        setErrorMessage('');
        setSelectedFile(defaultSelectedFile);
        setStatusMessage('Sample payload restored.');
    }

    function handleClear() {
        setInput('');
        setInputEncoding(defaultDecodeRequest.inputEncoding);
        setParseDelimited(defaultDecodeRequest.parseDelimited);
        setMaxDepth(String(defaultDecodeRequest.maxDepth));
        setMaxFields(String(defaultDecodeRequest.maxFields));
        setMaxBytes(String(defaultDecodeRequest.maxBytes));
        setResult(null);
        setSelectedPartPath(null);
        setExpandedPaths({});
        setErrorMessage('');
        setSelectedFile(defaultSelectedFile);
        setStatusMessage('Request cleared.');
    }

    function handleRecordToggle(path: string) {
        setExpandedPaths((currentValue) => ({
            ...currentValue,
            [path]: !currentValue[path],
        }));
    }

    function handleRecordSelect(path: string) {
        setSelectedPartPath(path);
    }

    function handleRecordKeyDown(event: KeyboardEvent<HTMLDivElement>, path: string) {
        if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            handleRecordSelect(path);
        }
    }

    function handleDragOver(event: DragEvent<HTMLElement>) {
        event.preventDefault();
        if (isBusy) {
            return;
        }

        setIsDragActive(true);
        setStatusMessage('Drop file to decode with current options.');
    }

    function handleDragLeave(event: DragEvent<HTMLElement>) {
        event.preventDefault();
        if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
            setIsDragActive(false);
            if (!isBusy) {
                setStatusMessage(result ? 'Ready for next decode.' : idleStatus);
            }
        }
    }

    async function handleDrop(event: DragEvent<HTMLElement>) {
        event.preventDefault();
        setIsDragActive(false);

        if (isBusy) {
            return;
        }

        const droppedFile = event.dataTransfer.files.item(0) as DroppedFile | null;
        if (!droppedFile) {
            setStatusMessage('Drop ignored: no file detected.');
            return;
        }

        const droppedPath = droppedFile.path;
        if (!droppedPath) {
            setErrorMessage('Dropped file path unavailable. Use Open and decode file instead.');
            setStatusMessage(`Dropped ${droppedFile.name}, but desktop file path was unavailable.`);
            return;
        }

        const maxBytesLimit = currentDecodeOptions().maxBytes;
        if (!confirmLargeInput(`Dropped file ${droppedFile.name}`, droppedFile.size, maxBytesLimit)) {
            setSelectedFile(`${droppedFile.name} (${formatByteSize(droppedFile.size)})`);
            setStatusMessage('Large file decode cancelled. Adjust MaxBytes, then retry if needed.');
            return;
        }

        setIsBusy(true);
        setErrorMessage('');
        setStatusMessage(`Decoding dropped file ${droppedFile.name}...`);
        await decodeFileAtPath(droppedPath, `Dropped file ${droppedFile.name}`);
        setIsBusy(false);
    }

    return (
        <div
            className={`app-shell${isDragActive ? ' app-shell--drag-active' : ''}`}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
        >
            <section className="hero-card">
                <p className="eyebrow">Story 14 / guarded decode workspace</p>
                <h1>Protobuf Decoder Desktop</h1>
                <p className="intro">
                    Paste hex or base64, choose file, or drop file on window. Backend enforces decode limits, frontend shows loading state, and large inputs get explicit guardrails before decode starts.
                </p>
                <div className="hero-meta">
                    <span className="hero-chip">Local only</span>
                    <span className="hero-chip">Auto / hex / base64</span>
                    <span className="hero-chip">Delimited streams</span>
                </div>
            </section>

            <section className="workspace-grid">
                <article className="panel">
                    <div className="panel-header">
                        <h2>Decode Request</h2>
                        <div className="header-actions">
                            <button className="ghost-button" onClick={handleLoadSample} type="button" disabled={isBusy}>
                                Load sample
                            </button>
                            <button className="ghost-button" onClick={handleClear} type="button" disabled={isBusy}>
                                Clear
                            </button>
                        </div>
                    </div>

                    <label className="field-label" htmlFor="input-data">Text payload</label>
                    <textarea
                        id="input-data"
                        className="payload-input"
                        value={input}
                        onChange={handleInputChange}
                        spellCheck={false}
                        placeholder="Paste hex, base64, or unknown payload here"
                    />

                    <div className="controls-row">
                        <label className="select-field">
                            <span>Encoding</span>
                            <select value={inputEncoding} onChange={(event) => setInputEncoding(event.target.value)}>
                                <option value="auto">auto</option>
                                <option value="hex">hex</option>
                                <option value="base64">base64</option>
                            </select>
                        </label>

                        <label className="checkbox-field">
                            <input
                                type="checkbox"
                                checked={parseDelimited}
                                onChange={(event) => setParseDelimited(event.target.checked)}
                            />
                            <span>Parse delimited</span>
                        </label>
                    </div>

                    <div className="limits-grid">
                        <label className="number-field">
                            <span>MaxDepth</span>
                            <input type="number" min="1" step="1" value={maxDepth} onChange={handleLimitChange(setMaxDepth)} />
                        </label>

                        <label className="number-field">
                            <span>MaxFields</span>
                            <input type="number" min="1" step="1" value={maxFields} onChange={handleLimitChange(setMaxFields)} />
                        </label>

                        <label className="number-field">
                            <span>MaxBytes</span>
                            <input type="number" min="1" step="1" value={maxBytes} onChange={handleLimitChange(setMaxBytes)} />
                        </label>
                    </div>

                    <div className="dropzone-card">
                        <p className="dropzone-title">Drop file to decode</p>
                        <p className="dropzone-copy">Desktop drag-and-drop uses current options. Files above current MaxBytes fail fast in backend. Files at 5 MiB or larger ask for confirmation first.</p>
                    </div>

                    <div className="limit-note">
                        <p className="limit-note-title">Safety limits</p>
                        <p className="limit-note-copy">Default MaxBytes is 10 MiB. Backend enforces MaxBytes, MaxFields, and MaxDepth even if frontend inputs are edited or stale.</p>
                    </div>

                    <div className="action-row">
                        <button className="primary-button" onClick={handleDecode} type="button" disabled={isBusy || input.trim() === ''}>
                            {isBusy ? 'Working...' : 'Decode text input'}
                        </button>
                        <button className="secondary-button" onClick={handleOpenFile} type="button" disabled={isBusy}>
                            Open and decode file
                        </button>
                    </div>

                    <div className="status-grid">
                        <div className="status-card">
                            <span className="status-label">Selected file</span>
                            <span className="status-value">{selectedFile}</span>
                        </div>
                        <div className="status-card">
                            <span className="status-label">Workspace status</span>
                            <span className="status-value">{statusMessage}</span>
                        </div>
                    </div>
                </article>

                <article className="panel">
                    <div className="panel-header">
                        <h2>Decode Result</h2>
                        <div className="header-actions">
                            <button className="ghost-button" onClick={handleCopyJSON} type="button" disabled={isBusy || !result}>
                                Copy JSON
                            </button>
                            <button className="ghost-button" onClick={() => handleExport('json')} type="button" disabled={isBusy || !result}>
                                Export JSON
                            </button>
                            <button className="ghost-button" onClick={() => handleExport('text')} type="button" disabled={isBusy || !result}>
                                Export text
                            </button>
                        </div>
                    </div>

                    {errorMessage ? <div className="error-banner">{errorMessage}</div> : null}
                    {!errorMessage && result?.error ? <div className="error-banner">{result.error}</div> : null}
                    {limitGuidance.length > 0 ? (
                        <div className="limit-panel">
                            <p className="limit-panel-title">Limit guidance</p>
                            <div className="limit-guidance-list">
                                {limitGuidance.map((message) => (
                                    <span className="limit-guidance-pill" key={message}>{message}</span>
                                ))}
                            </div>
                        </div>
                    ) : null}
                    {!errorMessage && result && resultWarnings.length > 0 ? (
                        <div className="warning-panel">
                            <p className="warning-title">Warnings</p>
                            <div className="warning-list">
                                {resultWarnings.map((warning) => (
                                    <span className="warning-pill" key={warning}>{warning}</span>
                                ))}
                            </div>
                        </div>
                    ) : null}

                    {isBusy ? <div className="state-card">Running decode request...</div> : null}
                    {!isBusy && !errorMessage && !result ? <div className="state-card">No result yet. Decode text input, choose file, or drop file on window.</div> : null}
                    {!isBusy && result ? (
                        <>
                            <div className="result-summary">
                                <span>{result.inputSize} bytes</span>
                                <span>{result.parts.length} parts</span>
                                <span>{resultWarnings.length} warnings</span>
                                <span>{result.error ? 'decoder error present' : 'decoder ok'}</span>
                            </div>

                            <div className="result-workspace">
                                <section className="tree-pane">
                                    <div className="tree-table">
                                        <div className="tree-row tree-row--header">
                                            <span>Field</span>
                                            <span>Wire</span>
                                            <span>Byte range</span>
                                            <span>Summary</span>
                                        </div>

                                        {partRecords.filter((record) => isRecordVisible(record.path, expandedPaths)).map((record) => {
                                            const expanded = expandedPaths[record.path] ?? false;
                                            return (
                                                <div
                                                    key={record.path}
                                                    className={`tree-row${selectedPartPath === record.path ? ' tree-row--selected' : ''}`}
                                                    onClick={() => handleRecordSelect(record.path)}
                                                    onKeyDown={(event) => handleRecordKeyDown(event, record.path)}
                                                    role="button"
                                                    tabIndex={0}
                                                >
                                                    <span className="tree-field-cell" style={{paddingLeft: `${record.level * 18}px`}}>
                                                        {record.hasChildren ? (
                                                            <button
                                                                className="tree-toggle"
                                                                onClick={(event) => {
                                                                    event.stopPropagation();
                                                                    handleRecordToggle(record.path);
                                                                }}
                                                                type="button"
                                                            >
                                                                {expanded ? '−' : '+'}
                                                            </button>
                                                        ) : <span className="tree-spacer" />}
                                                        <span className="tree-field-label">#{record.part.fieldNumber} {record.part.typeName}</span>
                                                    </span>
                                                    <span>{record.part.wireType}</span>
                                                    <span>{formatByteRange(record.globalRange)}</span>
                                                    <span className="tree-summary-cell">{summarizePart(record.part)}</span>
                                                </div>
                                            );
                                        })}
                                    </div>
                                </section>

                                <section className="detail-pane">
                                    <div className="detail-card">
                                        <h3>Field details</h3>
                                        {selectedRecord ? (
                                            <>
                                                <div className="detail-grid">
                                                    <div>
                                                        <span className="detail-label">Path</span>
                                                        <span className="detail-value">{selectedRecord.path}</span>
                                                    </div>
                                                    <div>
                                                        <span className="detail-label">Field</span>
                                                        <span className="detail-value">#{selectedRecord.part.fieldNumber}</span>
                                                    </div>
                                                    <div>
                                                        <span className="detail-label">Type</span>
                                                        <span className="detail-value">{selectedRecord.part.typeName}</span>
                                                    </div>
                                                    <div>
                                                        <span className="detail-label">Byte range</span>
                                                        <span className="detail-value">{formatByteRange(selectedRecord.globalRange)}</span>
                                                    </div>
                                                </div>

                                                <div className="raw-value-card">
                                                    <span className="detail-label">Raw hex</span>
                                                    <code>{selectedRecord.part.rawHex || 'n/a'}</code>
                                                </div>

                                                <div className="candidate-list">
                                                    {selectedRecord.part.value.map((variant: ValueVariant, index: number) => (
                                                        <div className="candidate-card" key={`${selectedRecord.path}-${variant.candidateType}-${index}`}>
                                                            <div className="candidate-header">
                                                                <strong>{variant.candidateType}</strong>
                                                                <span>{variant.confidence || 'candidate'}</span>
                                                            </div>
                                                            <code>{variant.displayValue}</code>
                                                            {variant.description ? <p>{variant.description}</p> : null}
                                                        </div>
                                                    ))}
                                                </div>
                                            </>
                                        ) : (
                                            <div className="state-card">Select field row to inspect candidates, raw hex, and byte range.</div>
                                        )}
                                    </div>

                                    <div className="detail-card">
                                        <h3>Raw hex preview</h3>
                                        <div className="hex-preview">
                                            {rawHexBytes.length > 0 ? Array.from({length: Math.ceil(rawHexBytes.length / 16)}, (_, rowIndex) => {
                                                const start = rowIndex * 16;
                                                const slice = rawHexBytes.slice(start, start + 16);
                                                return (
                                                    <div className="hex-row" key={`hex-row-${start}`}>
                                                        <span className="hex-offset">{formatOffset(start)}</span>
                                                        <div className="hex-byte-strip">
                                                            {slice.map((byteValue, byteIndex) => {
                                                                const absoluteIndex = start + byteIndex;
                                                                const isSelected = selectedRecord
                                                                    ? absoluteIndex >= selectedRecord.globalRange[0] && absoluteIndex < selectedRecord.globalRange[1]
                                                                    : false;
                                                                return (
                                                                    <span className={`hex-byte${isSelected ? ' hex-byte--selected' : ''}${byteValue === '..' ? ' hex-byte--unknown' : ''}`} key={`hex-byte-${absoluteIndex}`}>
                                                                        {byteValue}
                                                                    </span>
                                                                );
                                                            })}
                                                        </div>
                                                    </div>
                                                );
                                            }) : <div className="state-card">No raw hex preview available.</div>}
                                        </div>
                                    </div>

                                    <div className="detail-card">
                                        <h3>Leftover bytes</h3>
                                        {result.leftover !== '' ? <code className="leftover-block">{result.leftover}</code> : <div className="state-card">No leftover bytes.</div>}
                                    </div>
                                </section>
                            </div>
                        </>
                    ) : null}
                </article>
            </section>
        </div>
    );
}

export default App;
