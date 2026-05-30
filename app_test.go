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
		Input:          "0a03666f6f",
		InputEncoding:  "hex",
		ParseDelimited: true,
		MaxDepth:       7,
		MaxFields:      99,
		MaxBytes:       2048,
	})
	if err != nil {
		t.Fatalf("decode returned unexpected error: %v", err)
	}

	if result.InputSize != 5 {
		t.Fatalf("expected inputSize %d, got %d", 5, result.InputSize)
	}

	if len(result.Parts) != 1 {
		t.Fatalf("expected one decoded part, got %d", len(result.Parts))
	}

	if result.Parts[0].FieldNumber != 1 {
		t.Fatalf("expected fieldNumber 1, got %d", result.Parts[0].FieldNumber)
	}

	if result.Parts[0].TypeName != "LENDELIM" {
		t.Fatalf("expected LENDELIM typeName, got %q", result.Parts[0].TypeName)
	}

	if len(result.Parts[0].Value) != 1 {
		t.Fatalf("expected one raw value variant, got %#v", result.Parts[0].Value)
	}

	if result.Parts[0].Value[0].DisplayValue != "666f6f" {
		t.Fatalf("expected payload raw hex %q, got %q", "666f6f", result.Parts[0].Value[0].DisplayValue)
	}

	if result.Error != "" {
		t.Fatalf("expected empty decode error, got %q", result.Error)
	}

	if len(result.Warnings) != 2 {
		t.Fatalf("expected two request warnings, got %#v", result.Warnings)
	}

	if result.Parts[0].RawHex != "0a03666f6f" {
		t.Fatalf("expected normalized rawHex %q, got %q", "0a03666f6f", result.Parts[0].RawHex)
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