package main
package main

import (
	"encoding/json"
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

	if result.InputSize != len("0a03666f6f") {
		t.Fatalf("expected inputSize %d, got %d", len("0a03666f6f"), result.InputSize)
	}

	if len(result.Parts) != 1 {
		t.Fatalf("expected one mock part, got %d", len(result.Parts))
	}

	if result.Parts[0].FieldNumber != 1 {
		t.Fatalf("expected fieldNumber 1, got %d", result.Parts[0].FieldNumber)
	}

	if len(result.Parts[0].Value) < 2 {
		t.Fatalf("expected mock value variants, got %#v", result.Parts[0].Value)
	}

	if result.Parts[0].Value[1].DisplayValue != "10" {
		t.Fatalf("expected 64-bit candidate to be serialized as string %q, got %q", "10", result.Parts[0].Value[1].DisplayValue)
	}

	if len(result.Warnings) != 3 {
		t.Fatalf("expected warnings to describe mock contract, got %#v", result.Warnings)
	}
}