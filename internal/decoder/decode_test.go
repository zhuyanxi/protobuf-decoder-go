package decoder

import (
	"strings"
	"testing"
)

func TestDecodeBytesParsesSupportedWireTypes(t *testing.T) {
	data := []byte{
		0x08, 0x96, 0x01,
		0x11, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01,
		0x1a, 0x03, 0x66, 0x6f, 0x6f,
		0x25, 0x78, 0x56, 0x34, 0x12,
	}

	result := DecodeBytes(data, DecodeOptions{MaxFields: 8, MaxBytes: 64})
	if result.Error != "" {
		t.Fatalf("expected no decode error, got %s", result.Error)
	}

	if len(result.Parts) != 4 {
		t.Fatalf("expected 4 parts, got %d", len(result.Parts))
	}

	if result.Parts[0].TypeName != "VARINT" || result.Parts[0].Value[0].CandidateType != "uint64" || result.Parts[0].Value[0].DisplayValue != "150" {
		t.Fatalf("unexpected varint part %#v", result.Parts[0])
	}
	if len(result.Parts[0].Value) != 6 {
		t.Fatalf("expected 6 varint candidates, got %#v", result.Parts[0].Value)
	}

	if result.Parts[1].TypeName != "FIXED64" || result.Parts[1].Value[0].CandidateType != "uint64" || result.Parts[1].Value[0].DisplayValue != "72623859790382856" {
		t.Fatalf("unexpected fixed64 part %#v", result.Parts[1])
	}
	if len(result.Parts[1].Value) != 3 {
		t.Fatalf("expected 3 fixed64 candidates, got %#v", result.Parts[1].Value)
	}

	if result.Parts[2].TypeName != "LENDELIM" || result.Parts[2].Value[0].CandidateType != "string.utf8" || result.Parts[2].Value[0].DisplayValue != "foo" {
		t.Fatalf("unexpected lendelim part %#v", result.Parts[2])
	}
	if len(result.Parts[2].Value) != 2 || result.Parts[2].Value[1].DisplayValue != "666f6f" {
		t.Fatalf("expected string and bytes variants, got %#v", result.Parts[2].Value)
	}

	if result.Parts[3].TypeName != "FIXED32" || result.Parts[3].Value[0].CandidateType != "uint32" || result.Parts[3].Value[0].DisplayValue != "305419896" {
		t.Fatalf("unexpected fixed32 part %#v", result.Parts[3])
	}
	if len(result.Parts[3].Value) != 3 {
		t.Fatalf("expected 3 fixed32 candidates, got %#v", result.Parts[3].Value)
	}
}

func TestDecodeBytesRejectsUnsupportedWireType(t *testing.T) {
	result := DecodeBytes([]byte{0x0b, 0x00}, DecodeOptions{MaxBytes: 8})
	if !strings.Contains(result.Error, string(ErrUnsupportedWireType)) {
		t.Fatalf("expected unsupported wire type error, got %q", result.Error)
	}

	if result.Leftover != "0b00" {
		t.Fatalf("expected leftover 0b00, got %q", result.Leftover)
	}
}

func TestDecodeBytesReturnsParsedPartsAndLeftoverOnTruncatedField(t *testing.T) {
	data := []byte{0x08, 0x01, 0x1a, 0x03, 0x66, 0x6f}
	result := DecodeBytes(data, DecodeOptions{MaxBytes: 64})

	if len(result.Parts) != 1 {
		t.Fatalf("expected one parsed part before truncation, got %d", len(result.Parts))
	}

	if !strings.Contains(result.Error, string(ErrUnexpectedEOF)) {
		t.Fatalf("expected unexpected eof error, got %q", result.Error)
	}

	if result.Leftover != "1a03666f" {
		t.Fatalf("expected leftover 1a03666f, got %q", result.Leftover)
	}
}

func TestDecodeBytesRejectsInvalidFieldNumber(t *testing.T) {
	result := DecodeBytes([]byte{0x00}, DecodeOptions{MaxBytes: 8})
	if !strings.Contains(result.Error, string(ErrInvalidFieldNumber)) {
		t.Fatalf("expected invalid field number error, got %q", result.Error)
	}
}

func TestDecodeBytesEnforcesMaxFields(t *testing.T) {
	data := []byte{0x08, 0x01, 0x10, 0x02}
	result := DecodeBytes(data, DecodeOptions{MaxFields: 1, MaxBytes: 16})

	if len(result.Parts) != 1 {
		t.Fatalf("expected one parsed part before max fields hit, got %d", len(result.Parts))
	}

	if !strings.Contains(result.Error, string(ErrMaxFieldsExceeded)) {
		t.Fatalf("expected max fields error, got %q", result.Error)
	}

	if result.Leftover != "1002" {
		t.Fatalf("expected leftover 1002, got %q", result.Leftover)
	}
}

func TestDecodeBytesEnforcesMaxBytes(t *testing.T) {
	result := DecodeBytes([]byte{0x08, 0x01}, DecodeOptions{MaxBytes: 1})
	if !strings.Contains(result.Error, string(ErrMaxBytesExceeded)) {
		t.Fatalf("expected max bytes error, got %q", result.Error)
	}

	if result.Leftover != "0801" {
		t.Fatalf("expected leftover 0801, got %q", result.Leftover)
	}
}