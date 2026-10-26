package scaffold

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config"
)

func TestScaffoldDesktop(t *testing.T) {
	dir := t.TempDir()

	opts := Options{
		Name:      "test-desktop-app",
		Type:      "desktop",
		Database:  "postgres",
		Module:    "github.com/test/desktop-app",
		TargetDir: dir,
		InitGit:   false,
	}

	if err := Scaffold(opts); err != nil {
		t.Fatalf("Scaffold falhou: %v", err)
	}

	// 1. Validar .pipeline-ai.json via internal/config
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load falhou: %v", err)
	}
	if cfg.ProjectType != "desktop" {
		t.Errorf("esperava project_type desktop, obteve %s", cfg.ProjectType)
	}

	// 2. Validar go.mod
	goModBytes, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatalf("go.mod não encontrado: %v", err)
	}
	goMod := string(goModBytes)
	if !strings.Contains(goMod, "module github.com/test/desktop-app") {
		t.Errorf("go.mod não contém módulo correto: %s", goMod)
	}
	if !strings.Contains(goMod, "pipeline-hook") {
		t.Errorf("go.mod não contém diretiva tool para pipeline-hook: %s", goMod)
	}

	// 3. Validar arquivos obrigatórios
	expectedFiles := []string{
		"main.go",
		"Taskfile.yml",
		".gitignore",
		".githooks/pre-push",
		".agents/hooks.json",
		".agents/rules/invariants.md",
		".agents/rules/handoff-contract.md",
		".agents/skills/pm-triage/SKILL.md",
		".agents/skills/release-ops/SKILL.md",
		".agents/skills/dev-implementation/SKILL.md",
		".agents/skills/data-model/SKILL.md",
		".agents/skills/migration-reviewer/SKILL.md",
		".agents/skills/query-performance/SKILL.md",
		".agents/skills/context-loader/SKILL.md",
		"docs-ai/01-INVARIANTES-GLOBAIS.md",
		"docs-ai/03-MAPA-RAPIDO-MODULOS.md",
		"docs-ai/deliveries/INDEX.md",
		"docs-ai/deliveries/_template/report.md",
		"AGENTS.md",
		"GEMINI.md",
	}

	for _, f := range expectedFiles {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("arquivo esperado %s não existe: %v", f, err)
		}
	}

	// 4. Validar substituição de placeholders
	agentsDoc, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("falha ao ler AGENTS.md: %v", err)
	}
	if strings.Contains(string(agentsDoc), "{{NOME_DO_PROJETO}}") {
		t.Errorf("AGENTS.md ainda contém placeholder {{NOME_DO_PROJETO}}")
	}
	if !strings.Contains(string(agentsDoc), "test-desktop-app") {
		t.Errorf("AGENTS.md não contém nome do projeto interpolado")
	}
	if !strings.Contains(string(agentsDoc), "PostgreSQL") {
		t.Errorf("AGENTS.md não contém banco PostgreSQL")
	}
}

func TestScaffoldAPI(t *testing.T) {
	dir := t.TempDir()

	opts := Options{
		Name:      "test-api-svc",
		Type:      "api",
		Database:  "mongo",
		Module:    "github.com/test/api-svc",
		TargetDir: dir,
		InitGit:   false,
	}

	if err := Scaffold(opts); err != nil {
		t.Fatalf("Scaffold falhou: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load falhou: %v", err)
	}
	if cfg.ProjectType != "api" {
		t.Errorf("esperava project_type api, obteve %s", cfg.ProjectType)
	}

	expectedFiles := []string{
		"cmd/api/main.go",
		"Taskfile.yml",
		"migrations",
		".agents/skills/pm-triage/SKILL.md",
		"AGENTS.md",
	}

	for _, f := range expectedFiles {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("arquivo esperado %s não existe: %v", f, err)
		}
	}

	agentsDoc, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if !strings.Contains(string(agentsDoc), "MongoDB") {
		t.Errorf("AGENTS.md não contém MongoDB interpolado")
	}

	for _, a := range cfg.EditorAgents {
		if a == "query-performance" {
			t.Errorf("query-performance não deve constar em editor_agents do template api (é agente de análise/revisão, D-04)")
		}
	}
}

func TestTemplateAPIEditorAgents(t *testing.T) {
	data, err := templateFS.ReadFile("templates/pipeline-ai.api.json")
	if err != nil {
		t.Fatalf("falha ao ler templates/pipeline-ai.api.json: %v", err)
	}
	var tmpl struct {
		EditorAgents []string `json:"editor_agents"`
	}
	if err := json.Unmarshal(data, &tmpl); err != nil {
		t.Fatalf("falha ao parsear templates/pipeline-ai.api.json: %v", err)
	}
	for _, a := range tmpl.EditorAgents {
		if a == "query-performance" {
			t.Errorf("templates/pipeline-ai.api.json contém query-performance em editor_agents, mas este é um agente de análise/revisão (D-04)")
		}
	}
}

func TestScaffoldCLI(t *testing.T) {
	dir := t.TempDir()

	opts := Options{
		Name:      "test-cli-tool",
		Type:      "cli",
		Database:  "postgres",
		Module:    "github.com/test/cli-tool",
		TargetDir: dir,
		InitGit:   false,
	}

	if err := Scaffold(opts); err != nil {
		t.Fatalf("Scaffold falhou: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load falhou: %v", err)
	}
	if cfg.ProjectType != "cli" {
		t.Errorf("esperava project_type cli, obteve %s", cfg.ProjectType)
	}

	expectedFiles := []string{
		"cmd/test-cli-tool/main.go",
		"Taskfile.yml",
		".pipeline-ai.json",
		".agents/skills/pm-triage/SKILL.md",
		"AGENTS.md",
		"GEMINI.md",
		"docs-ai/01-INVARIANTES-GLOBAIS.md",
	}

	for _, f := range expectedFiles {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("arquivo esperado %s não existe: %v", f, err)
		}
	}

	frontendDir := filepath.Join(dir, "frontend")
	if _, err := os.Stat(frontendDir); !os.IsNotExist(err) {
		t.Errorf("projeto cli não deveria conter diretório frontend/: %v", err)
	}

	agentsDoc, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("falha ao ler AGENTS.md: %v", err)
	}
	if !strings.Contains(string(agentsDoc), "**Formato:** cli") {
		t.Errorf("AGENTS.md não contém formato cli interpolado: %s", string(agentsDoc))
	}
	if strings.Contains(string(agentsDoc), "frontend/src") {
		t.Errorf("AGENTS.md de projeto CLI não deveria referenciar frontend/src")
	}
}

func TestScaffoldCLIAliasTUI(t *testing.T) {
	opts := Options{
		Name: "test-tui-app",
		Type: "tui",
	}
	if err := opts.Normalize(); err != nil {
		t.Fatalf("Normalize() falhou para alias 'tui': %v", err)
	}
	if opts.Type != "cli" {
		t.Errorf("esperava tipo normalizado para 'cli', obteve %s", opts.Type)
	}
}

func TestScaffoldValidation(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr bool
	}{
		{
			name:    "nome ausente",
			opts:    Options{Type: "desktop"},
			wantErr: true,
		},
		{
			name:    "tipo inválido",
			opts:    Options{Name: "app", Type: "mobile"},
			wantErr: true,
		},
		{
			name:    "válido desktop",
			opts:    Options{Name: "app", Type: "desktop"},
			wantErr: false,
		},
		{
			name:    "válido cli",
			opts:    Options{Name: "app", Type: "cli"},
			wantErr: false,
		},
		{
			name:    "válido alias tui",
			opts:    Options{Name: "app", Type: "tui"},
			wantErr: false,
		},
		{
			name:    "nome malicioso path traversal relativo duplo",
			opts:    Options{Name: "../../escape", Type: "desktop"},
			wantErr: true,
		},
		{
			name:    "nome malicioso path traversal relativo simples",
			opts:    Options{Name: "../escape", Type: "desktop"},
			wantErr: true,
		},
		{
			name:    "nome com barra",
			opts:    Options{Name: "foo/bar", Type: "desktop"},
			wantErr: true,
		},
		{
			name:    "nome com caminho absoluto",
			opts:    Options{Name: "/tmp/abs", Type: "desktop"},
			wantErr: true,
		},
		{
			name:    "nome com caractere especial arroba",
			opts:    Options{Name: "test@name", Type: "desktop"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Normalize()
			if (err != nil) != tt.wantErr {
				t.Errorf("Normalize() erro = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestScaffoldInvalidProjectNames(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
	}{
		{name: "path traversal relativo duplo", projectName: "../../escape"},
		{name: "path traversal relativo simples", projectName: "../escape"},
		{name: "subdiretorio com barra", projectName: "foo/bar"},
		{name: "caminho absoluto unix", projectName: "/tmp/abs"},
		{name: "caractere especial arroba", projectName: "test@name"},
		{name: "caractere especial espaco", projectName: "test project"},
		{name: "ponto unico", projectName: "."},
		{name: "ponto duplo", projectName: ".."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			targetDir := filepath.Join(dir, "target")
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}

			opts := Options{
				Name:      tt.projectName,
				Type:      "cli",
				TargetDir: targetDir,
			}

			err := Scaffold(opts)
			if err == nil {
				t.Fatalf("Scaffold(%q) deveria ter rejeitado nome inválido/malicioso com erro descritivo, mas retornou nil", tt.projectName)
			}

			// Verificar que nenhum arquivo foi criado fora de targetDir (no diretório pai dir)
			entries, readErr := os.ReadDir(dir)
			if readErr != nil {
				t.Fatalf("ReadDir falhou: %v", readErr)
			}
			for _, entry := range entries {
				if entry.Name() != "target" {
					t.Errorf("arquivo ou diretório %q criado fora de TargetDir para nome malicioso %q", entry.Name(), tt.projectName)
				}
			}
		})
	}
}

func TestScaffoldWithGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git não encontrado no PATH")
	}

	dir := t.TempDir()
	opts := Options{
		Name:      "test-git-app",
		Type:      "desktop",
		TargetDir: dir,
		InitGit:   true,
	}

	if err := Scaffold(opts); err != nil {
		t.Fatalf("Scaffold falhou: %v", err)
	}

	gitDir := filepath.Join(dir, ".git")
	if info, err := os.Stat(gitDir); err != nil || !info.IsDir() {
		t.Fatalf("esperava diretório .git criado: %v", err)
	}
}

func TestSecurityShiftLeftTemplates(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		mustContain []string
	}{
		{
			name:     "handoff-contract define formalmente REQUISITOS_SEGURANCA condicionado a SECURITY: sim",
			filePath: "templates/agents/rules/handoff-contract.md",
			mustContain: []string{
				"REQUISITOS_SEGURANCA",
				"SECURITY: sim",
			},
		},
		{
			name:     "architecture-guardrails instrui modelagem de ameacas e REQUISITOS_SEGURANCA no handoff",
			filePath: "templates/agents/skills/architecture-guardrails/SKILL.md",
			mustContain: []string{
				"REQUISITOS_SEGURANCA",
				"SECURITY: sim",
				"ameaça",
			},
		},
		{
			name:     "test-writer instrui criacao de testes adversariais e negativos para REQUISITOS_SEGURANCA",
			filePath: "templates/agents/skills/test-writer/SKILL.md",
			mustContain: []string{
				"REQUISITOS_SEGURANCA",
				"adversaria",
			},
		},
		{
			name:     "security-review instrui auditoria cruzada contra REQUISITOS_SEGURANCA do PRE",
			filePath: "templates/agents/skills/security-review/SKILL.md",
			mustContain: []string{
				"REQUISITOS_SEGURANCA",
			},
		},
		{
			name:     "DECISOES.md atualiza D-13 com modelo bidirecional PRE e POST",
			filePath: "templates/docs-ai/DECISOES.md",
			mustContain: []string{
				"D-13",
				"bidirecional",
			},
		},
		{
			name:     "03-MAPA-RAPIDO-MODULOS.md reflete seguranca no PRE e POST",
			filePath: "templates/docs-ai/03-MAPA-RAPIDO-MODULOS.md",
			mustContain: []string{
				"REQUISITOS_SEGURANCA",
			},
		},
		{
			name:     "AGENTS.md.tmpl reflete fluxo com seguranca no PRE e POST",
			filePath: "templates/AGENTS.md.tmpl",
			mustContain: []string{
				"REQUISITOS_SEGURANCA",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := templateFS.ReadFile(tt.filePath)
			if err != nil {
				t.Fatalf("falha ao ler arquivo embutido %s: %v", tt.filePath, err)
			}
			content := string(data)
			for _, exp := range tt.mustContain {
				if !strings.Contains(strings.ToLower(content), strings.ToLower(exp)) {
					t.Errorf("template %s não contém %q", tt.filePath, exp)
				}
			}
		})
	}
}
