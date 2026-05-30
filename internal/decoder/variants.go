package decoder

import (
	"encoding/hex"
	"math"
	"strconv"
	"unicode/utf8"
)

func buildVarintVariants(value uint64) []ValueVariant {
	variants := []ValueVariant{
		{
			CandidateType: "uint64",
			DisplayValue:  strconv.FormatUint(value, 10),
			Description:   "Unsigned protobuf varint interpretation.",
			Confidence:    "candidate",
		},
		{
			CandidateType: "int32",
			DisplayValue:  strconv.FormatInt(int64(int32(uint32(value))), 10),
			Description:   "32-bit two's complement interpretation.",
			Confidence:    "candidate",
		},
		{
			CandidateType: "int64",
			DisplayValue:  strconv.FormatInt(int64(value), 10),
			Description:   "64-bit two's complement interpretation.",
			Confidence:    "candidate",
		},
		{
			CandidateType: "sint32",
			DisplayValue:  strconv.FormatInt(int64(decodeZigZag32(uint32(value))), 10),
			Description:   "ZigZag-decoded 32-bit signed interpretation.",
			Confidence:    "candidate",
		},
		{
			CandidateType: "sint64",
			DisplayValue:  strconv.FormatInt(decodeZigZag64(value), 10),
			Description:   "ZigZag-decoded 64-bit signed interpretation.",
			Confidence:    "candidate",
		},
	}

	boolHint := "bool false if value=0, true if value=1; otherwise likely enum/int"
	if value == 0 {
		boolHint = "false"
	} else if value == 1 {
		boolHint = "true"
	}

	variants = append(variants, ValueVariant{
		CandidateType: "bool.enum_hint",
		DisplayValue:  boolHint,
		Description:   "Schema may define this varint as bool or enum.",
		Confidence:    "hint",
	})

	return variants
}

func buildFixed32Variants(value uint32) []ValueVariant {
	return []ValueVariant{
		{
			CandidateType: "uint32",
			DisplayValue:  strconv.FormatUint(uint64(value), 10),
			Description:   "Unsigned 32-bit interpretation.",
			Confidence:    "candidate",
		},
		{
			CandidateType: "int32",
			DisplayValue:  strconv.FormatInt(int64(int32(value)), 10),
			Description:   "Signed 32-bit interpretation.",
			Confidence:    "candidate",
		},
		{
			CandidateType: "float32",
			DisplayValue:  formatFloat(float64(math.Float32frombits(value)), 32),
			Description:   "IEEE-754 float32 interpretation.",
			Confidence:    "candidate",
		},
	}
}

func buildFixed64Variants(value uint64) []ValueVariant {
	return []ValueVariant{
		{
			CandidateType: "uint64",
			DisplayValue:  strconv.FormatUint(value, 10),
			Description:   "Unsigned 64-bit interpretation.",
			Confidence:    "candidate",
		},
		{
			CandidateType: "int64",
			DisplayValue:  strconv.FormatInt(int64(value), 10),
			Description:   "Signed 64-bit interpretation.",
			Confidence:    "candidate",
		},
		{
			CandidateType: "double",
			DisplayValue:  formatFloat(math.Float64frombits(value), 64),
			Description:   "IEEE-754 float64 interpretation.",
			Confidence:    "candidate",
		},
	}
}

func buildLengthDelimitedVariants(payload []byte) []ValueVariant {
	variants := []ValueVariant{bytesHexVariant(payload)}
	if utf8.Valid(payload) {
		variants = append([]ValueVariant{stringVariant(payload)}, variants...)
	}
	return variants
}

func bytesHexVariant(payload []byte) ValueVariant {
	return ValueVariant{
		CandidateType: "bytes.hex",
		DisplayValue:  hex.EncodeToString(payload),
		Description:   "Raw bytes rendered as hex.",
		Confidence:    "candidate",
	}
}

func stringVariant(payload []byte) ValueVariant {
	return ValueVariant{
		CandidateType: "string.utf8",
		DisplayValue:  string(payload),
		Description:   "UTF-8 string candidate.",
		Confidence:    "candidate",
	}
}

func decodeZigZag32(value uint32) int32 {
	return int32((value >> 1) ^ uint32(-(int32(value & 1))))
}

func decodeZigZag64(value uint64) int64 {
	return int64((value >> 1) ^ uint64(-(int64(value & 1))))
}

func formatFloat(value float64, bitSize int) string {
	return strconv.FormatFloat(value, 'g', -1, bitSize)
}