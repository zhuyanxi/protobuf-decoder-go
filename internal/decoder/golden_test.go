package decoder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	inpututil "github.com/zhuyanxi/protobuf-decoder-go/internal/input"
)

const updateGoldenEnv = "UPDATE_GOLDEN"

type goldenSnapshot struct {
	Normalized normalizedSnapshot `json:"normalized"`
	Result     goldenResult       `json:"result"`
}

type normalizedSnapshot struct {
	Encoding string   `json:"encoding"`
	Warnings []string `json:"warnings,omitempty"`
}

type goldenResult struct {
	Parts     []goldenPart `json:"parts"`
	Leftover  string       `json:"leftover"`
	Error     string       `json:"error"`
	Warnings  []string     `json:"warnings,omitempty"`
	InputSize int          `json:"inputSize"`
}

type goldenPart struct {
	ByteRange   [2]int           `json:"byteRange"`
	Index       int              `json:"index"`
	FieldNumber int              `json:"fieldNumber"`
	WireType    int              `json:"wireType"`
	TypeName    string           `json:"typeName"`
	RawHex      string           `json:"rawHex"`
	Value       []goldenVariant  `json:"value"`
	Children    []goldenPart     `json:"children,omitempty"`
}

type goldenVariant struct {
	CandidateType string `json:"candidateType"`
	DisplayValue  string `json:"displayValue"`
	Description   string `json:"description,omitempty"`
	Confidence    string `json:"confidence,omitempty"`
}

func TestGoldenDecodeScenarios(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		encoding string
		options  DecodeOptions
	}{
		{
			name:     "frontend_sample_auto",
			input:    "0a03666f6f",
			encoding: "auto",
			options:  DecodeOptions{MaxDepth: 4, MaxFields: 256, MaxBytes: 1024},
		},
		{
			name:     "wire_types_hex",
			input:    "0896011108070605040302011a03666f6f2578563412",
			encoding: "hex",
			options:  DecodeOptions{MaxDepth: 4, MaxFields: 256, MaxBytes: 1024},
		},
		{
			name:     "nested_message_hex",
			input:    "0a020801",
			encoding: "hex",
			options:  DecodeOptions{MaxDepth: 4, MaxFields: 256, MaxBytes: 1024},
		},
		{
			name:     "base64_string",
			input:    "CgNmb28=",
			encoding: "base64",
			options:  DecodeOptions{MaxDepth: 4, MaxFields: 256, MaxBytes: 1024},
		},
		{
			name:     "grpc_header_hex",
			input:    "00000000020801",
			encoding: "hex",
			options:  DecodeOptions{MaxDepth: 4, MaxFields: 256, MaxBytes: 1024},
		},
		{
			name:     "delimited_stream_hex",
			input:    "020801021002",
			encoding: "hex",
			options:  DecodeOptions{ParseDelimited: true, MaxDepth: 4, MaxFields: 256, MaxBytes: 1024},
		},
		{
			name:     "illegal_unsupported_wire_type_hex",
			input:    "0b00",
			encoding: "hex",
			options:  DecodeOptions{MaxDepth: 4, MaxFields: 256, MaxBytes: 1024},
		},
		{
			name:     "illegal_truncated_varint_hex",
			input:    "80",
			encoding: "hex",
			options:  DecodeOptions{MaxDepth: 4, MaxFields: 256, MaxBytes: 1024},
		},
		{
			name:     "illegal_truncated_length_hex",
			input:    "08011a03666f",
			encoding: "hex",
			options:  DecodeOptions{MaxDepth: 4, MaxFields: 256, MaxBytes: 1024},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			normalized, err := inpututil.NormalizeText(testCase.input, testCase.encoding, testCase.options.MaxBytes)
			if err != nil {
				t.Fatalf("normalize input: %v", err)
			}

			snapshot := goldenSnapshot{
				Normalized: normalizedSnapshot{
					Encoding: normalized.Encoding,
					Warnings: normalized.Warnings,
				},
				Result: toGoldenResult(DecodeBytes(normalized.Bytes, testCase.options)),
			}

			actual, err := json.MarshalIndent(snapshot, "", "  ")
			if err != nil {
				t.Fatalf("marshal golden snapshot: %v", err)
			}
			actual = append(actual, '\n')

			goldenPath := filepath.Join("testdata", testCase.name+".golden.json")
			if os.Getenv(updateGoldenEnv) == "1" {
				if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
					t.Fatalf("create golden directory: %v", err)
				}
				if err := os.WriteFile(goldenPath, actual, 0o600); err != nil {
					t.Fatalf("write golden file: %v", err)
				}
			}

			expected, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden file: %v", err)
			}

			if string(expected) != string(actual) {
				t.Fatalf("golden mismatch for %s\nexpected:\n%s\nactual:\n%s", testCase.name, string(expected), string(actual))
			}
		})
	}
}

func toGoldenResult(result DecodeResult) goldenResult {
	parts := make([]goldenPart, 0, len(result.Parts))
	for _, part := range result.Parts {
		parts = append(parts, toGoldenPart(part))
	}

	return goldenResult{
		Parts:     parts,
		Leftover:  result.Leftover,
		Error:     result.Error,
		Warnings:  result.Warnings,
		InputSize: result.InputSize,
	}
}

func toGoldenPart(part Part) goldenPart {
	children := make([]goldenPart, 0, len(part.Children))
	for _, child := range part.Children {
		children = append(children, toGoldenPart(child))
	}

	values := make([]goldenVariant, 0, len(part.Value))
	for _, value := range part.Value {
		values = append(values, goldenVariant{
			CandidateType: value.CandidateType,
			DisplayValue:  value.DisplayValue,
			Description:   value.Description,
			Confidence:    value.Confidence,
		})
	}

	return goldenPart{
		ByteRange:   part.ByteRange,
		Index:       part.Index,
		FieldNumber: part.FieldNumber,
		WireType:    part.WireType,
		TypeName:    part.TypeName,
		RawHex:      part.RawHex,
		Value:       values,
		Children:    children,
	}
}
