package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIRouting(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantExit   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "ajuda sem argumentos",
			args:       []string{},
			wantExit:   0,
			wantStdout: "Uso: pipeline-ai <comando> [argumentos]",
		},
		{
			name:       "comando help",
			args:       []string{"help"},
			wantExit:   0,
			wantStdout: "Comandos disponíveis:",
		},
		{
			name:       "comando version",
			args:       []string{"version"},
			wantExit:   0,
			wantStdout: "pipeline-ai v0.",
		},
		{
			name:       "comando desconhecido",
			args:       []string{"comando-inexistente"},
			wantExit:   1,
			wantStderr: "comando desconhecido: \"comando-inexistente\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exitCode := run(tt.args, &stdout, &stderr)

			if exitCode != tt.wantExit {
				t.Errorf("run(%v) exitCode = %d, esperado %d", tt.args, exitCode, tt.wantExit)
			}
			if tt.wantStdout != "" && !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout não contém %q; saída:\n%s", tt.wantStdout, stdout.String())
			}
			if tt.wantStderr != "" && !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr não contém %q; saída:\n%s", tt.wantStderr, stderr.String())
			}
		})
	}
}

func TestCLISyncAndDoctor(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Doctor em diretório vazio
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"doctor", "-dir", tmpDir}, &stdout, &stderr)
	if exitCode == 0 {
		t.Errorf("doctor deveria retornar código diferente de 0 em diretório vazio sem config")
	}
	if !strings.Contains(stdout.String(), "pipeline-ai doctor") {
		t.Errorf("saída do doctor inesperada:\n%s", stdout.String())
	}

	// 2. Init com flags não-interativas
	stdout.Reset()
	stderr.Reset()
	initArgs := []string{
		"init",
		"-name", "meu-app-teste",
		"-type", "desktop",
		"-db", "postgres",
		"-dir", tmpDir,
		"-no-git",
	}
	exitCode = run(initArgs, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run(init) falhou: exitCode=%d, stderr:\n%s", exitCode, stderr.String())
	}

	// 3. Sync no projeto recém-criado
	stdout.Reset()
	stderr.Reset()
	exitCode = run([]string{"sync", "-dir", tmpDir}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run(sync) falhou: exitCode=%d, stderr:\n%s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Sincronização concluída com sucesso") {
		t.Errorf("saída do sync inesperada:\n%s", stdout.String())
	}

	// 4. Doctor no projeto inicializado
	stdout.Reset()
	stderr.Reset()
	_ = run([]string{"doctor", "-dir", tmpDir}, &stdout, &stderr)
	// Deve reportar Go Runtime e Configuração (.pipeline-ai.json) com PASS
	if !strings.Contains(stdout.String(), "Configuração (.pipeline-ai.json)") {
		t.Errorf("doctor não encontrou configuração:\n%s", stdout.String())
	}

	// 5. Doctor com flag -json
	stdout.Reset()
	stderr.Reset()
	_ = run([]string{"doctor", "-dir", tmpDir, "-json"}, &stdout, &stderr)
	if !strings.Contains(stdout.String(), `"ok":`) || !strings.Contains(stdout.String(), `"results":`) {
		t.Errorf("doctor -json não retornou campos esperados:\n%s", stdout.String())
	}
}

func TestCLIValidate(t *testing.T) {
	var stdout, stderr bytes.Buffer

	// validate sem argumento de relatório
	exitCode := run([]string{"validate"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Errorf("validate sem argumentos deveria falhar com erro de uso")
	}

	// validate com arquivo inexistente
	stdout.Reset()
	stderr.Reset()
	exitCode = run([]string{"validate", "arquivo-inexistente.md"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Errorf("validate com arquivo inexistente deveria falhar")
	}

	// validate com relatório válido e no-git
	tmpDir := t.TempDir()
	validReport := `---
delivery_id: DEL-999
task_code: TASK-1
title: Teste CLI
type: feat
status: draft
require_security: false
date: 2026-10-21
---

## Contexto

Texto de contexto explicativo sobre a entrega técnica realizada.

## Escopo

- Módulo testado

## Implementação

Implementação realizada de acordo com o escopo aprovado.

## Testes

Testes de unidade executados.

## Code Review

Revisão técnica executada.

## Final Checklist

- [x] Testes passando
- [x] Invariantes respeitados
- [x] Sem warnings
`
	reportPath := filepath.Join(tmpDir, "report.md")
	if err := os.WriteFile(reportPath, []byte(validReport), 0644); err != nil {
		t.Fatal(err)
	}

	// Cria config fake e index fake
	cfg := `{"version": 1, "project_type": "desktop", "protected_paths": ["internal/"], "editor_agents": ["dev-implementation"]}`
	if err := os.WriteFile(filepath.Join(tmpDir, ".pipeline-ai.json"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	delivDir := filepath.Join(tmpDir, "docs-ai", "deliveries")
	_ = os.MkdirAll(delivDir, 0755)
	indexContent := "| DEL-999 | TASK-1 | Teste CLI | feat | 2026-10-21 | draft |\n"
	if err := os.WriteFile(filepath.Join(delivDir, "INDEX.md"), []byte(indexContent), 0644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	exitCode = run([]string{"validate", "-no-git", reportPath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("validate falhou em relatório válido: exitCode=%d, stdout:\n%s\nstderr:\n%s", exitCode, stdout.String(), stderr.String())
	}
}
