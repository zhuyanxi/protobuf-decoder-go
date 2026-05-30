package decoder

import (
	"errors"
	"reflect"
	"testing"
)

func TestBufferReaderReadVarint(t *testing.T) {
	reader := NewBufferReader([]byte{0x96, 0x01, 0xff})

	value, start, end, err := reader.ReadVarint()
	if err != nil {
		t.Fatalf("read varint: %v", err)
	}

	if value != 150 {
		t.Fatalf("expected varint value 150, got %d", value)
	}

	if start != 0 || end != 2 {
		t.Fatalf("expected range [0,2), got [%d,%d)", start, end)
	}

	if reader.Position() != 2 {
		t.Fatalf("expected reader position 2, got %d", reader.Position())
	}

	if got := reader.ByteRange(start); got != [2]int{0, 2} {
		t.Fatalf("expected byte range [0 2], got %v", got)
	}
}

func TestBufferReaderReadVarintOverflowDoesNotAdvanceOffset(t *testing.T) {
	reader := NewBufferReader([]byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x82})

	_, start, end, err := reader.ReadVarint()
	if err == nil {
		t.Fatal("expected varint overflow error")
	}

	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParseError, got %T", err)
	}

	if parseErr.Kind != ErrVarintOverflow {
		t.Fatalf("expected ErrVarintOverflow, got %s", parseErr.Kind)
	}

	if start != 0 || end != 0 {
		t.Fatalf("expected unchanged range [0,0), got [%d,%d)", start, end)
	}

	if reader.Position() != 0 {
		t.Fatalf("expected reader position to remain 0, got %d", reader.Position())
	}
}

func TestBufferReaderReadFixed32(t *testing.T) {
	reader := NewBufferReader([]byte{0x78, 0x56, 0x34, 0x12, 0xaa})

	value, start, end, err := reader.ReadFixed32()
	if err != nil {
		t.Fatalf("read fixed32: %v", err)
	}

	if value != 0x12345678 {
		t.Fatalf("expected fixed32 0x12345678, got %#x", value)
	}

	if start != 0 || end != 4 {
		t.Fatalf("expected range [0,4), got [%d,%d)", start, end)
	}

	if reader.Position() != 4 || reader.Remaining() != 1 {
		t.Fatalf("expected position 4 remaining 1, got position %d remaining %d", reader.Position(), reader.Remaining())
	}
}

func TestBufferReaderReadFixed64(t *testing.T) {
	reader := NewBufferReader([]byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01})

	value, start, end, err := reader.ReadFixed64()
	if err != nil {
		t.Fatalf("read fixed64: %v", err)
	}

	if value != 0x0102030405060708 {
		t.Fatalf("expected fixed64 0x0102030405060708, got %#x", value)
	}

	if start != 0 || end != 8 {
		t.Fatalf("expected range [0,8), got [%d,%d)", start, end)
	}

	if reader.Position() != 8 || reader.Remaining() != 0 {
		t.Fatalf("expected position 8 remaining 0, got position %d remaining %d", reader.Position(), reader.Remaining())
	}
}

func TestBufferReaderReadBytes(t *testing.T) {
	reader := NewBufferReader([]byte{0x01, 0x02, 0x03, 0x04})

	value, start, end, err := reader.ReadBytes(3)
	if err != nil {
		t.Fatalf("read bytes: %v", err)
	}

	if !reflect.DeepEqual(value, []byte{0x01, 0x02, 0x03}) {
		t.Fatalf("expected [1 2 3], got %v", value)
	}

	if start != 0 || end != 3 {
		t.Fatalf("expected range [0,3), got [%d,%d)", start, end)
	}

	if reader.Position() != 3 || reader.Remaining() != 1 {
		t.Fatalf("expected position 3 remaining 1, got position %d remaining %d", reader.Position(), reader.Remaining())
	}
}

func TestBufferReaderReadBytesTruncatedDoesNotAdvanceOffset(t *testing.T) {
	reader := NewBufferReader([]byte{0x01, 0x02})

	_, start, end, err := reader.ReadBytes(3)
	if err == nil {
		t.Fatal("expected truncated bytes error")
	}

	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParseError, got %T", err)
	}

	if parseErr.Kind != ErrUnexpectedEOF {
		t.Fatalf("expected ErrUnexpectedEOF, got %s", parseErr.Kind)
	}

	if start != 0 || end != 0 {
		t.Fatalf("expected unchanged range [0,0), got [%d,%d)", start, end)
	}

	if reader.Position() != 0 {
		t.Fatalf("expected reader position 0, got %d", reader.Position())
	}
}

func TestBufferReaderReadFixed32TruncatedDoesNotAdvanceOffset(t *testing.T) {
	reader := NewBufferReader([]byte{0x01, 0x02, 0x03})

	_, start, end, err := reader.ReadFixed32()
	if err == nil {
		t.Fatal("expected truncated fixed32 error")
	}

	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected ParseError, got %T", err)
	}

	if parseErr.Kind != ErrUnexpectedEOF {
		t.Fatalf("expected ErrUnexpectedEOF, got %s", parseErr.Kind)
	}

	if start != 0 || end != 0 || reader.Position() != 0 {
		t.Fatalf("expected unchanged reader state, got range [%d,%d) position %d", start, end, reader.Position())
	}
}