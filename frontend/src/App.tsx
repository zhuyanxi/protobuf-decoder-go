import {ChangeEvent, useState} from 'react';
import './App.css';
import {Decode, OpenInputFile} from '../wailsjs/go/main/App';

type DecodePart = {
    index: number;
    label: string;
    typeName: string;
    raw: string;
    notes: string[];
};

type DecodeResult = {
    summary: string;
    normalizedInput: string;
    inputEncoding: string;
    parseDelimited: boolean;
    parts: DecodePart[];
};

type OpenFileResult = {
    path: string;
    cancelled: boolean;
};

const sampleInput = '0a03666f6f';

function App() {
    const [input, setInput] = useState(sampleInput);
    const [inputEncoding, setInputEncoding] = useState('auto');
    const [parseDelimited, setParseDelimited] = useState(false);
    const [result, setResult] = useState<DecodeResult | null>(null);
    const [selectedFile, setSelectedFile] = useState('No file selected');
    const [errorMessage, setErrorMessage] = useState('');
    const [isBusy, setIsBusy] = useState(false);

    async function handleDecode() {
        setIsBusy(true);
        setErrorMessage('');

        try {
            const decodeResult = await Decode({
                input,
                inputEncoding,
                parseDelimited,
            });

            setResult(decodeResult as DecodeResult);
        } catch (error) {
            const message = error instanceof Error ? error.message : String(error);
            setResult(null);
            setErrorMessage(message);
        } finally {
            setIsBusy(false);
        }
    }

    async function handleOpenFile() {
        setIsBusy(true);
        setErrorMessage('');

        try {
            const dialogResult = await OpenInputFile() as OpenFileResult;
            if (dialogResult.cancelled) {
                setSelectedFile('File selection cancelled');
                return;
            }

            setSelectedFile(dialogResult.path);
        } catch (error) {
            const message = error instanceof Error ? error.message : String(error);
            setErrorMessage(message);
        } finally {
            setIsBusy(false);
        }
    }

    function handleInputChange(event: ChangeEvent<HTMLTextAreaElement>) {
        setInput(event.target.value);
    }

    function handleReset() {
        setInput(sampleInput);
        setInputEncoding('auto');
        setParseDelimited(false);
        setResult(null);
        setErrorMessage('');
        setSelectedFile('No file selected');
    }

    return (
        <div className="app-shell">
            <section className="hero-card">
                <p className="eyebrow">Story 1 / Wails shell</p>
                <h1>Protobuf Decoder Desktop</h1>
                <p className="intro">
                    Minimal desktop shell for Go bindings, mock decode calls, and native file dialog verification.
                </p>
            </section>

            <section className="workspace-grid">
                <article className="panel">
                    <div className="panel-header">
                        <h2>Decode Request</h2>
                        <button className="ghost-button" onClick={handleReset} type="button">
                            Reset sample
                        </button>
                    </div>

                    <label className="field-label" htmlFor="input-data">Sample payload</label>
                    <textarea
                        id="input-data"
                        className="payload-input"
                        value={input}
                        onChange={handleInputChange}
                        spellCheck={false}
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

                    <div className="action-row">
                        <button className="primary-button" onClick={handleDecode} type="button" disabled={isBusy}>
                            {isBusy ? 'Working...' : 'Run mock decode'}
                        </button>
                        <button className="secondary-button" onClick={handleOpenFile} type="button" disabled={isBusy}>
                            Open file dialog
                        </button>
                    </div>

                    <div className="status-card">
                        <span className="status-label">Selected file</span>
                        <span className="status-value">{selectedFile}</span>
                    </div>
                </article>

                <article className="panel">
                    <div className="panel-header">
                        <h2>Mock Result</h2>
                    </div>

                    {errorMessage ? <div className="error-banner">{errorMessage}</div> : null}

                    <pre className="result-block">
                        {result ? JSON.stringify(result, null, 2) : 'Run mock decode to verify Wails Go binding.'}
                    </pre>
                </article>
            </section>
        </div>
    );
}

export default App;
