package input

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeTextHexWithSeparators(t *testing.T) {
	result, err := NormalizeText("0x0a 03-66:6f_6f", "hex", 32)
	if err != nil {
		t.Fatalf("normalize hex: %v", err)
	}

	if result.Encoding != "hex" {
		t.Fatalf("expected hex encoding, got %q", result.Encoding)
	}

	if got := string(result.Bytes); got != "\n\x03foo" {
		t.Fatalf("expected decoded bytes for foo payload, got %q", got)
	}
}

func TestNormalizeTextBase64(t *testing.T) {
	result, err := NormalizeText("CgNmb28=", "base64", 32)
	if err != nil {
		t.Fatalf("normalize base64: %v", err)
	}

	if result.Encoding != "base64" {
		t.Fatalf("expected base64 encoding, got %q", result.Encoding)
	}

	if got := string(result.Bytes); got != "\n\x03foo" {
		t.Fatalf("expected decoded bytes for foo payload, got %q", got)
	}
}

func TestNormalizeTextAutoPrefersHexOnAmbiguity(t *testing.T) {
	result, err := NormalizeText("deadbeef", "auto", 32)
	if err != nil {
		t.Fatalf("normalize auto: %v", err)
	}

	if result.Encoding != "hex" {
		t.Fatalf("expected auto to choose hex, got %q", result.Encoding)
	}

	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "Auto-detected") {
		t.Fatalf("expected auto-detect warning, got %#v", result.Warnings)
	}
}

func TestNormalizeTextHexRejectsInvalidCharacter(t *testing.T) {
	_, err := NormalizeText("0a03zz", "hex", 32)
	if err == nil {
		t.Fatal("expected invalid hex error")
	}

	if !strings.Contains(err.Error(), "invalid hex character") {
		t.Fatalf("expected invalid hex error, got %v", err)
	}
}

func TestNormalizeTextRejectsOversizedPayload(t *testing.T) {
	_, err := NormalizeText("0a03666f6f", "hex", 2)
	if err == nil {
		t.Fatal("expected maxBytes error")
	}

	if !strings.Contains(err.Error(), "exceeds maxBytes") {
		t.Fatalf("expected maxBytes error, got %v", err)
	}
}

func TestLoadFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "payload.bin")
	if err := os.WriteFile(filePath, []byte{0x0a, 0x03, 0x66, 0x6f, 0x6f}, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	data, err := LoadFile(filePath, 16)
	if err != nil {
		t.Fatalf("load file: %v", err)
	}

	if string(data) != "\n\x03foo" {
		t.Fatalf("expected raw file bytes, got %q", string(data))
	}
}

func TestLoadFileRejectsOversizedFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "payload.bin")
	if err := os.WriteFile(filePath, []byte{0x0a, 0x03, 0x66, 0x6f, 0x6f}, 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err := LoadFile(filePath, 4)
	if err == nil {
		t.Fatal("expected oversized file error")
	}

	if !strings.Contains(err.Error(), "exceeds maxBytes") {
		t.Fatalf("expected maxBytes error, got %v", err)
	}
}