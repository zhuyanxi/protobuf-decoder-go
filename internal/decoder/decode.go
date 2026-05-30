package decoder

import (
	"encoding/hex"
	"fmt"
	"math"
)

const (
	defaultMaxFields = 256
	defaultMaxBytes  = 1024 * 1024
)

const (
	ErrUnsupportedWireType ErrorKind = "unsupported_wire_type"
	ErrInvalidFieldNumber  ErrorKind = "invalid_field_number"
	ErrMaxFieldsExceeded   ErrorKind = "max_fields_exceeded"
	ErrMaxBytesExceeded    ErrorKind = "max_bytes_exceeded"
)

type DecodeOptions struct {
	ParseDelimited bool
	MaxDepth       int
	MaxFields      int
	MaxBytes       int
}

type DecodeResult struct {
	Parts     []Part
	Leftover  string
	Error     string
	Warnings  []string
	InputSize int
}

type Part struct {
	ByteRange   [2]int
	Index       int
	FieldNumber int
	WireType    int
	TypeName    string
	RawHex      string
	Value       []ValueVariant
	Children    []Part
}

type ValueVariant struct {
	CandidateType string
	DisplayValue  string
	Description   string
	Confidence    string
}

func DecodeBytes(data []byte, options DecodeOptions) DecodeResult {
	resolved := normalizeOptions(options)
	result := DecodeResult{InputSize: len(data)}

	if len(data) > resolved.MaxBytes {
		result.Leftover = hex.EncodeToString(data)
		result.Error = (&ParseError{
			Offset:  0,
			Kind:    ErrMaxBytesExceeded,
			Message: fmt.Sprintf("input size %d exceeds maxBytes %d", len(data), resolved.MaxBytes),
		}).Error()
		return result
	}

	reader := NewBufferReader(data)
	fieldIndex := 0

	for reader.Remaining() > 0 {
		if fieldIndex >= resolved.MaxFields {
			result.Leftover = hex.EncodeToString(data[reader.Position():])
			result.Error = (&ParseError{
				Offset:  reader.Position(),
				Kind:    ErrMaxFieldsExceeded,
				Message: fmt.Sprintf("decoded fields exceeded maxFields %d", resolved.MaxFields),
			}).Error()
			return result
		}

		part, errOffset, err := decodePart(reader, fieldIndex+1)
		if err != nil {
			result.Leftover = hex.EncodeToString(data[errOffset:])
			result.Error = err.Error()
			return result
		}

		result.Parts = append(result.Parts, part)
		fieldIndex++
	}

	return result
}

func decodePart(reader *BufferReader, index int) (Part, int, error) {
	tag, tagStart, _, err := reader.ReadVarint()
	if err != nil {
		return Part{}, tagStart, err
	}

	fieldNumber := int(tag >> 3)
	wireType := int(tag & 0x7)
	if fieldNumber <= 0 {
		return Part{}, tagStart, &ParseError{
			Offset:  tagStart,
			Kind:    ErrInvalidFieldNumber,
			Message: fmt.Sprintf("field number %d is invalid", fieldNumber),
		}
	}

	part := Part{
		Index:       index,
		FieldNumber: fieldNumber,
		WireType:    wireType,
	}

	switch wireType {
	case 0:
		value, _, _, readErr := reader.ReadVarint()
		if readErr != nil {
			return Part{}, tagStart, readErr
		}
		part.TypeName = "VARINT"
		part.ByteRange = [2]int{tagStart, reader.Position()}
		part.RawHex = hex.EncodeToString(reader.data[tagStart:reader.Position()])
		part.Value = buildVarintVariants(value)
		return part, 0, nil
	case 1:
		value, _, _, readErr := reader.ReadFixed64()
		if readErr != nil {
			return Part{}, tagStart, readErr
		}
		part.TypeName = "FIXED64"
		part.ByteRange = [2]int{tagStart, reader.Position()}
		part.RawHex = hex.EncodeToString(reader.data[tagStart:reader.Position()])
		part.Value = buildFixed64Variants(value)
		return part, 0, nil
	case 2:
		lengthValue, _, _, readErr := reader.ReadVarint()
		if readErr != nil {
			return Part{}, tagStart, readErr
		}
		if lengthValue > math.MaxInt {
			return Part{}, tagStart, &ParseError{
				Offset:  tagStart,
				Kind:    ErrInvalidLength,
				Message: fmt.Sprintf("length-delimited payload length %d exceeds platform int", lengthValue),
			}
		}
		payload, _, _, bytesErr := reader.ReadBytes(int(lengthValue))
		if bytesErr != nil {
			return Part{}, tagStart, bytesErr
		}
		part.TypeName = "LENDELIM"
		part.ByteRange = [2]int{tagStart, reader.Position()}
		part.RawHex = hex.EncodeToString(reader.data[tagStart:reader.Position()])
		part.Value = buildLengthDelimitedVariants(payload)
		return part, 0, nil
	case 5:
		value, _, _, readErr := reader.ReadFixed32()
		if readErr != nil {
			return Part{}, tagStart, readErr
		}
		part.TypeName = "FIXED32"
		part.ByteRange = [2]int{tagStart, reader.Position()}
		part.RawHex = hex.EncodeToString(reader.data[tagStart:reader.Position()])
		part.Value = buildFixed32Variants(value)
		return part, 0, nil
	default:
		return Part{}, tagStart, &ParseError{
			Offset:  tagStart,
			Kind:    ErrUnsupportedWireType,
			Message: fmt.Sprintf("wire type %d is unsupported", wireType),
		}
	}
}

func normalizeOptions(options DecodeOptions) DecodeOptions {
	resolved := options
	if resolved.MaxFields <= 0 {
		resolved.MaxFields = defaultMaxFields
	}
	if resolved.MaxBytes <= 0 {
		resolved.MaxBytes = defaultMaxBytes
	}
	return resolved
}
