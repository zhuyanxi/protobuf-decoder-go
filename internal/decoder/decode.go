package decoder

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
)

const (
	defaultMaxDepth  = 4
	defaultMaxFields = 256
	defaultMaxBytes  = 10 * 1024 * 1024
)

const (
	ErrUnsupportedWireType ErrorKind = "unsupported_wire_type"
	ErrInvalidFieldNumber  ErrorKind = "invalid_field_number"
	ErrMaxFieldsExceeded   ErrorKind = "max_fields_exceeded"
	ErrMaxBytesExceeded    ErrorKind = "max_bytes_exceeded"
	ErrTruncatedDelimitedMessage ErrorKind = "truncated_delimited_message"
	ErrTruncatedGRPCMessage ErrorKind = "truncated_grpc_message"
	ErrUnsupportedGRPCCompression ErrorKind = "unsupported_grpc_compression"
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
	return decodeBytesAtDepth(data, resolved, 0, 0, true, true)
}

func decodeBytesAtDepth(data []byte, options DecodeOptions, depth int, baseOffset int, detectGRPC bool, allowDelimited bool) DecodeResult {
	resolved := normalizeOptions(options)
	result := DecodeResult{InputSize: len(data)}

	if len(data) > resolved.MaxBytes {
		result.Leftover = hex.EncodeToString(data)
		result.Error = (&ParseError{
			Offset:  baseOffset,
			Kind:    ErrMaxBytesExceeded,
			Message: fmt.Sprintf("input size %d exceeds maxBytes %d", len(data), resolved.MaxBytes),
		}).Error()
		return result
	}

	if detectGRPC {
		header, matched, grpcErr := detectGRPCHeader(data)
		if grpcErr != nil {
			result.Leftover = hex.EncodeToString(data)
			result.Error = grpcErr.Error()
			return result
		}
		if matched {
			bodyResult := decodeBytesAtDepth(data[5:], resolved, depth, baseOffset+5, false, allowDelimited)
			bodyResult.Parts = append([]Part{buildGRPCHeaderPart(header)}, bodyResult.Parts...)
			bodyResult.InputSize = len(data)
			bodyResult.Warnings = append([]string{fmt.Sprintf("Detected gRPC message header: skipped 5 bytes, message length %d.", header.MessageLength)}, bodyResult.Warnings...)
			return bodyResult
		}
	}

	if allowDelimited && resolved.ParseDelimited {
		return decodeDelimitedStream(data, resolved, baseOffset)
	}

	reader := NewBufferReader(data)
	fieldIndex := 0

	for reader.Remaining() > 0 {
		if fieldIndex >= resolved.MaxFields {
			result.Leftover = hex.EncodeToString(data[reader.Position():])
			result.Error = (&ParseError{
				Offset:  baseOffset + reader.Position(),
				Kind:    ErrMaxFieldsExceeded,
				Message: fmt.Sprintf("decoded fields exceeded maxFields %d", resolved.MaxFields),
			}).Error()
			return result
		}

		part, warnings, errOffset, err := decodePart(reader, fieldIndex+1, resolved, depth, baseOffset)
		if len(warnings) > 0 {
			result.Warnings = append(result.Warnings, warnings...)
		}
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

func decodePart(reader *BufferReader, index int, options DecodeOptions, depth int, baseOffset int) (Part, []string, int, error) {
	tag, tagStart, _, err := reader.ReadVarint()
	if err != nil {
		return Part{}, nil, tagStart, err
	}

	fieldNumber := int(tag >> 3)
	wireType := int(tag & 0x7)
	if fieldNumber <= 0 {
		return Part{}, nil, tagStart, &ParseError{
			Offset:  baseOffset + tagStart,
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
			return Part{}, nil, tagStart, readErr
		}
		part.TypeName = "VARINT"
		part.ByteRange = [2]int{baseOffset + tagStart, baseOffset + reader.Position()}
		part.RawHex = hex.EncodeToString(reader.data[tagStart:reader.Position()])
		part.Value = buildVarintVariants(value)
		return part, nil, 0, nil
	case 1:
		value, _, _, readErr := reader.ReadFixed64()
		if readErr != nil {
			return Part{}, nil, tagStart, readErr
		}
		part.TypeName = "FIXED64"
		part.ByteRange = [2]int{baseOffset + tagStart, baseOffset + reader.Position()}
		part.RawHex = hex.EncodeToString(reader.data[tagStart:reader.Position()])
		part.Value = buildFixed64Variants(value)
		return part, nil, 0, nil
	case 2:
		lengthValue, _, _, readErr := reader.ReadVarint()
		if readErr != nil {
			return Part{}, nil, tagStart, readErr
		}
		if lengthValue > math.MaxInt {
			return Part{}, nil, tagStart, &ParseError{
				Offset:  baseOffset + tagStart,
				Kind:    ErrInvalidLength,
				Message: fmt.Sprintf("length-delimited payload length %d exceeds platform int", lengthValue),
			}
		}
		payload, _, _, bytesErr := reader.ReadBytes(int(lengthValue))
		if bytesErr != nil {
			return Part{}, nil, tagStart, bytesErr
		}
		part.TypeName = "LENDELIM"
		part.ByteRange = [2]int{baseOffset + tagStart, baseOffset + reader.Position()}
		part.RawHex = hex.EncodeToString(reader.data[tagStart:reader.Position()])
		part.Value = buildLengthDelimitedVariants(payload)

		warnings := make([]string, 0)
		if len(payload) > 0 {
			if depth >= options.MaxDepth {
				warnings = append(warnings, fmt.Sprintf("Nested decode skipped for field %d: maxDepth %d reached.", fieldNumber, options.MaxDepth))
			} else {
				nested := decodeBytesAtDepth(payload, options, depth+1, 0, false, false)
				if nested.Error == "" && nested.Leftover == "" && len(nested.Parts) > 0 {
					part.Children = nested.Parts
					part.Value = append([]ValueVariant{nestedMessageVariant(len(nested.Parts))}, part.Value...)
					warnings = append(warnings, nested.Warnings...)
				} else if len(nested.Parts) > 0 || len(nested.Warnings) > 0 {
					warnings = append(warnings, nested.Warnings...)
					warnings = append(warnings, fmt.Sprintf("Nested protobuf candidate rejected for field %d: %s", fieldNumber, nestedFailureReason(nested)))
				}
			}
		}

		return part, warnings, 0, nil
	case 5:
		value, _, _, readErr := reader.ReadFixed32()
		if readErr != nil {
			return Part{}, nil, tagStart, readErr
		}
		part.TypeName = "FIXED32"
		part.ByteRange = [2]int{baseOffset + tagStart, baseOffset + reader.Position()}
		part.RawHex = hex.EncodeToString(reader.data[tagStart:reader.Position()])
		part.Value = buildFixed32Variants(value)
		return part, nil, 0, nil
	default:
		return Part{}, nil, tagStart, &ParseError{
			Offset:  baseOffset + tagStart,
			Kind:    ErrUnsupportedWireType,
			Message: fmt.Sprintf("wire type %d is unsupported", wireType),
		}
	}
}

func normalizeOptions(options DecodeOptions) DecodeOptions {
	resolved := options
	if resolved.MaxDepth <= 0 {
		resolved.MaxDepth = defaultMaxDepth
	}
	if resolved.MaxFields <= 0 {
		resolved.MaxFields = defaultMaxFields
	}
	if resolved.MaxBytes <= 0 {
		resolved.MaxBytes = defaultMaxBytes
	}
	return resolved
}

func nestedFailureReason(result DecodeResult) string {
	if result.Error != "" {
		return result.Error
	}
	if result.Leftover != "" {
		return fmt.Sprintf("leftover bytes %s", result.Leftover)
	}
	return "payload did not fully parse as nested protobuf"
}

func decodeDelimitedStream(data []byte, options DecodeOptions, baseOffset int) DecodeResult {
	resolved := normalizeOptions(options)
	result := DecodeResult{InputSize: len(data)}
	reader := NewBufferReader(data)
	messageIndex := 0
	messageOptions := resolved
	messageOptions.ParseDelimited = false

	for reader.Remaining() > 0 {
		messageIndex++
		delimiterStart := reader.Position()
		lengthValue, _, delimiterEnd, err := reader.ReadVarint()
		if err != nil {
			result.Leftover = hex.EncodeToString(data[delimiterStart:])
			result.Error = err.Error()
			return result
		}

		if lengthValue > math.MaxInt {
			result.Leftover = hex.EncodeToString(data[delimiterStart:])
			result.Error = (&ParseError{
				Offset:  baseOffset + delimiterStart,
				Kind:    ErrInvalidLength,
				Message: fmt.Sprintf("delimited message length %d exceeds platform int", lengthValue),
			}).Error()
			return result
		}

		messageLength := int(lengthValue)
		if messageLength > reader.Remaining() {
			result.Leftover = hex.EncodeToString(data[delimiterStart:])
			result.Error = (&ParseError{
				Offset:  baseOffset + delimiterStart,
				Kind:    ErrTruncatedDelimitedMessage,
				Message: fmt.Sprintf("delimited message length %d exceeds remaining payload %d", messageLength, reader.Remaining()),
			}).Error()
			return result
		}

		result.Parts = append(result.Parts, buildMessageDelimiterPart(
			messageIndex,
			baseOffset+delimiterStart,
			baseOffset+delimiterEnd,
			data[delimiterStart:delimiterEnd],
			messageLength,
		))

		payloadStart := reader.Position()
		payload, _, _, readErr := reader.ReadBytes(messageLength)
		if readErr != nil {
			result.Leftover = hex.EncodeToString(data[delimiterStart:])
			result.Error = readErr.Error()
			return result
		}

		messageResult := decodeBytesAtDepth(payload, messageOptions, 0, baseOffset+payloadStart, false, false)
		result.Parts = append(result.Parts, messageResult.Parts...)
		result.Warnings = append(result.Warnings, messageResult.Warnings...)
		if messageResult.Error != "" {
			result.Error = messageResult.Error
			leftoverStart := payloadStart
			if messageResult.Leftover != "" {
				leftoverBytes := len(messageResult.Leftover) / 2
				leftoverStart = payloadStart + messageLength - leftoverBytes
			}
			result.Leftover = hex.EncodeToString(data[leftoverStart:])
			return result
		}
	}

	return result
}

type grpcHeaderInfo struct {
	Compressed   bool
	MessageLength int
	RawHex       string
}

func detectGRPCHeader(data []byte) (grpcHeaderInfo, bool, error) {
	if len(data) < 5 {
		return grpcHeaderInfo{}, false, nil
	}

	flag := data[0]
	if flag != 0 && flag != 1 {
		return grpcHeaderInfo{}, false, nil
	}

	messageLength := uint64(binary.BigEndian.Uint32(data[1:5]))
	remaining := uint64(len(data) - 5)
	header := grpcHeaderInfo{
		Compressed:   flag != 0,
		MessageLength: int(messageLength),
		RawHex:       hex.EncodeToString(data[:5]),
	}

	if messageLength > uint64(math.MaxInt) {
		return grpcHeaderInfo{}, false, &ParseError{
			Offset:  0,
			Kind:    ErrInvalidLength,
			Message: fmt.Sprintf("gRPC frame length %d exceeds platform int", messageLength),
		}
	}

	if messageLength > remaining {
		return grpcHeaderInfo{}, false, &ParseError{
			Offset:  0,
			Kind:    ErrTruncatedGRPCMessage,
			Message: fmt.Sprintf("gRPC frame length %d exceeds remaining payload %d", messageLength, remaining),
		}
	}

	if messageLength != remaining {
		return grpcHeaderInfo{}, false, nil
	}

	if header.Compressed {
		return grpcHeaderInfo{}, false, &ParseError{
			Offset:  0,
			Kind:    ErrUnsupportedGRPCCompression,
			Message: "gRPC compressed messages are not supported",
		}
	}

	return header, true, nil
}

func buildGRPCHeaderPart(header grpcHeaderInfo) Part {
	return Part{
		ByteRange:   [2]int{0, 5},
		Index:       0,
		FieldNumber: 0,
		WireType:    -1,
		TypeName:    "GRPC_HEADER",
		RawHex:      header.RawHex,
		Value: []ValueVariant{
			{
				CandidateType: "grpc.compressed",
				DisplayValue:  fmt.Sprintf("%t", header.Compressed),
				Description:   "gRPC frame compression flag.",
				Confidence:    "confirmed",
			},
			{
				CandidateType: "grpc.message_length",
				DisplayValue:  fmt.Sprintf("%d", header.MessageLength),
				Description:   "gRPC frame message length in bytes.",
				Confidence:    "confirmed",
			},
		},
	}
}

func buildMessageDelimiterPart(messageIndex int, start int, end int, raw []byte, messageLength int) Part {
	return Part{
		ByteRange:   [2]int{start, end},
		Index:       messageIndex,
		FieldNumber: 0,
		WireType:    -1,
		TypeName:    "MessageDelimiter",
		RawHex:      hex.EncodeToString(raw),
		Value: []ValueVariant{
			{
				CandidateType: "delimited.message_index",
				DisplayValue:  fmt.Sprintf("%d", messageIndex),
				Description:   "Delimited stream message index.",
				Confidence:    "confirmed",
			},
			{
				CandidateType: "delimited.message_length",
				DisplayValue:  fmt.Sprintf("%d", messageLength),
				Description:   "Delimited stream message length in bytes.",
				Confidence:    "confirmed",
			},
		},
	}
}
