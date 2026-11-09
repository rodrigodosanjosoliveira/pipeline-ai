package semantic_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic"
)

func TestResolvePreset(t *testing.T) {
	tests := []struct {
		name         string
		preset       string
		wantMode     string
		wantContains []string
		wantErr      error
	}{
		{
			name:         "preset yagni",
			preset:       "yagni",
			wantMode:     "noul",
			wantContains: []string{"YAGNI", "overengineering"},
			wantErr:      nil,
		},
		{
			name:         "alias minimalism",
			preset:       "minimalism",
			wantMode:     "noul",
			wantContains: []string{"YAGNI", "overengineering"},
			wantErr:      nil,
		},
		{
			name:         "alias ponytail",
			preset:       "ponytail",
			wantMode:     "noul",
			wantContains: []string{"YAGNI", "overengineering"},
			wantErr:      nil,
		},
		{
			name:         "preset case-insensitive e com espacos",
			preset:       "  YAGNI  ",
			wantMode:     "noul",
			wantContains: []string{"YAGNI"},
			wantErr:      nil,
		},
		{
			name:     "preset vazio retorna ErrInvalidPreset",
			preset:   "",
			wantMode: "",
			wantErr:  semantic.ErrInvalidPreset,
		},
		{
			name:     "preset desconhecido retorna ErrInvalidPreset",
			preset:   "inexistente",
			wantMode: "",
			wantErr:  semantic.ErrInvalidPreset,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, question, err := semantic.ResolvePreset(tt.preset)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ResolvePreset(%q) err = %v, esperado %v", tt.preset, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolvePreset(%q) retornou erro inesperado: %v", tt.preset, err)
			}
			if mode != tt.wantMode {
				t.Errorf("ResolvePreset(%q) mode = %q, esperado %q", tt.preset, mode, tt.wantMode)
			}
			if strings.TrimSpace(question) == "" {
				t.Errorf("ResolvePreset(%q) retornou question vazia", tt.preset)
			}
			for _, sub := range tt.wantContains {
				if !strings.Contains(strings.ToLower(question), strings.ToLower(sub)) {
					t.Errorf("ResolvePreset(%q) question = %q, não contém %q", tt.preset, question, sub)
				}
			}
		})
	}
}
