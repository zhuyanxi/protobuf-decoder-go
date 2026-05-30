package decoder

import (
	"math"
	"strings"
	"testing"
)

func TestBuildVarintVariants(t *testing.T) {
	variants := buildVarintVariants(150)
	if len(variants) != 6 {
		t.Fatalf("expected 6 varint variants, got %d", len(variants))
	}

	assertVariant(t, variants[0], "uint64", "150")
	assertVariant(t, variants[1], "int32", "150")
	assertVariant(t, variants[2], "int64", "150")
	assertVariant(t, variants[3], "sint32", "75")
	assertVariant(t, variants[4], "sint64", "75")
	if variants[5].CandidateType != "bool.enum_hint" || !strings.Contains(variants[5].DisplayValue, "enum") {
		t.Fatalf("unexpected bool/enum hint %#v", variants[5])
	}
}

func TestBuildFixed32Variants(t *testing.T) {
	variants := buildFixed32Variants(math.Float32bits(3.5))
	if len(variants) != 3 {
		t.Fatalf("expected 3 fixed32 variants, got %d", len(variants))
	}

	assertVariant(t, variants[0], "uint32", "1080033280")
	assertVariant(t, variants[1], "int32", "1080033280")
	assertVariant(t, variants[2], "float32", "3.5")
}

func TestBuildFixed64VariantsFormatsNaNAndInfAsStrings(t *testing.T) {
	infVariants := buildFixed64Variants(math.Float64bits(math.Inf(1)))
	assertVariant(t, infVariants[2], "double", "+Inf")

	nanVariants := buildFixed64Variants(math.Float64bits(math.NaN()))
	assertVariant(t, nanVariants[2], "double", "NaN")
}

func TestBuildLengthDelimitedVariants(t *testing.T) {
	utf8Variants := buildLengthDelimitedVariants([]byte("foo"))
	if len(utf8Variants) != 2 {
		t.Fatalf("expected string and bytes variants, got %#v", utf8Variants)
	}
	assertVariant(t, utf8Variants[0], "string.utf8", "foo")
	assertVariant(t, utf8Variants[1], "bytes.hex", "666f6f")

	binaryVariants := buildLengthDelimitedVariants([]byte{0xff, 0x00})
	if len(binaryVariants) != 1 {
		t.Fatalf("expected bytes-only variant, got %#v", binaryVariants)
	}
	assertVariant(t, binaryVariants[0], "bytes.hex", "ff00")
}

func assertVariant(t *testing.T, variant ValueVariant, candidateType string, displayValue string) {
	t.Helper()
	if variant.CandidateType != candidateType || variant.DisplayValue != displayValue {
		t.Fatalf("expected variant %s=%s, got %#v", candidateType, displayValue, variant)
	}
	if variant.Confidence == "" {
		t.Fatalf("expected confidence on variant %#v", variant)
	}
}