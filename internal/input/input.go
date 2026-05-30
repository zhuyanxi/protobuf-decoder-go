package input

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"unicode"
)

type NormalizedText struct {
	Bytes    []byte
	Encoding string
	Warnings []string
}

type Error struct {
	Encoding string
	Position int
	Reason   string
}

func (e *Error) Error() string {
	if e.Position >= 0 {
		return fmt.Sprintf("%s input error at position %d: %s", e.Encoding, e.Position, e.Reason)
	}

	return fmt.Sprintf("%s input error: %s", e.Encoding, e.Reason)
}

func NormalizeText(raw string, encoding string, maxBytes int) (NormalizedText, error) {
	switch encoding {
	case "hex":
		return decodeHex(raw, maxBytes)
	case "base64":
		return decodeBase64(raw, maxBytes)
	case "auto":
		return decodeAuto(raw, maxBytes)
	default:
		return NormalizedText{}, &Error{Encoding: encoding, Position: -1, Reason: "unsupported encoding"}
	}
}

func LoadFile(path string, maxBytes int) ([]byte, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	info, err := os.Stat(trimmedPath)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", trimmedPath, err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("read file %q: path is a directory", trimmedPath)
	}

	if maxBytes > 0 && info.Size() > int64(maxBytes) {
		return nil, fmt.Errorf("read file %q: file size %d exceeds maxBytes %d", trimmedPath, info.Size(), maxBytes)
	}

	data, err := os.ReadFile(trimmedPath)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", trimmedPath, err)
	}

	if err := enforceMaxBytes(len(data), maxBytes, "file"); err != nil {
		return nil, err
	}

	return data, nil
}

func decodeAuto(raw string, maxBytes int) (NormalizedText, error) {
	hexResult, hexErr := decodeHex(raw, maxBytes)
	base64Result, base64Err := decodeBase64(raw, maxBytes)

	switch {
	case hexErr == nil && base64Err != nil:
		hexResult.Warnings = append(hexResult.Warnings, "Auto-detected input encoding: hex")
		return hexResult, nil
	case hexErr != nil && base64Err == nil:
		base64Result.Warnings = append(base64Result.Warnings, "Auto-detected input encoding: base64")
		return base64Result, nil
	case hexErr == nil && base64Err == nil:
		if preferHex(raw) {
			hexResult.Warnings = append(hexResult.Warnings, "Auto-detected input encoding: hex", "Input also matched base64. Hex chosen by heuristic.")
			return hexResult, nil
		}

		base64Result.Warnings = append(base64Result.Warnings, "Auto-detected input encoding: base64", "Input also matched hex. Base64 chosen by heuristic.")
		return base64Result, nil
	default:
		return NormalizedText{}, fmt.Errorf("auto input detection failed: %v; %v", hexErr, base64Err)
	}
}

func decodeHex(raw string, maxBytes int) (NormalizedText, error) {
	cleaned, err := cleanHex(raw)
	if err != nil {
		return NormalizedText{}, err
	}

	decoded, decodeErr := hex.DecodeString(cleaned)
	if decodeErr != nil {
		return NormalizedText{}, &Error{Encoding: "hex", Position: -1, Reason: decodeErr.Error()}
	}

	if err := enforceMaxBytes(len(decoded), maxBytes, "hex"); err != nil {
		return NormalizedText{}, err
	}

	return NormalizedText{Bytes: decoded, Encoding: "hex"}, nil
}

func decodeBase64(raw string, maxBytes int) (NormalizedText, error) {
	cleaned := stripWhitespace(raw)
	if cleaned == "" {
		return NormalizedText{}, &Error{Encoding: "base64", Position: -1, Reason: "input is empty"}
	}

	encodings := []struct {
		name string
		enc  *base64.Encoding
	}{
		{name: "standard", enc: base64.StdEncoding},
		{name: "raw-standard", enc: base64.RawStdEncoding},
		{name: "url", enc: base64.URLEncoding},
		{name: "raw-url", enc: base64.RawURLEncoding},
	}

	var lastErr error
	for _, candidate := range encodings {
		decoded, err := candidate.enc.DecodeString(cleaned)
		if err != nil {
			lastErr = err
			continue
		}

		if err := enforceMaxBytes(len(decoded), maxBytes, "base64"); err != nil {
			return NormalizedText{}, err
		}

		result := NormalizedText{Bytes: decoded, Encoding: "base64"}
		if candidate.name != "standard" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Accepted %s base64 input.", candidate.name))
		}

		return result, nil
	}

	return NormalizedText{}, &Error{Encoding: "base64", Position: -1, Reason: lastErr.Error()}
}

func cleanHex(raw string) (string, error) {
	var builder strings.Builder
	lastHexPosition := -1

	for index, value := range raw {
		switch {
		case unicode.IsSpace(value):
			continue
		case value == ',' || value == ':' || value == '-' || value == '_':
			continue
		case value == '0' && index+1 < len(raw) && (raw[index+1] == 'x' || raw[index+1] == 'X'):
			continue
		case value == 'x' || value == 'X':
			if index > 0 && raw[index-1] == '0' {
				continue
			}
			return "", &Error{Encoding: "hex", Position: index, Reason: fmt.Sprintf("invalid hex character %q", value)}
		case isHexDigit(value):
			builder.WriteRune(value)
			lastHexPosition = index
		default:
			return "", &Error{Encoding: "hex", Position: index, Reason: fmt.Sprintf("invalid hex character %q", value)}
		}
	}

	cleaned := builder.String()
	if cleaned == "" {
		return "", &Error{Encoding: "hex", Position: -1, Reason: "input is empty"}
	}

	if len(cleaned)%2 != 0 {
		return "", &Error{Encoding: "hex", Position: lastHexPosition, Reason: "hex input has odd length after normalization"}
	}

	return cleaned, nil
}

func stripWhitespace(raw string) string {
	var builder strings.Builder
	for _, value := range raw {
		if unicode.IsSpace(value) {
			continue
		}
		builder.WriteRune(value)
	}
	return builder.String()
}

func enforceMaxBytes(length int, maxBytes int, encoding string) error {
	if maxBytes > 0 && length > maxBytes {
		return &Error{Encoding: encoding, Position: -1, Reason: fmt.Sprintf("decoded size %d exceeds maxBytes %d", length, maxBytes)}
	}

	return nil
}

func preferHex(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if strings.Contains(strings.ToLower(trimmed), "0x") {
		return true
	}

	if strings.ContainsAny(trimmed, ",:-_") {
		return true
	}

	compact := stripWhitespace(trimmed)
	if compact != "" && isHexOnly(compact) && len(compact)%2 == 0 {
		return true
	}

	if strings.ContainsAny(trimmed, "+/=") {
		return false
	}

	return false
}

func isHexOnly(value string) bool {
	for _, item := range value {
		if !isHexDigit(item) {
			return false
		}
	}
	return true
}

func isHexDigit(value rune) bool {
	return ('0' <= value && value <= '9') || ('a' <= value && value <= 'f') || ('A' <= value && value <= 'F')
}