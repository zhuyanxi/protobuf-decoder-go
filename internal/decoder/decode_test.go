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

func TestDecodeBytesEnforcesMaxFieldsAcrossNestedMessages(t *testing.T) {
	data := []byte{0x0a, 0x02, 0x08, 0x01, 0x10, 0x02}
	result := DecodeBytes(data, DecodeOptions{MaxFields: 2, MaxBytes: 32, MaxDepth: 4})

	if len(result.Parts) != 1 {
		t.Fatalf("expected parent field before global max fields hit, got %#v", result.Parts)
	}

	if len(result.Parts[0].Children) != 1 {
		t.Fatalf("expected nested child to consume shared field budget, got %#v", result.Parts[0].Children)
	}

	if !strings.Contains(result.Error, string(ErrMaxFieldsExceeded)) {
		t.Fatalf("expected global max fields error, got %q", result.Error)
	}

	if result.Leftover != "1002" {
		t.Fatalf("expected leftover 1002 after nested field consumed global budget, got %q", result.Leftover)
	}
}

func TestDecodeBytesEnforcesMaxFieldsAcrossDelimitedMessages(t *testing.T) {
	data := []byte{0x02, 0x08, 0x01, 0x02, 0x10, 0x02}
	result := DecodeBytes(data, DecodeOptions{MaxFields: 1, MaxBytes: 32, ParseDelimited: true})

	if len(result.Parts) != 3 {
		t.Fatalf("expected first delimiter, first field, and second delimiter before global max fields hit, got %#v", result.Parts)
	}

	if !strings.Contains(result.Error, string(ErrMaxFieldsExceeded)) {
		t.Fatalf("expected global max fields error, got %q", result.Error)
	}

	if result.Leftover != "1002" {
		t.Fatalf("expected leftover 1002 for second message payload, got %q", result.Leftover)
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

func TestDecodeBytesAddsNestedChildrenForCompleteNestedPayload(t *testing.T) {
	data := []byte{0x0a, 0x02, 0x08, 0x01}
	result := DecodeBytes(data, DecodeOptions{MaxBytes: 32, MaxDepth: 4})

	if result.Error != "" {
		t.Fatalf("expected no error, got %q", result.Error)
	}

	if len(result.Parts) != 1 {
		t.Fatalf("expected one top-level part, got %d", len(result.Parts))
	}

	if len(result.Parts[0].Children) != 1 {
		t.Fatalf("expected one nested child, got %#v", result.Parts[0].Children)
	}

	if result.Parts[0].Value[0].CandidateType != "nested.protobuf" {
		t.Fatalf("expected nested.protobuf candidate first, got %#v", result.Parts[0].Value)
	}

	if result.Parts[0].Children[0].TypeName != "VARINT" || result.Parts[0].Children[0].Value[0].DisplayValue != "1" {
		t.Fatalf("unexpected nested child %#v", result.Parts[0].Children[0])
	}
}

func TestDecodeBytesRejectsPartialNestedCandidate(t *testing.T) {
	data := []byte{0x0a, 0x03, 0x08, 0x01, 0xff}
	result := DecodeBytes(data, DecodeOptions{MaxBytes: 32, MaxDepth: 4})

	if result.Error != "" {
		t.Fatalf("expected top-level decode success, got %q", result.Error)
	}

	if len(result.Parts) != 1 {
		t.Fatalf("expected one top-level part, got %d", len(result.Parts))
	}

	if len(result.Parts[0].Children) != 0 {
		t.Fatalf("expected rejected nested candidate to keep no children, got %#v", result.Parts[0].Children)
	}

	if result.Parts[0].Value[0].CandidateType == "nested.protobuf" {
		t.Fatalf("expected no nested.protobuf candidate on partial nested parse, got %#v", result.Parts[0].Value)
	}

	if len(result.Warnings) == 0 || !strings.Contains(strings.Join(result.Warnings, " | "), "Nested protobuf candidate rejected") {
		t.Fatalf("expected nested rejection warning, got %#v", result.Warnings)
	}
}

func TestDecodeBytesHonorsMaxDepthForNestedPayload(t *testing.T) {
	data := []byte{0x0a, 0x04, 0x0a, 0x02, 0x08, 0x01}
	result := DecodeBytes(data, DecodeOptions{MaxBytes: 32, MaxDepth: 1})

	if result.Error != "" {
		t.Fatalf("expected top-level decode success, got %q", result.Error)
	}

	if len(result.Parts) != 1 || len(result.Parts[0].Children) != 1 {
		t.Fatalf("expected one nested child at first level, got %#v", result.Parts)
	}

	if len(result.Parts[0].Children[0].Children) != 0 {
		t.Fatalf("expected no grand-children once maxDepth hit, got %#v", result.Parts[0].Children[0].Children)
	}

	if len(result.Warnings) == 0 || !strings.Contains(strings.Join(result.Warnings, " | "), "maxDepth 1 reached") {
		t.Fatalf("expected maxDepth warning, got %#v", result.Warnings)
	}
}

func TestDecodeBytesSkipsValidGRPCHeader(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x00, 0x02, 0x08, 0x01}
	result := DecodeBytes(data, DecodeOptions{MaxBytes: 32})

	if result.Error != "" {
		t.Fatalf("expected no error, got %q", result.Error)
	}

	if len(result.Parts) != 2 {
		t.Fatalf("expected header part plus body field, got %#v", result.Parts)
	}

	if result.Parts[0].TypeName != "GRPC_HEADER" || result.Parts[0].RawHex != "0000000002" {
		t.Fatalf("unexpected grpc header part %#v", result.Parts[0])
	}

	if result.Parts[1].TypeName != "VARINT" || result.Parts[1].ByteRange != [2]int{5, 7} {
		t.Fatalf("expected body field byte range shifted past header, got %#v", result.Parts[1])
	}

	if len(result.Warnings) == 0 || !strings.Contains(strings.Join(result.Warnings, " | "), "Detected gRPC message header") {
		t.Fatalf("expected grpc detection warning, got %#v", result.Warnings)
	}
}

func TestDecodeBytesDoesNotSkipInvalidGRPCHeaderLength(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x00, 0x01, 0x08, 0x01}
	result := DecodeBytes(data, DecodeOptions{MaxBytes: 32})

	if !strings.Contains(result.Error, string(ErrInvalidFieldNumber)) {
		t.Fatalf("expected normal protobuf parse failure, got %q", result.Error)
	}

	if len(result.Parts) != 0 {
		t.Fatalf("expected no parsed parts when header not skipped, got %#v", result.Parts)
	}
}

func TestDecodeBytesRejectsTruncatedGRPCPayload(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x00, 0x03, 0x08, 0x01}
	result := DecodeBytes(data, DecodeOptions{MaxBytes: 32})

	if !strings.Contains(result.Error, string(ErrTruncatedGRPCMessage)) {
		t.Fatalf("expected truncated grpc error, got %q", result.Error)
	}

	if result.Leftover != "00000000030801" {
		t.Fatalf("expected full leftover on grpc truncation, got %q", result.Leftover)
	}
}

func TestDecodeBytesRejectsCompressedGRPCPayload(t *testing.T) {
	data := []byte{0x01, 0x00, 0x00, 0x00, 0x02, 0x78, 0x9c}
	result := DecodeBytes(data, DecodeOptions{MaxBytes: 32})

	if !strings.Contains(result.Error, string(ErrUnsupportedGRPCCompression)) {
		t.Fatalf("expected unsupported grpc compression error, got %q", result.Error)
	}

	if result.Leftover != "0100000002789c" {
		t.Fatalf("expected full leftover on compressed grpc payload, got %q", result.Leftover)
	}
}

func TestDecodeBytesParsesDelimitedMessageStream(t *testing.T) {
	data := []byte{0x02, 0x08, 0x01, 0x02, 0x10, 0x02}
	result := DecodeBytes(data, DecodeOptions{MaxBytes: 32, ParseDelimited: true})

	if result.Error != "" {
		t.Fatalf("expected no error, got %q", result.Error)
	}

	if len(result.Parts) != 4 {
		t.Fatalf("expected two delimiters and two fields, got %#v", result.Parts)
	}

	if result.Parts[0].TypeName != "MessageDelimiter" || result.Parts[0].ByteRange != [2]int{0, 1} {
		t.Fatalf("unexpected first delimiter %#v", result.Parts[0])
	}

	if result.Parts[1].TypeName != "VARINT" || result.Parts[1].ByteRange != [2]int{1, 3} {
		t.Fatalf("unexpected first message field %#v", result.Parts[1])
	}

	if result.Parts[2].TypeName != "MessageDelimiter" || result.Parts[2].ByteRange != [2]int{3, 4} {
		t.Fatalf("unexpected second delimiter %#v", result.Parts[2])
	}

	if result.Parts[3].TypeName != "VARINT" || result.Parts[3].ByteRange != [2]int{4, 6} {
		t.Fatalf("unexpected second message field %#v", result.Parts[3])
	}
}

func TestDecodeBytesRejectsTruncatedDelimitedMessage(t *testing.T) {
	data := []byte{0x03, 0x08, 0x01}
	result := DecodeBytes(data, DecodeOptions{MaxBytes: 32, ParseDelimited: true})

	if !strings.Contains(result.Error, string(ErrTruncatedDelimitedMessage)) {
		t.Fatalf("expected truncated delimited message error, got %q", result.Error)
	}

	if result.Leftover != "030801" {
		t.Fatalf("expected full leftover on truncated delimited message, got %q", result.Leftover)
	}
}

func TestDecodeBytesReturnsRemainingStreamAfterDelimitedMessageError(t *testing.T) {
	data := []byte{0x02, 0x08, 0x01, 0x02, 0x1a, 0x03, 0x02, 0x10, 0x02}
	result := DecodeBytes(data, DecodeOptions{MaxBytes: 32, ParseDelimited: true})

	if len(result.Parts) != 3 {
		t.Fatalf("expected first delimiter, first field, second delimiter before error, got %#v", result.Parts)
	}

	if !strings.Contains(result.Error, string(ErrUnexpectedEOF)) {
		t.Fatalf("expected internal message parse error, got %q", result.Error)
	}

	if result.Leftover != "1a03021002" {
		t.Fatalf("expected leftover from broken message through end of stream, got %q", result.Leftover)
	}
}

func TestDecodeBytesRejectsDelimitedLengthVarintOverflow(t *testing.T) {
	data := []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02}
	result := DecodeBytes(data, DecodeOptions{MaxBytes: 32, ParseDelimited: true})

	if !strings.Contains(result.Error, string(ErrVarintOverflow)) {
		t.Fatalf("expected varint overflow on delimiter, got %q", result.Error)
	}

	if result.Leftover != "8080808080808080808002" {
		t.Fatalf("expected full leftover on delimiter overflow, got %q", result.Leftover)
	}
}
