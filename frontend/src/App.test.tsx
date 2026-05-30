import {act} from 'react';
import {createRoot, Root} from 'react-dom/client';
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import App from './App';

vi.mock('../wailsjs/go/main/App', () => ({
    CopyResultJSON: vi.fn(),
    Decode: vi.fn(),
    DecodeFile: vi.fn(),
    ExportResult: vi.fn(),
    OpenInputFile: vi.fn(),
}));

describe('App', () => {
    let container: HTMLDivElement;
    let root: Root;

    beforeEach(() => {
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
        await act(async () => {
            root.render(<App />);
        });

        expect(container.textContent).toContain('Decode Request');
        expect(container.textContent).toContain('Decode Result');
        expect(container.textContent).toContain('Decode text input');
        expect(container.textContent).toContain('Open and decode file');
    });
});