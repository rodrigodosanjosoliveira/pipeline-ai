package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir falhou para %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile falhou para %s: %v", path, err)
	}
}

func setupBaseWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Configuração do pipeline-ai
	cfgContent := `{
  "version": 1,
  "project_type": "api",
  "protected_paths": ["internal/"],
  "editor_agents": ["dev-implementation", "test-writer", "docs-update"]
}`
	writeFile(t, filepath.Join(dir, ".pipeline-ai.json"), cfgContent)

	// Módulo Go com código válido formatado e teste que passa
	writeFile(t, filepath.Join(dir, "go.mod"), "module sampleapp\n\ngo 1.24\n")
	writeFile(t, filepath.Join(dir, "sample.go"), "package sampleapp\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n")
	writeFile(t, filepath.Join(dir, "sample_test.go"), "package sampleapp\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(2, 3) != 5 {\n\t\tt.Fatal(\"soma incorreta\")\n\t}\n}\n")

	// Estrutura de hooks e skills para o doctor passar
	writeFile(t, filepath.Join(dir, ".githooks", "pre-push"), "#!/bin/sh\nexit 0\n")
	writeFile(t, filepath.Join(dir, ".git", "HEAD"), "ref: refs/heads/main\n")
	writeFile(t, filepath.Join(dir, ".agents", "hooks.json"), "{}")
	writeFile(t, filepath.Join(dir, ".agents", "skills", "test-writer", "SKILL.md"), "# Test Writer")

	return dir
}

func setupValidDelivery(t *testing.T, dir, deliveryID, taskCode string) {
	t.Helper()
	if err := state.Open(dir, deliveryID, taskCode, 24*time.Hour); err != nil {
		t.Fatalf("state.Open falhou: %v", err)
	}

	reportContent := `---
delivery_id: ` + deliveryID + `
task_code: ` + taskCode + `
title: Delivery Teste
status: ready
require_security: false
require_migration: false
require_performance: false
---

# Contexto

Contexto da delivery de teste.

# Escopo

Escopo da delivery de teste.

# Implementação

Implementação da delivery de teste.

# Testes

=== RUN   TestAdd
--- PASS: TestAdd (0.00s)
PASS
ok  	sampleapp	0.002s

# Code Review

Aprovado sem ressalvas.

# Final Checklist

- [x] Testes passando
- [x] Formatação gofmt
- [x] Análise go vet
`
	writeFile(t, filepath.Join(dir, "docs-ai", "deliveries", deliveryID, "report.md"), reportContent)
	writeFile(t, filepath.Join(dir, "docs-ai", "deliveries", "INDEX.md"), "| "+deliveryID+" | "+taskCode+" | Delivery Teste | ready |\n")
}

func findStep(steps []StepResult, name string) *StepResult {
	for i := range steps {
		if steps[i].Name == name {
			return &steps[i]
		}
	}
	return nil
}

func TestRunner_SuccessWithActiveDelivery(t *testing.T) {
	dir := setupBaseWorkspace(t)
	setupValidDelivery(t, dir, "DEL-001", "TASK-001")

	opts := Options{
		Dir:   dir,
		NoGit: true,
	}

	res, err := Run(opts)
	if err != nil {
		t.Fatalf("Run retornou erro inesperado: %v", err)
	}
	if res == nil {
		t.Fatal("res retornado é nil (Run não implementado)")
	}

	if res.Overall != StatusPass {
		t.Errorf("res.Overall = %q, esperado %q", res.Overall, StatusPass)
	}
	if !res.OK {
		t.Errorf("res.OK = false, esperado true")
	}

	stepsExpected := []string{"doctor", "sensor", "validate"}
	for _, stepName := range stepsExpected {
		step := findStep(res.Steps, stepName)
		if step == nil {
			t.Fatalf("step %q não encontrado em res.Steps: %+v", stepName, res.Steps)
		}
		if step.Status != StatusPass {
			t.Errorf("step %q status = %q, esperado %q (msg: %s)", stepName, step.Status, StatusPass, step.Message)
		}
	}
}

func TestRunner_NoActiveDelivery_SensorsAndDoctorOnly(t *testing.T) {
	dir := setupBaseWorkspace(t)
	// Sem active delivery e sem relatório configurado

	opts := Options{
		Dir:   dir,
		NoGit: true,
	}

	res, err := Run(opts)
	if err != nil {
		t.Fatalf("Run retornou erro inesperado: %v", err)
	}
	if res == nil {
		t.Fatal("res retornado é nil (Run não implementado)")
	}

	if res.Overall != StatusPass {
		t.Errorf("res.Overall = %q, esperado %q", res.Overall, StatusPass)
	}
	if !res.OK {
		t.Errorf("res.OK = false, esperado true")
	}

	doctorStep := findStep(res.Steps, "doctor")
	if doctorStep == nil || doctorStep.Status != StatusPass {
		t.Errorf("doctorStep = %+v, esperado StatusPass", doctorStep)
	}

	sensorStep := findStep(res.Steps, "sensor")
	if sensorStep == nil || sensorStep.Status != StatusPass {
		t.Errorf("sensorStep = %+v, esperado StatusPass", sensorStep)
	}

	validateStep := findStep(res.Steps, "validate")
	if validateStep == nil {
		t.Fatal("validateStep não encontrado em res.Steps")
	}
	if validateStep.Status != StatusSkip {
		t.Errorf("validateStep status = %q, esperado %q quando não há delivery ativa", validateStep.Status, StatusSkip)
	}
}

func TestRunner_SensorFailure(t *testing.T) {
	tests := []struct {
		name         string
		setupCorrupt func(t *testing.T, dir string)
	}{
		{
			name: "falha_gofmt",
			setupCorrupt: func(t *testing.T, dir string) {
				// Arquivo Go com formatação fora do padrão gofmt
				badGo := "package sampleapp\n\nfunc Unformatted(  a int  )   int{\nreturn a\n}\n"
				writeFile(t, filepath.Join(dir, "badfmt.go"), badGo)
			},
		},
		{
			name: "falha_teste_quebrando",
			setupCorrupt: func(t *testing.T, dir string) {
				failingTest := "package sampleapp\n\nimport \"testing\"\n\nfunc TestBroken(t *testing.T) {\n\tt.Fatal(\"falha intencional de teste\")\n}\n"
				writeFile(t, filepath.Join(dir, "broken_test.go"), failingTest)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupBaseWorkspace(t)
			setupValidDelivery(t, dir, "DEL-001", "TASK-001")
			tt.setupCorrupt(t, dir)

			opts := Options{
				Dir:   dir,
				NoGit: true,
			}

			res, err := Run(opts)
			if err != nil {
				t.Fatalf("Run retornou erro inesperado: %v", err)
			}
			if res == nil {
				t.Fatal("res retornado é nil (Run não implementado)")
			}

			if res.Overall != StatusFail {
				t.Errorf("res.Overall = %q, esperado %q", res.Overall, StatusFail)
			}
			if res.OK {
				t.Errorf("res.OK = true, esperado false em caso de falha de sensor")
			}

			sensorStep := findStep(res.Steps, "sensor")
			if sensorStep == nil {
				t.Fatal("sensorStep ausente em res.Steps")
			}
			if sensorStep.Status != StatusFail {
				t.Errorf("sensorStep status = %q, esperado %q", sensorStep.Status, StatusFail)
			}
		})
	}
}

func TestRunner_ValidateFailure(t *testing.T) {
	tests := []struct {
		name         string
		setupInvalid func(t *testing.T, dir string)
	}{
		{
			name: "checklist_incompleto",
			setupInvalid: func(t *testing.T, dir string) {
				setupValidDelivery(t, dir, "DEL-001", "TASK-001")
				reportPath := filepath.Join(dir, "docs-ai", "deliveries", "DEL-001", "report.md")
				content, err := os.ReadFile(reportPath)
				if err != nil {
					t.Fatalf("leitura do report falhou: %v", err)
				}
				// Substitui um item marcado por desmarcado
				corrupted := string(content) + "\n- [ ] Checklist pendente obrigatório\n"
				writeFile(t, reportPath, corrupted)
			},
		},
		{
			name: "sem_evidencia_de_testes",
			setupInvalid: func(t *testing.T, dir string) {
				setupValidDelivery(t, dir, "DEL-001", "TASK-001")
				reportPath := filepath.Join(dir, "docs-ai", "deliveries", "DEL-001", "report.md")
				badReport := `---
delivery_id: DEL-001
task_code: TASK-001
title: Delivery Teste
status: ready
require_security: false
require_migration: false
require_performance: false
---

# Contexto
Contexto.

# Escopo
Escopo.

# Implementação
Implementação.

# Testes
go test ./... foram executados com sucesso mas sem log real.

# Code Review
Aprovado sem ressalvas.

# Final Checklist
- [x] Item 1
`
				writeFile(t, reportPath, badReport)
			},
		},
		{
			name: "relatorio_inexistente_com_delivery_ativa",
			setupInvalid: func(t *testing.T, dir string) {
				if err := state.Open(dir, "DEL-404", "TASK-404", 24*time.Hour); err != nil {
					t.Fatalf("state.Open falhou: %v", err)
				}
				// Não cria docs-ai/deliveries/DEL-404/report.md
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupBaseWorkspace(t)
			tt.setupInvalid(t, dir)

			opts := Options{
				Dir:   dir,
				NoGit: true,
			}

			res, err := Run(opts)
			if err != nil {
				t.Fatalf("Run retornou erro inesperado: %v", err)
			}
			if res == nil {
				t.Fatal("res retornado é nil (Run não implementado)")
			}

			if res.Overall != StatusFail {
				t.Errorf("res.Overall = %q, esperado %q", res.Overall, StatusFail)
			}
			if res.OK {
				t.Errorf("res.OK = true, esperado false quando validate falha")
			}

			validateStep := findStep(res.Steps, "validate")
			if validateStep == nil {
				t.Fatal("validateStep ausente em res.Steps")
			}
			if validateStep.Status != StatusFail {
				t.Errorf("validateStep status = %q, esperado %q", validateStep.Status, StatusFail)
			}
		})
	}
}
