# Platform Install Guide

## Common requirements

- Go 1.21+
- Node.js 20+
- Wails CLI v2

Install Wails CLI:

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Verify toolchain:

```sh
go version
npm --version
wails doctor
```

## macOS

Supported notes:

- Apple Silicon and Intel builds supported by Wails.
- macOS 15+ should use Go 1.23.3+.
- Current CI builds unsigned `.app` bundle.

Local build:

```sh
npm --prefix frontend install
wails build
```

Open built app:

- Artifact lands in `build/bin/protobuf-decoder-go.app`.
- If Gatekeeper blocks unsigned build, right-click app, choose `Open`, then confirm.

Signing status:

- Current repository does not yet automate signing or notarization.
- Future release flow should sign with Developer ID Application certificate, notarize with `xcrun notarytool`, then staple ticket.

## Windows

Runtime requirement:

- Microsoft Edge WebView2 Runtime must exist on target machine.
- `wails doctor` reports whether local environment already has WebView2.

Build notes:

- CI builds Windows artifact on `windows-latest` runner.
- Current workflow uploads raw build artifact, not NSIS installer.

User install guidance:

- Install Evergreen WebView2 Runtime from Microsoft if app fails to launch due to missing runtime.
- If organization manages locked-down workstations, ask administrator to deploy WebView2 runtime before app rollout.

Optional later packaging:

- NSIS installer path is possible in future release step with `wails build -nsis`.

## Linux

Runtime/build dependency notes:

- GTK 3 development headers required.
- WebKitGTK development headers required.
- On Ubuntu-family systems, CI installs:
  - `build-essential`
  - `pkg-config`
  - `libgtk-3-dev`
  - `libwebkit2gtk-4.1-dev` and `libsoup-3.0-dev` when available
  - fallback `libwebkit2gtk-4.0-dev` when 4.1 packages are unavailable

Ubuntu example:

```sh
sudo apt-get update
sudo apt-get install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.0-dev
```

If distro provides WebKitGTK 4.1 packages:

```sh
sudo apt-get update
sudo apt-get install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev libsoup-3.0-dev
wails build -tags webkit2_41
```

Support note:

- Distribution package names vary. Check `wails doctor` and local package manager if exact names differ.

## Artifacts

- Current GitHub Actions release workflow uploads contents of `build/bin/` for each native runner.
- GUI launch smoke automation is not yet part of CI; workflow validates packaging stages and build logs only.