package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
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
}

type DecodePart struct {
	Index    int      `json:"index"`
	Label    string   `json:"label"`
	TypeName string   `json:"typeName"`
	Raw      string   `json:"raw"`
	Notes    []string `json:"notes"`
}

type DecodeResult struct {
	Summary         string       `json:"summary"`
	NormalizedInput string       `json:"normalizedInput"`
	InputEncoding   string       `json:"inputEncoding"`
	ParseDelimited  bool         `json:"parseDelimited"`
	Parts           []DecodePart `json:"parts"`
}

type OpenFileResult struct {
	Path      string `json:"path"`
	Cancelled bool   `json:"cancelled"`
}

func (a *App) Decode(req DecodeRequest) (DecodeResult, error) {
	normalizedInput := strings.TrimSpace(req.Input)
	if normalizedInput == "" {
		return DecodeResult{}, errors.New("input is required")
	}

	encoding := strings.TrimSpace(req.InputEncoding)
	if encoding == "" {
		encoding = "auto"
	}

	part := DecodePart{
		Index:    1,
		Label:    "Mock field",
		TypeName: "MockDecodePart",
		Raw:      normalizedInput,
		Notes: []string{
			"Story 1 mock result from Go backend",
			fmt.Sprintf("Input length: %d characters", len(normalizedInput)),
			fmt.Sprintf("Selected encoding: %s", encoding),
		},
	}

	return DecodeResult{
		Summary:         "Wails binding is active. Mock decode result returned from Go backend.",
		NormalizedInput: normalizedInput,
		InputEncoding:   encoding,
		ParseDelimited:  req.ParseDelimited,
		Parts:           []DecodePart{part},
	}, nil
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
