# CI Platform Notes

## PR CI scope

- `ci.yml` runs `go test ./...` on Ubuntu.
- `ci.yml` runs frontend smoke tests with Vitest plus `npm run build` on Ubuntu.
- `ci.yml` runs native `wails build -clean` smoke jobs on Ubuntu, macOS, and Windows runners.

## Linux runner strategy

- CI target is `ubuntu-latest`.
- Workflow installs `build-essential`, `pkg-config`, and `libgtk-3-dev` before Wails build.
- Workflow prefers `libwebkit2gtk-4.1-dev` with `libsoup-3.0-dev` when runner image provides it.
- Workflow falls back to `libwebkit2gtk-4.0-dev` when 4.1 packages are unavailable.
- When 4.1 package is present, workflow builds with `-tags webkit2_41`.

## Windows runner strategy

- CI builds unsigned Windows artifact on native runner.
- Runtime dependency is Microsoft WebView2. Wails marks it as required for app startup, but GitHub Actions build job does not bundle runtime.
- Release/install guidance for end users: require Evergreen WebView2 Runtime on target machines and link Microsoft download page in release notes.
- Optional NSIS installer generation can be layered later if project decides to ship installer instead of raw executable.

## macOS runner strategy

- CI builds unsigned `.app` artifact on native macOS runner.
- Signing and notarization are intentionally outside current workflow scope.
- Follow-up release steps: sign app with Developer ID Application certificate, submit with `xcrun notarytool`, then staple notarization ticket.

## Artifact expectations

- Release workflow uploads contents of `build/bin/` for Linux, macOS, and Windows.
- Current workflows validate build/package generation, not GUI launch automation.
- Smoke coverage focuses on dependency installation, frontend build, Go tests, and native Wails packaging stages so CI logs show exact failing stage.