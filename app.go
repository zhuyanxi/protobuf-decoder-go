package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	inpututil "github.com/zhuyanxi/protobuf-decoder-go/internal/input"
)

const (
	defaultInputEncoding = "auto"
	defaultMaxDepth      = 4
	defaultMaxFields     = 256
	defaultMaxBytes      = 1024 * 1024
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

type DecodeRequest struct {
	Input          string `json:"input"`
	InputEncoding  string `json:"inputEncoding"`
	ParseDelimited bool   `json:"parseDelimited"`
	MaxDepth       int    `json:"maxDepth"`
	MaxFields      int    `json:"maxFields"`
	MaxBytes       int    `json:"maxBytes"`
}

type DecodeOptions struct {
	ParseDelimited bool `json:"parseDelimited"`
	MaxDepth       int  `json:"maxDepth"`
	MaxFields      int  `json:"maxFields"`
	MaxBytes       int  `json:"maxBytes"`
}

type DecodeResult struct {
	Parts     []Part   `json:"parts"`
	Leftover  string   `json:"leftover"`
	Error     string   `json:"error,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
	InputSize int      `json:"inputSize"`
}

type Part struct {
	ByteRange   [2]int         `json:"byteRange"`
	Index       int            `json:"index"`
	FieldNumber int            `json:"fieldNumber"`
	WireType    int            `json:"wireType"`
	TypeName    string         `json:"typeName"`
	RawHex      string         `json:"rawHex"`
	Value       []ValueVariant `json:"value"`
	Children    []Part        `json:"children,omitempty"`
}

type ValueVariant struct {
	CandidateType string `json:"candidateType"`
	DisplayValue  string `json:"displayValue"`
	Description   string `json:"description,omitempty"`
	Confidence    string `json:"confidence,omitempty"`
}

type OpenFileResult struct {
	Path      string `json:"path"`
	Cancelled bool   `json:"cancelled"`
}

func (r DecodeRequest) Options() DecodeOptions {
	options := DecodeOptions{
		ParseDelimited: r.ParseDelimited,
		MaxDepth:       r.MaxDepth,
		MaxFields:      r.MaxFields,
		MaxBytes:       r.MaxBytes,
	}

	if options.MaxDepth <= 0 {
		options.MaxDepth = defaultMaxDepth
	}

	if options.MaxFields <= 0 {
		options.MaxFields = defaultMaxFields
	}

	if options.MaxBytes <= 0 {
		options.MaxBytes = defaultMaxBytes
	}

	return options
}

func buildMockDecodeResult(data []byte, warnings []string) DecodeResult {
	rawHex := hex.EncodeToString(data)

	part := Part{
		ByteRange:   [2]int{0, len(data)},
		Index:       1,
		FieldNumber: 1,
		WireType:    2,
		TypeName:    "MockLengthDelimited",
		RawHex:      rawHex,
		Value: []ValueVariant{
			{
				CandidateType: "bytes.hex",
				DisplayValue:  rawHex,
				Description:   "Normalized input bytes represented as hex.",
				Confidence:    "mock",
			},
			{
				CandidateType: "int64",
				DisplayValue:  strconv.FormatInt(int64(len(data)), 10),
				Description:   "Mock 64-bit candidate encoded as string to avoid frontend precision loss.",
				Confidence:    "low",
			},
		},
	}

	return DecodeResult{
		Parts:     []Part{part},
		Warnings:  warnings,
		InputSize: len(data),
	}
}

func normalizeInputEncoding(value string) (string, error) {
	encoding := strings.TrimSpace(value)
	if encoding == "" {
		return defaultInputEncoding, nil
	}

	switch encoding {
	case "auto", "hex", "base64":
		return encoding, nil
	default:
		return "", fmt.Errorf("unsupported input encoding: %s", encoding)
	}
}

func (a *App) Decode(req DecodeRequest) (DecodeResult, error) {
	if strings.TrimSpace(req.Input) == "" {
		return DecodeResult{}, errors.New("input is required")
	}

	encoding, err := normalizeInputEncoding(req.InputEncoding)
	if err != nil {
		return DecodeResult{}, err
	}

	options := req.Options()
	normalizedInput, err := inpututil.NormalizeText(req.Input, encoding, options.MaxBytes)
	if err != nil {
		return DecodeResult{}, err
	}

	warnings := append([]string{
		"Mock decode result. Story 3 validates normalized text input before real parser implementation.",
		fmt.Sprintf("Input encoding: %s", normalizedInput.Encoding),
		fmt.Sprintf("Options: parseDelimited=%t maxDepth=%d maxFields=%d maxBytes=%d", options.ParseDelimited, options.MaxDepth, options.MaxFields, options.MaxBytes),
	}, normalizedInput.Warnings...)

	return buildMockDecodeResult(normalizedInput.Bytes, warnings), nil
}

func (a *App) DecodeFile(path string, options DecodeOptions) (DecodeResult, error) {
	resolvedOptions := options
	if resolvedOptions.MaxDepth <= 0 {
		resolvedOptions.MaxDepth = defaultMaxDepth
	}
	if resolvedOptions.MaxFields <= 0 {
		resolvedOptions.MaxFields = defaultMaxFields
	}
	if resolvedOptions.MaxBytes <= 0 {
		resolvedOptions.MaxBytes = defaultMaxBytes
	}

	fileBytes, err := inpututil.LoadFile(path, resolvedOptions.MaxBytes)
	if err != nil {
		return DecodeResult{}, err
	}

	warnings := []string{
		"Mock decode result. Story 3 validates file input before real parser implementation.",
		fmt.Sprintf("Input source: file (%d bytes)", len(fileBytes)),
		fmt.Sprintf("Options: parseDelimited=%t maxDepth=%d maxFields=%d maxBytes=%d", resolvedOptions.ParseDelimited, resolvedOptions.MaxDepth, resolvedOptions.MaxFields, resolvedOptions.MaxBytes),
	}

	return buildMockDecodeResult(fileBytes, warnings), nil
}

func (a *App) OpenInputFile() (OpenFileResult, error) {
	if a.ctx == nil {
		return OpenFileResult{}, errors.New("application context is not ready")
	}

	selectedPath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open Protobuf Input File",
	})
	if err != nil {
		return OpenFileResult{}, err
	}

	if selectedPath == "" {
		return OpenFileResult{Cancelled: true}, nil
	}

	return OpenFileResult{Path: selectedPath, Cancelled: false}, nil
}
