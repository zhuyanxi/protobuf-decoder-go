# Troubleshooting

## Decode says input is empty

Cause:

- Empty text payload.
- File picker cancelled.

Fix:

- Paste payload.
- Load sample payload.
- Choose file again.

## Hex input error

Typical causes:

- Invalid hex character.
- Odd number of hex digits after cleanup.

Supported cleanup rules:

- Whitespace ignored.
- `,`, `:`, `-`, `_` ignored.
- `0x` prefixes supported.

Fix:

- Remove non-hex characters.
- Make sure cleaned byte stream has even number of hex digits.

## Base64 input error

Typical causes:

- Corrupted padding.
- Mixed hex/base64 payload pasted under wrong encoding.

Fix:

- Switch encoding to `auto` and retry.
- If `auto` reports ambiguity, choose explicit encoding manually.

## `leftover` bytes shown

Meaning:

- Decoder stopped before consuming full payload.
- Usually caused by unsupported wire type, truncated field, size limit, or invalid nested candidate.

Read it with:

- `error`
- warnings
- byte range of last successful field

## Truncated input or unexpected EOF

Meaning:

- Payload ended before varint, fixed-width field, or length-delimited bytes finished.

Fix:

- Recopy payload from source.
- Verify log/trace did not trim bytes.
- For gRPC payloads, verify 5-byte header length matches body.

## Nested protobuf candidate rejected

Meaning:

- Length-delimited bytes looked like nested protobuf at first.
- Full nested parse still failed or left leftover bytes.

Fix:

- Treat bytes/string candidates as fallback truth.
- Inspect `leftover` and warnings before assuming nested structure.

## `MaxBytes` / `MaxFields` / `MaxDepth` limit hit

Meaning:

- Guardrails stopped decode to keep app responsive.
- `MaxFields` is global across top-level, nested, and delimited messages.

Fix:

- Raise only needed limit.
- Prefer small increments on trusted data.
- Do not raise limits blindly for untrusted payloads.

## Large file confirmation appears

Meaning:

- File is at least 5 MiB, or larger than current `MaxBytes`.

Fix:

- Keep current limit and cancel if payload is untrusted.
- Raise `MaxBytes` before retry if decode is expected to fail due to size.

## Windows app does not open

Most common cause:

- WebView2 runtime missing.

Fix:

- Install Microsoft Evergreen WebView2 Runtime.
- Re-run app.
- Use `wails doctor` on build machine to confirm dependency.

## Linux build fails on GTK/WebKitGTK

Most common cause:

- Missing `libgtk-3-dev` or WebKitGTK development package.

Fix:

- Install packages listed in [platform-install.md](platform-install.md).
- If distro provides WebKitGTK 4.1, build with `-tags webkit2_41`.

## macOS app blocked by Gatekeeper

Meaning:

- Current builds are unsigned or not notarized.

Fix:

- Right-click app and choose `Open`.
- For distribution beyond local testing, sign and notarize future releases.