package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config"
)

const cfgBody = `{
  "version": 1,
  "project_type": "desktop",
  "protected_paths": ["internal/", "frontend/src/"],
  "editor_agents": ["dev-implementation", "test-writer"]
}`

func setupProject(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return dir
}

func TestRun(t *testing.T) {
	root := setupProject(t, cfgBody)
	protected := filepath.Join(root, "internal", "auth", "token.go")
	livre := filepath.Join(root, "docs-ai", "nota.md")

	tests := []struct {
		name    string
		payload string
		want    int
	}{
		{
			name:    "sessão principal em caminho protegido bloqueia",
			payload: `{"tool_name":"Edit","cwd":"` + root + `","tool_input":{"file_path":"` + protected + `"}}`,
			want:    exitBlock,
		},
		{
			name:    "agent autorizado passa",
			payload: `{"tool_name":"Edit","agent_type":"dev-implementation","cwd":"` + root + `","tool_input":{"file_path":"` + protected + `"}}`,
			want:    exitAllow,
		},
		{
			name:    "agent não autorizado bloqueia",
			payload: `{"tool_name":"Write","agent_type":"pm-triage","cwd":"` + root + `","tool_input":{"file_path":"` + protected + `"}}`,
			want:    exitBlock,
		},
		{
			name:    "caminho não protegido passa",
			payload: `{"tool_name":"Edit","cwd":"` + root + `","tool_input":{"file_path":"` + livre + `"}}`,
			want:    exitAllow,
		},
		{
			name:    "ferramenta de leitura passa",
			payload: `{"tool_name":"Read","cwd":"` + root + `","tool_input":{"file_path":"` + protected + `"}}`,
			want:    exitAllow,
		},
		{
			name:    "MultiEdit é coberto",
			payload: `{"tool_name":"MultiEdit","cwd":"` + root + `","tool_input":{"file_path":"` + protected + `"}}`,
			want:    exitBlock,
		},
		{
			name:    "payload sem file_path passa",
			payload: `{"tool_name":"Edit","cwd":"` + root + `","tool_input":{}}`,
			want:    exitAllow,
		},
		{
			name:    "JSON inválido não trava a sessão",
			payload: `{"tool_name":`,
			want:    exitAllow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			got := run(strings.NewReader(tt.payload), &stderr)
			if got != tt.want {
				t.Errorf("run = %d, quer %d (stderr: %s)", got, tt.want, stderr.String())
			}
		})
	}
}

// TestRegressaoCampoNaRaiz fixa o bug que existia na versão em bash: o caminho
// vem em tool_input.file_path. Um payload com file_path na raiz não deve
// bloquear nada, porque não é o formato real do PreToolUse.
func TestRegressaoCampoNaRaiz(t *testing.T) {
	root := setupProject(t, cfgBody)
	protected := filepath.Join(root, "internal", "auth", "token.go")

	payload := `{"tool_name":"Edit","cwd":"` + root + `","file_path":"` + protected + `"}`

	var stderr bytes.Buffer
	if got := run(strings.NewReader(payload), &stderr); got != exitAllow {
		t.Errorf("run = %d, quer %d: campo na raiz não é o formato do PreToolUse", got, exitAllow)
	}
}

func TestConfigQuebradaBloqueia(t *testing.T) {
	root := setupProject(t, `{"version":1,`)
	payload := `{"tool_name":"Edit","cwd":"` + root + `","tool_input":{"file_path":"` + filepath.Join(root, "internal", "x.go") + `"}}`

	var stderr bytes.Buffer
	if got := run(strings.NewReader(payload), &stderr); got != exitBlock {
		t.Errorf("run = %d, quer %d: config quebrada deve bloquear", got, exitBlock)
	}
	if !strings.Contains(stderr.String(), config.FileName) {
		t.Errorf("stderr não menciona o arquivo de config: %s", stderr.String())
	}
}

func TestMensagemDeBloqueioOrienta(t *testing.T) {
	root := setupProject(t, cfgBody)
	payload := `{"tool_name":"Edit","cwd":"` + root + `","tool_input":{"file_path":"` + filepath.Join(root, "internal", "x.go") + `"}}`

	var stderr bytes.Buffer
	run(strings.NewReader(payload), &stderr)
	msg := stderr.String()

	for _, want := range []string{"BLOQUEADO", "/triage", "dev-implementation"} {
		if !strings.Contains(msg, want) {
			t.Errorf("mensagem não contém %q:\n%s", want, msg)
		}
	}
}
