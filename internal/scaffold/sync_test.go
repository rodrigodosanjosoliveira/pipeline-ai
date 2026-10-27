package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncAgents(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Diretório sem .agents deve falhar ou avisar que não é um projeto inicializado
	var stdout, stderr bytes.Buffer
	err := Sync(tmpDir, &stdout, &stderr)
	if err == nil {
		t.Errorf("Sync em diretório sem .agents deveria retornar erro")
	}

	// 2. Simula projeto existente inicializado
	agentsDir := filepath.Join(tmpDir, ".agents")
	skillsDir := filepath.Join(agentsDir, "skills")
	rulesDir := filepath.Join(agentsDir, "rules")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Arquivo customizado que NÃO deve ser apagado pelo sync
	customRule := filepath.Join(rulesDir, "minha-regra-customizada.md")
	if err := os.WriteFile(customRule, []byte("# Regra do Usuário"), 0644); err != nil {
		t.Fatal(err)
	}

	// Arquivo de negócio que NUNCA deve ser tocado
	userCode := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(userCode, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Executa sync
	stdout.Reset()
	stderr.Reset()
	err = Sync(tmpDir, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Sync falhou em projeto com .agents: %v", err)
	}

	// Validações pós-sync:
	// a) Regra customizada do usuário deve continuar existindo intacta
	if data, err := os.ReadFile(customRule); err != nil || string(data) != "# Regra do Usuário" {
		t.Errorf("Sync apagou ou corrompeu regra customizada do usuário: %v", err)
	}

	// b) Arquivo de negócio main.go deve continuar intacto
	if data, err := os.ReadFile(userCode); err != nil || string(data) != "package main\nfunc main() {}\n" {
		t.Errorf("Sync modificou código de negócio do usuário")
	}

	// c) Skills oficiais devem ter sido sincronizadas
	triageSkill := filepath.Join(skillsDir, "pm-triage", "SKILL.md")
	if _, err := os.Stat(triageSkill); os.IsNotExist(err) {
		t.Errorf("Sync não sincronizou pm-triage/SKILL.md: %v", err)
	}

	// d) hooks.json deve existir
	hooksJSON := filepath.Join(agentsDir, "hooks.json")
	if _, err := os.Stat(hooksJSON); os.IsNotExist(err) {
		t.Errorf("Sync não sincronizou hooks.json: %v", err)
	}

	// e) orchestration.md deve ter sido sincronizado
	orchestrationRule := filepath.Join(rulesDir, "orchestration.md")
	if _, err := os.Stat(orchestrationRule); os.IsNotExist(err) {
		t.Errorf("Sync não sincronizou .agents/rules/orchestration.md: %v", err)
	}
}

func TestSyncSecurityShiftLeft(t *testing.T) {
	tmpDir := t.TempDir()

	agentsDir := filepath.Join(tmpDir, ".agents")
	skillsDir := filepath.Join(agentsDir, "skills")
	rulesDir := filepath.Join(agentsDir, "rules")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Escreve arquivos legados/desatualizados sem REQUISITOS_SEGURANCA
	oldHandoff := filepath.Join(rulesDir, "handoff-contract.md")
	if err := os.WriteFile(oldHandoff, []byte("# Contrato Legado\nSem campos de seguranca no PRE\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := Sync(tmpDir, &stdout, &stderr); err != nil {
		t.Fatalf("Sync falhou: %v", err)
	}

	// Valida que handoff-contract.md e as skills sincronizadas possuem REQUISITOS_SEGURANCA
	tests := []struct {
		name    string
		relPath string
		want    string
	}{
		{
			name:    "handoff-contract sincronizado contém REQUISITOS_SEGURANCA",
			relPath: filepath.Join(".agents", "rules", "handoff-contract.md"),
			want:    "REQUISITOS_SEGURANCA",
		},
		{
			name:    "architecture-guardrails sincronizado contém REQUISITOS_SEGURANCA",
			relPath: filepath.Join(".agents", "skills", "architecture-guardrails", "SKILL.md"),
			want:    "REQUISITOS_SEGURANCA",
		},
		{
			name:    "test-writer sincronizado contém REQUISITOS_SEGURANCA",
			relPath: filepath.Join(".agents", "skills", "test-writer", "SKILL.md"),
			want:    "REQUISITOS_SEGURANCA",
		},
		{
			name:    "security-review sincronizado contém REQUISITOS_SEGURANCA",
			relPath: filepath.Join(".agents", "skills", "security-review", "SKILL.md"),
			want:    "REQUISITOS_SEGURANCA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.relPath)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("falha ao ler arquivo sincronizado %s: %v", tt.relPath, err)
			}
			if !strings.Contains(string(data), tt.want) {
				t.Errorf("arquivo sincronizado %s não contém %q", tt.relPath, tt.want)
			}
		})
	}
}
