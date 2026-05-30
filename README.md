# Protobuf Decoder Desktop

## Story 1 Scope

This repository now contains the Wails React + TypeScript desktop shell for the schema-less Protobuf decoder rewrite.
Current implementation is intentionally narrow:

- Wails desktop app scaffold is in place.
- Go backend exposes mock `Decode` and `OpenInputFile` bindings.
- React frontend verifies Go method binding and native file dialog wiring.
- Production build pipeline is available through Wails CLI.

## Prerequisites

- Go 1.21+
- Node.js 15+
- Wails CLI v2

## Development Commands

Run environment diagnostics:

```sh
wails doctor
```

Start desktop development mode:

```sh
wails dev
```

Create production build:

```sh
wails build
```

Wails regenerates `frontend/wailsjs/` during `wails dev` and `wails build`, so generated bindings are not kept in git.

## Current Shell Behavior

- `Decode` returns fixed mock result from Go backend using current UI request values.
- `OpenInputFile` opens native file dialog and reports selected path or cancel state.
- Frontend includes sample payload, encoding selector, parse-delimited toggle, and result panel.

## Next Stories

Upcoming stories replace mock decode logic with real Protobuf wire parser, add file decoding, and build full result tree/table UI.
