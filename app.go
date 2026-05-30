package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	decoderutil "github.com/zhuyanxi/protobuf-decoder-go/internal/decoder"
	inpututil "github.com/zhuyanxi/protobuf-decoder-go/internal/input"
)

const (
	defaultInputEncoding = "auto"
	defaultMaxDepth      = 4
	defaultMaxFields     = 256
	defaultMaxBytes      = 10 * 1024 * 1024
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
	Size      int64  `json:"size"`
	Cancelled bool   `json:"cancelled"`
}

type SaveFileResult struct {
	Path      string `json:"path"`
	Cancelled bool   `json:"cancelled"`
	Format    string `json:"format"`
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

func toDecoderOptions(options DecodeOptions) decoderutil.DecodeOptions {
	return decoderutil.DecodeOptions{
		ParseDelimited: options.ParseDelimited,
		MaxDepth:       options.MaxDepth,
		MaxFields:      options.MaxFields,
		MaxBytes:       options.MaxBytes,
	}
}

func fromDecoderResult(result decoderutil.DecodeResult) DecodeResult {
	parts := make([]Part, 0, len(result.Parts))
	for _, part := range result.Parts {
		parts = append(parts, fromDecoderPart(part))
	}

	return DecodeResult{
		Parts:     parts,
		Leftover:  result.Leftover,
		Error:     result.Error,
		Warnings:  result.Warnings,
		InputSize: result.InputSize,
	}
}

func fromDecoderPart(part decoderutil.Part) Part {
	children := make([]Part, 0, len(part.Children))
	for _, child := range part.Children {
		children = append(children, fromDecoderPart(child))
	}

	values := make([]ValueVariant, 0, len(part.Value))
	for _, value := range part.Value {
		values = append(values, ValueVariant{
			CandidateType: value.CandidateType,
			DisplayValue:  value.DisplayValue,
			Description:   value.Description,
			Confidence:    value.Confidence,
		})
	}

	return Part{
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

	decoded := decoderutil.DecodeBytes(normalizedInput.Bytes, toDecoderOptions(options))
	decoded.Warnings = append([]string{
		fmt.Sprintf("Input encoding: %s", normalizedInput.Encoding),
		fmt.Sprintf("Options: parseDelimited=%t maxDepth=%d maxFields=%d maxBytes=%d", options.ParseDelimited, options.MaxDepth, options.MaxFields, options.MaxBytes),
	}, append(normalizedInput.Warnings, decoded.Warnings...)...)

	return fromDecoderResult(decoded), nil
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
		fmt.Sprintf("Input source: file (%d bytes)", len(fileBytes)),
		fmt.Sprintf("Options: parseDelimited=%t maxDepth=%d maxFields=%d maxBytes=%d", resolvedOptions.ParseDelimited, resolvedOptions.MaxDepth, resolvedOptions.MaxFields, resolvedOptions.MaxBytes),
	}

	decoded := decoderutil.DecodeBytes(fileBytes, toDecoderOptions(resolvedOptions))
	decoded.Warnings = append(warnings, decoded.Warnings...)

	return fromDecoderResult(decoded), nil
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

	info, err := os.Stat(selectedPath)
	if err != nil {
		return OpenFileResult{}, fmt.Errorf("read file %q: %w", selectedPath, err)
	}

	return OpenFileResult{Path: selectedPath, Size: info.Size(), Cancelled: false}, nil
}

func (a *App) CopyResultJSON(result DecodeResult) error {
	if a.ctx == nil {
		return errors.New("application context is not ready")
	}

	payload, err := buildExportPayload(result, "json")
	if err != nil {
		return err
	}

	return runtime.ClipboardSetText(a.ctx, payload)
}

func (a *App) ExportResult(result DecodeResult, format string) (SaveFileResult, error) {
	if a.ctx == nil {
		return SaveFileResult{}, errors.New("application context is not ready")
	}

	resolvedFormat, err := normalizeExportFormat(format)
	if err != nil {
		return SaveFileResult{}, err
	}

	payload, err := buildExportPayload(result, resolvedFormat)
	if err != nil {
		return SaveFileResult{}, err
	}

	selectedPath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           exportDialogTitle(resolvedFormat),
		DefaultFilename: exportFilename(resolvedFormat),
		Filters: []runtime.FileFilter{exportFileFilter(resolvedFormat)},
	})
	if err != nil {
		return SaveFileResult{}, err
	}

	if selectedPath == "" {
		return SaveFileResult{Cancelled: true, Format: resolvedFormat}, nil
	}

	if err := os.WriteFile(selectedPath, []byte(payload), 0o600); err != nil {
		return SaveFileResult{}, err
	}

	return SaveFileResult{Path: selectedPath, Cancelled: false, Format: resolvedFormat}, nil
}

func normalizeExportFormat(value string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(value))
	switch format {
	case "json", "text":
		return format, nil
	default:
		return "", fmt.Errorf("unsupported export format: %s", value)
	}
}

func buildExportPayload(result DecodeResult, format string) (string, error) {
	resolvedFormat, err := normalizeExportFormat(format)
	if err != nil {
		return "", err
	}

	switch resolvedFormat {
	case "json":
		payload, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(payload), nil
	case "text":
		return formatTextResult(result), nil
	default:
		return "", fmt.Errorf("unsupported export format: %s", format)
	}
}

func formatTextResult(result DecodeResult) string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "Input size: %d bytes\n", result.InputSize)
	fmt.Fprintf(&builder, "Parts: %d\n", len(result.Parts))

	if len(result.Warnings) > 0 {
		builder.WriteString("Warnings:\n")
		for _, warning := range result.Warnings {
			fmt.Fprintf(&builder, "- %s\n", warning)
		}
	}

	if result.Error != "" {
		fmt.Fprintf(&builder, "Error: %s\n", result.Error)
	}

	if result.Leftover != "" {
		fmt.Fprintf(&builder, "Leftover: %s\n", result.Leftover)
	}

	if len(result.Parts) > 0 {
		builder.WriteString("Fields:\n")
		appendTextParts(&builder, result.Parts, 0)
	}

	return strings.TrimRight(builder.String(), "\n")
}

func appendTextParts(builder *strings.Builder, parts []Part, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, part := range parts {
		fmt.Fprintf(builder, "%s- #%d %s wire=%d range=[%d, %d) raw=%s\n", indent, part.FieldNumber, part.TypeName, part.WireType, part.ByteRange[0], part.ByteRange[1], part.RawHex)
		for _, variant := range part.Value {
			fmt.Fprintf(builder, "%s  * %s: %s", indent, variant.CandidateType, variant.DisplayValue)
			if variant.Confidence != "" {
				fmt.Fprintf(builder, " [%s]", variant.Confidence)
			}
			if variant.Description != "" {
				fmt.Fprintf(builder, " - %s", variant.Description)
			}
			builder.WriteString("\n")
		}

		if len(part.Children) > 0 {
			appendTextParts(builder, part.Children, depth+1)
		}
	}
}

func exportDialogTitle(format string) string {
	if format == "text" {
		return "Export Protobuf Decode Result (Text)"
	}

	return "Export Protobuf Decode Result (JSON)"
}

func exportFilename(format string) string {
	if format == "text" {
		return "protobuf-decode-result.txt"
	}

	return "protobuf-decode-result.json"
}

func exportFileFilter(format string) runtime.FileFilter {
	if format == "text" {
		return runtime.FileFilter{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"}
	}

	return runtime.FileFilter{DisplayName: "JSON Files (*.json)", Pattern: "*.json"}
}
