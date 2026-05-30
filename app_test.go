package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeRequestOptionsDefaults(t *testing.T) {
	request := DecodeRequest{}
	options := request.Options()

	if options.MaxDepth != defaultMaxDepth {
		t.Fatalf("expected default maxDepth %d, got %d", defaultMaxDepth, options.MaxDepth)
	}

	if options.MaxFields != defaultMaxFields {
		t.Fatalf("expected default maxFields %d, got %d", defaultMaxFields, options.MaxFields)
	}

	if options.MaxBytes != defaultMaxBytes {
		t.Fatalf("expected default maxBytes %d, got %d", defaultMaxBytes, options.MaxBytes)
	}
}

func TestDecodeResultJSONContract(t *testing.T) {
	result := DecodeResult{
		Parts: []Part{{
			ByteRange:   [2]int{0, 5},
			Index:       1,
			FieldNumber: 3,
			WireType:    2,
			TypeName:    "LENDELIM",
			RawHex:      "0a03666f6f",
			Value: []ValueVariant{{
				CandidateType: "int64",
				DisplayValue:  "123",
				Description:   "candidate",
				Confidence:    "medium",
			}},
		}},
		Leftover:  "ff",
		Warnings:  []string{"candidate only"},
		InputSize: 10,
	}

	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal decode result: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal decode result: %v", err)
	}

	for _, key := range []string{"parts", "leftover", "warnings", "inputSize"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("expected top-level key %q in JSON payload %s", key, string(payload))
		}
	}

	parts, ok := decoded["parts"].([]any)
	if !ok || len(parts) != 1 {
		t.Fatalf("expected one part in JSON payload %s", string(payload))
	}

	part, ok := parts[0].(map[string]any)
	if !ok {
		t.Fatalf("expected part object in JSON payload %s", string(payload))
	}

	for _, key := range []string{"byteRange", "index", "fieldNumber", "wireType", "typeName", "rawHex", "value"} {
		if _, ok := part[key]; !ok {
			t.Fatalf("expected part key %q in JSON payload %s", key, string(payload))
		}
	}

	values, ok := part["value"].([]any)
	if !ok || len(values) != 1 {
		t.Fatalf("expected one value variant in JSON payload %s", string(payload))
	}

	variant, ok := values[0].(map[string]any)
	if !ok {
		t.Fatalf("expected value variant object in JSON payload %s", string(payload))
	}

	for _, key := range []string{"candidateType", "displayValue", "description", "confidence"} {
		if _, ok := variant[key]; !ok {
			t.Fatalf("expected variant key %q in JSON payload %s", key, string(payload))
		}
	}

	if value, ok := variant["displayValue"].(string); !ok || value != "123" {
		t.Fatalf("expected displayValue to stay string, got %#v", variant["displayValue"])
	}
}

func TestDecodeReturnsStructuredContract(t *testing.T) {
	app := NewApp()
	result, err := app.Decode(DecodeRequest{
		Input:          "050a03666f6f",
		InputEncoding:  "hex",
		ParseDelimited: true,
		MaxDepth:       7,
		MaxFields:      99,
		MaxBytes:       2048,
	})
	if err != nil {
		t.Fatalf("decode returned unexpected error: %v", err)
	}

	if result.InputSize != 6 {
		t.Fatalf("expected inputSize %d, got %d", 6, result.InputSize)
	}

	if len(result.Parts) != 2 {
		t.Fatalf("expected delimiter and one decoded part, got %d", len(result.Parts))
	}

	if result.Parts[0].TypeName != "MessageDelimiter" {
		t.Fatalf("expected MessageDelimiter part, got %q", result.Parts[0].TypeName)
	}

	if result.Parts[1].FieldNumber != 1 {
		t.Fatalf("expected fieldNumber 1, got %d", result.Parts[1].FieldNumber)
	}

	if result.Parts[1].TypeName != "LENDELIM" {
		t.Fatalf("expected LENDELIM typeName, got %q", result.Parts[1].TypeName)
	}

	if len(result.Parts[1].Value) != 2 {
		t.Fatalf("expected string and bytes variants, got %#v", result.Parts[1].Value)
	}

	if result.Parts[1].Value[0].CandidateType != "string.utf8" || result.Parts[1].Value[0].DisplayValue != "foo" {
		t.Fatalf("expected UTF-8 string candidate foo, got %#v", result.Parts[1].Value[0])
	}

	if result.Parts[1].Value[1].CandidateType != "bytes.hex" || result.Parts[1].Value[1].DisplayValue != "666f6f" {
		t.Fatalf("expected payload raw hex %q, got %#v", "666f6f", result.Parts[1].Value[1])
	}

	if result.Error != "" {
		t.Fatalf("expected empty decode error, got %q", result.Error)
	}

	if len(result.Warnings) != 2 {
		t.Fatalf("expected two request warnings, got %#v", result.Warnings)
	}

	if result.Parts[1].RawHex != "0a03666f6f" {
		t.Fatalf("expected normalized rawHex %q, got %q", "0a03666f6f", result.Parts[1].RawHex)
	}
}

func TestDecodeAutoDetectsHexInput(t *testing.T) {
	app := NewApp()
	result, err := app.Decode(DecodeRequest{Input: "08 01", InputEncoding: "auto"})
	if err != nil {
		t.Fatalf("decode auto returned unexpected error: %v", err)
	}

	if len(result.Parts) != 1 {
		t.Fatalf("expected one parsed part, got %d", len(result.Parts))
	}

	if result.Parts[0].RawHex != "0801" {
		t.Fatalf("expected normalized rawHex %q, got %q", "0801", result.Parts[0].RawHex)
	}

	if len(result.Warnings) < 4 || !strings.Contains(strings.Join(result.Warnings, " | "), "Auto-detected input encoding: hex") {
		t.Fatalf("expected auto-detect warning, got %#v", result.Warnings)
	}
}

func TestDecodeFileReadsBinaryPayload(t *testing.T) {
	app := NewApp()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "payload.bin")
	if err := os.WriteFile(filePath, []byte{0x0a, 0x03, 0x66, 0x6f, 0x6f}, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	result, err := app.DecodeFile(filePath, DecodeOptions{MaxBytes: 16})
	if err != nil {
		t.Fatalf("decode file returned unexpected error: %v", err)
	}

	if result.InputSize != 5 {
		t.Fatalf("expected file inputSize %d, got %d", 5, result.InputSize)
	}

	if result.Parts[0].RawHex != "0a03666f6f" {
		t.Fatalf("expected normalized file rawHex %q, got %q", "0a03666f6f", result.Parts[0].RawHex)
	}
}

func TestDecodeFileRejectsMissingFile(t *testing.T) {
	app := NewApp()
	_, err := app.DecodeFile("/tmp/does-not-exist.bin", DecodeOptions{MaxBytes: 16})
	if err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestDecodeFileRejectsOversizedFile(t *testing.T) {
	app := NewApp()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "payload.bin")
	if err := os.WriteFile(filePath, []byte{0x08, 0x01, 0x10, 0x02}, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err := app.DecodeFile(filePath, DecodeOptions{MaxBytes: 3})
	if err == nil {
		t.Fatal("expected oversized file error")
	}

	if !strings.Contains(err.Error(), "exceeds maxBytes 3") {
		t.Fatalf("expected maxBytes guidance in file error, got %v", err)
	}
}

func TestBuildExportPayloadJSONPrettyPrints(t *testing.T) {
	payload, err := buildExportPayload(DecodeResult{
		Parts: []Part{{
			FieldNumber: 1,
			TypeName:    "VARINT",
			RawHex:      "0801",
			Value: []ValueVariant{{
				CandidateType: "int64",
				DisplayValue:  "9223372036854775807",
			}},
		}},
		Warnings: []string{"candidate only"},
	}, "json")
	if err != nil {
		t.Fatalf("build json export payload: %v", err)
	}

	if !strings.Contains(payload, "\n  \"parts\": [") {
		t.Fatalf("expected pretty JSON payload, got %q", payload)
	}

	if !strings.Contains(payload, "9223372036854775807") {
		t.Fatalf("expected large integer to remain string in payload, got %q", payload)
	}
}

func TestBuildExportPayloadTextIncludesNestedFields(t *testing.T) {
	payload, err := buildExportPayload(DecodeResult{
		Parts: []Part{{
			ByteRange:   [2]int{0, 5},
			FieldNumber: 1,
			WireType:    2,
			TypeName:    "LENDELIM",
			RawHex:      "0a03666f6f",
			Value: []ValueVariant{{
				CandidateType: "string.utf8",
				DisplayValue:  "foo",
				Confidence:    "high",
			}},
			Children: []Part{{
				ByteRange:   [2]int{0, 2},
				FieldNumber: 2,
				WireType:    0,
				TypeName:    "VARINT",
				RawHex:      "1001",
				Value: []ValueVariant{{
					CandidateType: "uint64",
					DisplayValue:  "1",
				}},
			}},
		}},
		Leftover:  "ff",
		Error:     "truncated payload",
		Warnings:  []string{"candidate only"},
		InputSize: 6,
	}, "text")
	if err != nil {
		t.Fatalf("build text export payload: %v", err)
	}

	for _, fragment := range []string{
		"Input size: 6 bytes",
		"Warnings:",
		"Error: truncated payload",
		"Leftover: ff",
		"- #1 LENDELIM wire=2 range=[0, 5) raw=0a03666f6f",
		"* string.utf8: foo [high]",
		"  - #2 VARINT wire=0 range=[0, 2) raw=1001",
	} {
		if !strings.Contains(payload, fragment) {
			t.Fatalf("expected fragment %q in text payload %q", fragment, payload)
		}
	}
}

func TestExportResultRejectsUnsupportedFormat(t *testing.T) {
	app := NewApp()
	_, err := app.ExportResult(DecodeResult{}, "xml")
	if err == nil {
		t.Fatal("expected unsupported export format error")
	}

	if !strings.Contains(err.Error(), "application context is not ready") {
		t.Fatalf("expected context error before dialog when app not started, got %v", err)
	}
}

func TestNormalizeExportFormatRejectsUnknownValue(t *testing.T) {
	_, err := normalizeExportFormat("xml")
	if err == nil {
		t.Fatal("expected unsupported export format error")
	}
}