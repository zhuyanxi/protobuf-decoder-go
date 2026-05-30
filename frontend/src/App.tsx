import {ChangeEvent, DragEvent, useState} from 'react';
import './App.css';
import {Decode, DecodeFile, OpenInputFile} from '../wailsjs/go/main/App';
import {main} from '../wailsjs/go/models';

type DecodeRequest = main.DecodeRequest;
type DecodeOptions = main.DecodeOptions;
type DecodeResult = main.DecodeResult;
type OpenFileResult = main.OpenFileResult;
type DroppedFile = File & {path?: string};

const sampleInput = '0a03666f6f';
const defaultSelectedFile = 'No file selected';
const idleStatus = 'Paste payload, tune limits, or drop file to start decoding.';

const defaultDecodeRequest: DecodeRequest = {
    input: sampleInput,
    inputEncoding: 'auto',
    parseDelimited: false,
    maxDepth: 4,
    maxFields: 256,
    maxBytes: 1024 * 1024,
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
            setResult(decodeResult);
            setSelectedFile(path);
            setStatusMessage(`${sourceLabel} decoded (${decodeResult.inputSize} bytes).`);
        } catch (error) {
            const message = error instanceof Error ? error.message : String(error);
            setResult(null);
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
            setResult(decodeResult);
            setStatusMessage(`Decoded text input (${decodeResult.inputSize} bytes).`);
        } catch (error) {
            const message = error instanceof Error ? error.message : String(error);
            setResult(null);
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

            await decodeFileAtPath(dialogResult.path, 'File');
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
        setErrorMessage('');
        setSelectedFile(defaultSelectedFile);
        setStatusMessage('Request cleared.');
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
                <p className="eyebrow">Story 11 / decode workspace</p>
                <h1>Protobuf Decoder Desktop</h1>
                <p className="intro">
                    Paste hex or base64, choose file, or drop file on window. Tune decode limits before each request, then inspect raw JSON contract from Wails backend.
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
                        <p className="dropzone-copy">Desktop drag-and-drop uses current options. If file path is unavailable, fall back to native picker.</p>
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
                    </div>

                    {errorMessage ? <div className="error-banner">{errorMessage}</div> : null}

                    {isBusy ? <div className="state-card">Running decode request...</div> : null}
                    {!isBusy && !errorMessage && !result ? <div className="state-card">No result yet. Decode text input, choose file, or drop file on window.</div> : null}
                    {!isBusy && result ? (
                        <>
                            <div className="result-summary">
                                <span>{result.inputSize} bytes</span>
                                <span>{result.parts.length} parts</span>
                                <span>{result.warnings?.length ?? 0} warnings</span>
                                <span>{result.error ? 'decoder error present' : 'decoder ok'}</span>
                            </div>
                            <pre className="result-block">{JSON.stringify(result, null, 2)}</pre>
                        </>
                    ) : null}
                </article>
            </section>
        </div>
    );
}

export default App;
