package scaffold

import (
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
			name:    "válido",
			opts:    Options{Name: "app", Type: "desktop"},
			wantErr: false,
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
