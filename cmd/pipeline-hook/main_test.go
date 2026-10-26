package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		name               string
		payload            string
		want               int
		wantStderrContains string
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
			name:    "payload de edicao sem file_path bloqueia",
			payload: `{"tool_name":"Edit","cwd":"` + root + `","tool_input":{}}`,
			want:    exitBlock,
		},
		{
			name:    "payload de edicao com file_path vazio bloqueia",
			payload: `{"tool_name":"Edit","cwd":"` + root + `","tool_input":{"file_path":"   "}}`,
			want:    exitBlock,
		},
		{
			name:               "JSON inválido não trava a sessão",
			payload:            `{"tool_name":`,
			want:               exitBlock,
			wantStderrContains: "doctor",
		},
		{
			name:               "stdin vazio bloqueia fail-closed sugerindo doctor",
			payload:            "",
			want:               exitBlock,
			wantStderrContains: "doctor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := run(strings.NewReader(tt.payload), &stdout, &stderr)
			if got != tt.want {
				t.Errorf("run = %d, quer %d (stderr: %s)", got, tt.want, stderr.String())
			}
			if tt.wantStderrContains != "" && !strings.Contains(stderr.String(), tt.wantStderrContains) {
				t.Errorf("stderr não contém %q (stderr: %s)", tt.wantStderrContains, stderr.String())
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

	var stdout, stderr bytes.Buffer
	if got := run(strings.NewReader(payload), &stdout, &stderr); got != exitAllow {
		t.Errorf("run = %d, quer %d: campo na raiz não é o formato do PreToolUse", got, exitAllow)
	}
}

func TestConfigQuebradaBloqueia(t *testing.T) {
	root := setupProject(t, `{"version":1,`)
	payload := `{"tool_name":"Edit","cwd":"` + root + `","tool_input":{"file_path":"` + filepath.Join(root, "internal", "x.go") + `"}}`

	var stdout, stderr bytes.Buffer
	if got := run(strings.NewReader(payload), &stdout, &stderr); got != exitBlock {
		t.Errorf("run = %d, quer %d: config quebrada deve bloquear", got, exitBlock)
	}
	if !strings.Contains(stderr.String(), config.FileName) {
		t.Errorf("stderr não menciona o arquivo de config: %s", stderr.String())
	}
}

func TestMensagemDeBloqueioOrienta(t *testing.T) {
	root := setupProject(t, cfgBody)
	payload := `{"tool_name":"Edit","cwd":"` + root + `","tool_input":{"file_path":"` + filepath.Join(root, "internal", "x.go") + `"}}`

	var stdout, stderr bytes.Buffer
	run(strings.NewReader(payload), &stdout, &stderr)
	msg := stderr.String()

	for _, want := range []string{"BLOQUEADO", "/triage", "dev-implementation"} {
		if !strings.Contains(msg, want) {
			t.Errorf("mensagem não contém %q:\n%s", want, msg)
		}
	}
}

func TestAntigravityProtocol(t *testing.T) {
	root := setupProject(t, cfgBody)
	protected := filepath.Join(root, "internal", "auth", "token.go")
	livre := filepath.Join(root, "docs-ai", "nota.md")

	tests := []struct {
		name         string
		payload      string
		wantExit     int
		wantDecision string
		containsText string
	}{
		{
			name: "write_to_file em caminho protegido bloqueia",
			payload: `{
				"toolCall": {"name": "write_to_file", "args": {"TargetFile": "` + protected + `"}},
				"workspacePaths": ["` + root + `"]
			}`,
			wantExit:     exitAllow,
			wantDecision: "deny",
			containsText: "BLOQUEADO pelo pipeline-ai",
		},
		{
			name: "replace_file_content em caminho protegido bloqueia",
			payload: `{
				"toolCall": {"name": "replace_file_content", "args": {"TargetFile": "` + protected + `"}},
				"workspacePaths": ["` + root + `"]
			}`,
			wantExit:     exitAllow,
			wantDecision: "deny",
			containsText: "dev-implementation",
		},
		{
			name: "multi_replace_file_content em caminho protegido bloqueia",
			payload: `{
				"toolCall": {"name": "multi_replace_file_content", "args": {"TargetFile": "` + protected + `"}},
				"workspacePaths": ["` + root + `"]
			}`,
			wantExit:     exitAllow,
			wantDecision: "deny",
			containsText: "BLOQUEADO",
		},
		{
			name: "agent autorizado em subagentId passa",
			payload: `{
				"toolCall": {"name": "replace_file_content", "args": {"TargetFile": "` + protected + `"}},
				"workspacePaths": ["` + root + `"],
				"subagentId": "dev-implementation"
			}`,
			wantExit:     exitAllow,
			wantDecision: "allow",
		},
		{
			name: "write_to_file em caminho não protegido passa",
			payload: `{
				"toolCall": {"name": "write_to_file", "args": {"TargetFile": "` + livre + `"}},
				"workspacePaths": ["` + root + `"]
			}`,
			wantExit:     exitAllow,
			wantDecision: "allow",
		},
		{
			name: "ferramenta de leitura passa",
			payload: `{
				"toolCall": {"name": "run_command", "args": {"CommandLine": "go test ./..."}},
				"workspacePaths": ["` + root + `"]
			}`,
			wantExit:     exitAllow,
			wantDecision: "allow",
		},
		{
			name: "NotebookEdit em caminho protegido com notebook_path e agente não autorizado bloqueia",
			payload: `{
				"toolCall": {"name": "NotebookEdit", "args": {"notebook_path": "` + filepath.Join(root, "internal", "foo.go") + `"}},
				"workspacePaths": ["` + root + `"],
				"subagentId": "pm-triage"
			}`,
			wantExit:     exitAllow,
			wantDecision: "deny",
			containsText: "BLOQUEADO",
		},
		{
			name: "NotebookEdit em caminho protegido com notebook_path na sessao principal bloqueia",
			payload: `{
				"toolCall": {"name": "NotebookEdit", "args": {"notebook_path": "` + filepath.Join(root, "internal", "foo.go") + `"}},
				"workspacePaths": ["` + root + `"]
			}`,
			wantExit:     exitAllow,
			wantDecision: "deny",
			containsText: "BLOQUEADO",
		},
		{
			name: "write_to_file com caminho ausente bloqueia com deny",
			payload: `{
				"toolCall": {"name": "write_to_file", "args": {}},
				"workspacePaths": ["` + root + `"]
			}`,
			wantExit:     exitAllow,
			wantDecision: "deny",
		},
		{
			name: "write_to_file com caminho vazio bloqueia com deny",
			payload: `{
				"toolCall": {"name": "write_to_file", "args": {"TargetFile": "   "}},
				"workspacePaths": ["` + root + `"]
			}`,
			wantExit:     exitAllow,
			wantDecision: "deny",
		},
		{
			name: "Edit com caminho ausente bloqueia com deny",
			payload: `{
				"toolCall": {"name": "Edit", "args": {}},
				"workspacePaths": ["` + root + `"]
			}`,
			wantExit:     exitAllow,
			wantDecision: "deny",
		},
		{
			name:         "payload Antigravity quebrado contendo toolCall responde com deny",
			payload:      `{"toolCall": {"name": "write_to_file", "args": {`,
			wantExit:     exitAllow,
			wantDecision: "deny",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := run(strings.NewReader(tt.payload), &stdout, &stderr)
			if got != tt.wantExit {
				t.Errorf("exit = %d, quer %d", got, tt.wantExit)
			}
			outStr := stdout.String()
			if !strings.Contains(outStr, `"decision":"`+tt.wantDecision+`"`) {
				t.Errorf("stdout não tem decision %q: %s", tt.wantDecision, outStr)
			}
			if tt.containsText != "" && !strings.Contains(outStr, tt.containsText) {
				t.Errorf("stdout não contém %q: %s", tt.containsText, outStr)
			}
		})
	}
}

func TestAntigravityConfigQuebrada(t *testing.T) {
	root := setupProject(t, `{"version":1,`)
	payload := `{
		"toolCall": {"name": "write_to_file", "args": {"TargetFile": "` + filepath.Join(root, "internal", "x.go") + `"}},
		"workspacePaths": ["` + root + `"]
	}`

	var stdout, stderr bytes.Buffer
	got := run(strings.NewReader(payload), &stdout, &stderr)
	if got != exitAllow {
		t.Errorf("exit = %d, quer %d", got, exitAllow)
	}
	outStr := stdout.String()
	if !strings.Contains(outStr, `"decision":"deny"`) {
		t.Errorf("stdout não tem decision deny: %s", outStr)
	}
	if !strings.Contains(outStr, config.FileName) {
		t.Errorf("stdout não menciona o config: %s", outStr)
	}
}

type errReader struct{}

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error simulado")
}

func TestStdinIlegivelBloqueia(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := run(errReader{}, &stdout, &stderr)
	if got != exitBlock {
		t.Errorf("run = %d, quer %d para stdin com erro de leitura", got, exitBlock)
	}
	if !strings.Contains(stderr.String(), "doctor") {
		t.Errorf("stderr não sugere doctor: %s", stderr.String())
	}
}

func setActiveDelivery(t *testing.T, root, delivID, taskCode string, expiresAt time.Time) {
	t.Helper()
	dir := filepath.Join(root, ".pipeline-state")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir .pipeline-state: %v", err)
	}
	body := fmt.Sprintf(`{
  "delivery_id": %q,
  "task_code": %q,
  "opened_at": %q,
  "expires_at": %q
}`, delivID, taskCode, time.Now().UTC().Format(time.RFC3339), expiresAt.UTC().Format(time.RFC3339))
	if err := os.WriteFile(filepath.Join(dir, "active-delivery.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write active-delivery.json: %v", err)
	}
}

func TestHybridGateActiveDelivery(t *testing.T) {
	root := setupProject(t, cfgBody)
	protected := filepath.Join(root, "internal", "auth", "token.go")

	tests := []struct {
		name         string
		setupState   func(t *testing.T, root string)
		payload      string
		wantExit     int
		wantDecision string
		containsText string
	}{
		{
			name: "sessao principal liberada com delivery ativa valida no protocolo CLI",
			setupState: func(t *testing.T, root string) {
				setActiveDelivery(t, root, "DEL-010", "TASK-010", time.Now().Add(1*time.Hour))
			},
			payload:  `{"tool_name":"Edit","cwd":"` + root + `","tool_input":{"file_path":"` + protected + `"}}`,
			wantExit: exitAllow,
		},
		{
			name: "sessao principal liberada com delivery ativa valida no protocolo Antigravity",
			setupState: func(t *testing.T, root string) {
				setActiveDelivery(t, root, "DEL-010", "TASK-010", time.Now().Add(1*time.Hour))
			},
			payload: `{
				"toolCall": {"name": "write_to_file", "args": {"TargetFile": "` + protected + `"}},
				"workspacePaths": ["` + root + `"]
			}`,
			wantExit:     exitAllow,
			wantDecision: "allow",
		},
		{
			name: "sessao principal bloqueada quando delivery ativa esta expirada no protocolo CLI",
			setupState: func(t *testing.T, root string) {
				setActiveDelivery(t, root, "DEL-010", "TASK-010", time.Now().Add(-15*time.Minute))
			},
			payload:      `{"tool_name":"Edit","cwd":"` + root + `","tool_input":{"file_path":"` + protected + `"}}`,
			wantExit:     exitBlock,
			containsText: "expir",
		},
		{
			name: "sessao principal bloqueada quando delivery ativa esta expirada no protocolo Antigravity",
			setupState: func(t *testing.T, root string) {
				setActiveDelivery(t, root, "DEL-010", "TASK-010", time.Now().Add(-15*time.Minute))
			},
			payload: `{
				"toolCall": {"name": "write_to_file", "args": {"TargetFile": "` + protected + `"}},
				"workspacePaths": ["` + root + `"]
			}`,
			wantExit:     exitAllow,
			wantDecision: "deny",
			containsText: "expir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subRoot := setupProject(t, cfgBody)
			subProtected := filepath.Join(subRoot, "internal", "auth", "token.go")
			payload := strings.ReplaceAll(tt.payload, protected, subProtected)
			payload = strings.ReplaceAll(payload, root, subRoot)

			if tt.setupState != nil {
				tt.setupState(t, subRoot)
			}

			var stdout, stderr bytes.Buffer
			got := run(strings.NewReader(payload), &stdout, &stderr)
			if got != tt.wantExit {
				t.Errorf("exit = %d, quer %d (stderr: %s, stdout: %s)", got, tt.wantExit, stderr.String(), stdout.String())
			}

			if tt.wantDecision != "" {
				outStr := stdout.String()
				if !strings.Contains(outStr, `"decision":"`+tt.wantDecision+`"`) {
					t.Errorf("stdout não contém decision %q: %s", tt.wantDecision, outStr)
				}
			}

			if tt.containsText != "" {
				combined := stdout.String() + "\n" + stderr.String()
				if !strings.Contains(strings.ToLower(combined), strings.ToLower(tt.containsText)) {
					t.Errorf("saída não contém %q (stdout: %s, stderr: %s)", tt.containsText, stdout.String(), stderr.String())
				}
			}
		})
	}
}

func TestImmutablePathsHookBlocking(t *testing.T) {
	const cfgWithImmutable = `{
  "version": 1,
  "project_type": "desktop",
  "protected_paths": ["internal/", "cmd/"],
  "immutable_paths": [
    ".pipeline-ai.json",
    "AGENTS.md",
    "internal/core/"
  ],
  "editor_agents": ["dev-implementation", "test-writer"]
}`

	tests := []struct {
		name         string
		filePath     string
		agentType    string
		withDelivery bool
		wantExit     int
		wantDecision string
		containsText string
	}{
		{
			name:         "dev-implementation tentando editar .pipeline-ai.json bloqueia sempre",
			filePath:     ".pipeline-ai.json",
			agentType:    "dev-implementation",
			withDelivery: false,
			wantExit:     exitAllow,
			wantDecision: "deny",
			containsText: "imutável",
		},
		{
			name:         "sessao principal com delivery ativa tentando editar AGENTS.md bloqueia sempre",
			filePath:     "AGENTS.md",
			agentType:    "",
			withDelivery: true,
			wantExit:     exitAllow,
			wantDecision: "deny",
			containsText: "imutável",
		},
		{
			name:         "dev-implementation com delivery ativa tentando editar arquivo em internal/core/ bloqueia sempre",
			filePath:     filepath.Join("internal", "core", "guard.go"),
			agentType:    "dev-implementation",
			withDelivery: true,
			wantExit:     exitAllow,
			wantDecision: "deny",
			containsText: "imutável",
		},
		{
			name:         "dev-implementation tentando editar caminho protegido normal (nao imutavel) passa",
			filePath:     filepath.Join("internal", "auth", "token.go"),
			agentType:    "dev-implementation",
			withDelivery: false,
			wantExit:     exitAllow,
			wantDecision: "allow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := setupProject(t, cfgWithImmutable)
			target := filepath.Join(root, tt.filePath)

			if tt.withDelivery {
				setActiveDelivery(t, root, "DEL-010", "TASK-010", time.Now().Add(1*time.Hour))
			}

			payload := fmt.Sprintf(`{
				"toolCall": {"name": "write_to_file", "args": {"TargetFile": %q}},
				"workspacePaths": [%q],
				"subagentId": %q
			}`, target, root, tt.agentType)

			var stdout, stderr bytes.Buffer
			got := run(strings.NewReader(payload), &stdout, &stderr)
			if got != tt.wantExit {
				t.Errorf("exit = %d, quer %d (stderr: %s)", got, tt.wantExit, stderr.String())
			}

			outStr := stdout.String()
			if tt.wantDecision != "" && !strings.Contains(outStr, `"decision":"`+tt.wantDecision+`"`) {
				t.Errorf("stdout não contém decision %q: %s", tt.wantDecision, outStr)
			}
			if tt.containsText != "" {
				combined := strings.ToLower(stdout.String() + "\n" + stderr.String())
				if !strings.Contains(combined, strings.ToLower(tt.containsText)) && !strings.Contains(combined, "immutable") {
					t.Errorf("saída não menciona imutabilidade: stdout=%s stderr=%s", stdout.String(), stderr.String())
				}
			}
		})
	}
}
