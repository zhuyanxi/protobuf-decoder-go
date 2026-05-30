# Release Notes Template

## Summary

- Release version:
- Release date:
- Scope:

## Highlights

-
-
-

## Features

-
-

## Fixes

-
-

## Known limits

- Schema-less decode returns heuristics, not schema truth.
- Nested protobuf guess may reject payload and fall back to bytes/string candidates.
- Unsigned macOS builds may require manual Gatekeeper open flow.
- Windows targets require Microsoft WebView2 runtime.
- Linux targets require GTK/WebKitGTK runtime compatibility.

## Artifacts

- macOS:
- Windows:
- Linux:

## Install notes

- Windows: mention WebView2 runtime requirement and download link.
- Linux: mention target distro and WebKitGTK package family used for build.
- macOS: mention signing/notarization status.

## Verification

- `go test ./...`
- `npm --prefix frontend test`
- `npm --prefix frontend run build`
- `wails build`

## Upgrade notes

-
-