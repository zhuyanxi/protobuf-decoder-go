package decoder
package decoder

import (
	"encoding/binary"
	"fmt"
)

const maxVarintBytes = 10

type ErrorKind string

const (
	ErrUnexpectedEOF   ErrorKind = "unexpected_eof"
	ErrVarintOverflow  ErrorKind = "varint_overflow"
	ErrInvalidLength   ErrorKind = "invalid_length"
	ErrOffsetOutOfRange ErrorKind = "offset_out_of_range"
)

type ParseError struct {
	Offset  int
	Kind    ErrorKind
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("decoder error at offset %d (%s): %s", e.Offset, e.Kind, e.Message)
}

type BufferReader struct {
	data   []byte
	offset int
	limit  int
}

func NewBufferReader(data []byte) *BufferReader {
	return &BufferReader{
		data:  data,
		limit: len(data),
	}
}

func (r *BufferReader) Position() int {
	return r.offset
}

func (r *BufferReader) Remaining() int {
	if r.offset >= r.limit {
		return 0
	}

	return r.limit - r.offset
}

func (r *BufferReader) ByteRange(start int) [2]int {
	return [2]int{start, r.offset}
}

func (r *BufferReader) ReadVarint() (uint64, int, int, error) {
	start := r.offset
	if start >= r.limit {
		return 0, start, start, &ParseError{
			Offset:  start,
			Kind:    ErrUnexpectedEOF,
			Message: "no bytes available for varint",
		}
	}

	var value uint64
	for index := 0; index < maxVarintBytes; index++ {
		position := start + index
		if position >= r.limit {
			return 0, start, start, &ParseError{
				Offset:  position,
				Kind:    ErrUnexpectedEOF,
				Message: "truncated varint",
			}
		}

		current := r.data[position]
		if index == maxVarintBytes-1 && current > 1 {
			return 0, start, start, &ParseError{
				Offset:  start,
				Kind:    ErrVarintOverflow,
				Message: "varint exceeds 64-bit limit",
			}
		}

		value |= uint64(current&0x7f) << (7 * index)
		if current < 0x80 {
			end := position + 1
			r.offset = end
			return value, start, end, nil
		}
	}

	return 0, start, start, &ParseError{
		Offset:  start,
		Kind:    ErrVarintOverflow,
		Message: "varint exceeds 10 bytes",
	}
}

func (r *BufferReader) ReadFixed32() (uint32, int, int, error) {
	start := r.offset
	end := start + 4
	if end > r.limit {
		return 0, start, start, &ParseError{
			Offset:  start,
			Kind:    ErrUnexpectedEOF,
			Message: "truncated fixed32",
		}
	}

	value := binary.LittleEndian.Uint32(r.data[start:end])
	r.offset = end
	return value, start, end, nil
}

func (r *BufferReader) ReadFixed64() (uint64, int, int, error) {
	start := r.offset
	end := start + 8
	if end > r.limit {
		return 0, start, start, &ParseError{
			Offset:  start,
			Kind:    ErrUnexpectedEOF,
			Message: "truncated fixed64",
		}
	}

	value := binary.LittleEndian.Uint64(r.data[start:end])
	r.offset = end
	return value, start, end, nil
}

func (r *BufferReader) ReadBytes(length int) ([]byte, int, int, error) {
	start := r.offset
	if length < 0 {
		return nil, start, start, &ParseError{
			Offset:  start,
			Kind:    ErrInvalidLength,
			Message: "negative byte length",
		}
	}

	end := start + length
	if end < start {
		return nil, start, start, &ParseError{
			Offset:  start,
			Kind:    ErrOffsetOutOfRange,
			Message: "byte length overflowed reader offset",
		}
	}

	if end > r.limit {
		return nil, start, start, &ParseError{
			Offset:  start,
			Kind:    ErrUnexpectedEOF,
			Message: fmt.Sprintf("requested %d bytes with %d remaining", length, r.Remaining()),
		}
	}

	segment := r.data[start:end]
	r.offset = end
	return segment, start, end, nil
}