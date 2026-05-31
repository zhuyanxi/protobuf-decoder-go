import {act} from 'react';
import {createRoot, Root} from 'react-dom/client';
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import {CopyResultJSON, Decode, DecodeFile, ExportResult, OpenInputFile} from '../wailsjs/go/main/App';
import {main} from '../wailsjs/go/models';
import App from './App';

vi.mock('../wailsjs/go/main/App', () => ({
    CopyResultJSON: vi.fn(),
    Decode: vi.fn(),
    DecodeFile: vi.fn(),
    ExportResult: vi.fn(),
    OpenInputFile: vi.fn(),
}));

const sampleDecodeResult = {
    parts: [
        {
            byteRange: [0, 4],
            index: 1,
            fieldNumber: 1,
            wireType: 2,
            typeName: 'LENDELIM',
            rawHex: '0a020801',
            value: [
                {
                    candidateType: 'nested.protobuf',
                    displayValue: '1 nested field',
                    description: 'Payload fully parsed as nested protobuf',
                    confidence: 'high',
                },
                {
                    candidateType: 'bytes.hex',
                    displayValue: '0801',
                    description: 'Raw payload bytes',
                    confidence: 'high',
                },
            ],
            children: [
                {
                    byteRange: [0, 2],
                    index: 1,
                    fieldNumber: 1,
                    wireType: 0,
                    typeName: 'VARINT',
                    rawHex: '0801',
                    value: [
                        {
                            candidateType: 'uint64',
                            displayValue: '1',
                            description: 'Unsigned varint',
                            confidence: 'high',
                        },
                    ],
                    children: [],
                },
            ],
        },
    ],
    leftover: '',
    warnings: ['Input encoding: hex', 'Nested protobuf candidate accepted for field 1.'],
    inputSize: 4,
} as unknown as main.DecodeResult;

function buttonByText(container: HTMLElement, text: string): HTMLButtonElement {
    const button = Array.from(container.querySelectorAll('button')).find((item) => item.textContent === text);
    if (!(button instanceof HTMLButtonElement)) {
        throw new Error(`button not found: ${text}`);
    }

    return button;
}

async function flushPromises() {
    await Promise.resolve();
    await Promise.resolve();
}

async function clickButton(container: HTMLElement, text: string) {
    const button = buttonByText(container, text);
    await act(async () => {
        button.dispatchEvent(new MouseEvent('click', {bubbles: true}));
        await flushPromises();
    });
}

async function renderApp(root: Root) {
    await act(async () => {
        root.render(<App />);
    });
}

describe('App', () => {
    let container: HTMLDivElement;
    let root: Root;

    beforeEach(() => {
        vi.clearAllMocks();
        container = document.createElement('div');
        document.body.appendChild(container);
        root = createRoot(container);
    });

    afterEach(async () => {
        await act(async () => {
            root.unmount();
        });
        container.remove();
    });

    it('renders decode workspace controls', async () => {
        await renderApp(root);

        expect(container.textContent).toContain('Decode Request');
        expect(container.textContent).toContain('Decode Result');
        expect(container.textContent).toContain('Decode text input');
        expect(container.textContent).toContain('Open and decode file');
    });

    it('renders decode result, warnings, details, and expanded nested fields', async () => {
        vi.mocked(Decode).mockResolvedValue(sampleDecodeResult);
        await renderApp(root);

        await clickButton(container, 'Decode text input');

        expect(Decode).toHaveBeenCalledWith({
            input: '0a03666f6f',
            inputEncoding: 'auto',
            parseDelimited: false,
            maxDepth: 4,
            maxFields: 256,
            maxBytes: 10485760,
        });
        expect(container.textContent).toContain('Decoded text input (4 bytes).');
        expect(container.textContent).toContain('4 bytes');
        expect(container.textContent).toContain('1 parts');
        expect(container.textContent).toContain('Input encoding: hex');
        expect(container.textContent).toContain('#1 LENDELIM');
        expect(container.textContent).toContain('nested.protobuf');
        expect(container.textContent).toContain('0a020801');
        expect(container.textContent).not.toContain('#1 VARINT');

        await clickButton(container, '+');

        expect(container.textContent).toContain('#1 VARINT');
        expect(container.textContent).toContain('uint64: 1');
    });

    it('keeps export controls disabled before result and calls copy/export after decode', async () => {
        vi.mocked(Decode).mockResolvedValue(sampleDecodeResult);
        vi.mocked(CopyResultJSON).mockResolvedValue(undefined);
        vi.mocked(ExportResult).mockResolvedValue({path: '/tmp/result.json', cancelled: false, format: 'json'});
        await renderApp(root);

        expect(buttonByText(container, 'Copy JSON').disabled).toBe(true);
        expect(buttonByText(container, 'Export JSON').disabled).toBe(true);

        await clickButton(container, 'Decode text input');

        expect(buttonByText(container, 'Copy JSON').disabled).toBe(false);
        expect(buttonByText(container, 'Export JSON').disabled).toBe(false);

        await clickButton(container, 'Copy JSON');
        expect(CopyResultJSON).toHaveBeenCalledWith(sampleDecodeResult);
        expect(container.textContent).toContain('Copied pretty JSON to clipboard.');

        await clickButton(container, 'Export JSON');
        expect(ExportResult).toHaveBeenCalledWith(sampleDecodeResult, 'json');
        expect(container.textContent).toContain('Saved JSON export to /tmp/result.json.');
    });

    it('clears prior result and shows decode error feedback', async () => {
        vi.mocked(Decode).mockResolvedValueOnce(sampleDecodeResult).mockRejectedValueOnce(new Error('hex input error at position 2: invalid hex character'));
        await renderApp(root);

        await clickButton(container, 'Decode text input');
        expect(container.textContent).toContain('#1 LENDELIM');

        await clickButton(container, 'Decode text input');

        expect(container.textContent).toContain('hex input error at position 2: invalid hex character');
        expect(container.textContent).toContain('Text decode failed.');
        expect(container.textContent).not.toContain('#1 LENDELIM');
    });

    it('asks for large file confirmation before DecodeFile', async () => {
        const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false);
        vi.mocked(OpenInputFile).mockResolvedValue({path: '/tmp/large.bin', size: 6 * 1024 * 1024, cancelled: false});
        await renderApp(root);

        await clickButton(container, 'Open and decode file');

        expect(confirmSpy).toHaveBeenCalledWith(expect.stringContaining('Large inputs can take longer to decode'));
        expect(DecodeFile).not.toHaveBeenCalled();
        expect(container.textContent).toContain('Large file decode cancelled. Adjust MaxBytes, then retry if needed.');
    });
});