package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
			wantStdout: "pipeline-ai v0.9.0",
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

func TestCLIMCP(t *testing.T) {
	// Testa inicialização e encerramento limpo do MCP via CLI
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}\n")
	exitCode := runWithStdin([]string{"mcp"}, stdin, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("runWithStdin(mcp) esperava exit code 0, obteve %d. Stderr: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"protocolVersion":"2024-11-05"`) {
		t.Errorf("resposta do initialize não encontrada no stdout:\n%s", stdout.String())
	}
}

func TestCLIInitCLI(t *testing.T) {
	tmpDir := t.TempDir()

	var stdout, stderr bytes.Buffer
	initArgs := []string{
		"init",
		"-name", "ferramenta-cli",
		"-type", "cli",
		"-db", "postgres",
		"-dir", tmpDir,
		"-no-git",
	}
	exitCode := run(initArgs, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run(init -type cli) falhou: exitCode=%d, stderr:\n%s", exitCode, stderr.String())
	}

	mainFile := filepath.Join(tmpDir, "cmd", "ferramenta-cli", "main.go")
	if _, err := os.Stat(mainFile); err != nil {
		t.Errorf("arquivo principal do CLI %s não foi criado: %v", mainFile, err)
	}
}

func TestCLIDeliveryCommands(t *testing.T) {
	tmpDir := t.TempDir()

	// Cria arquivo .pipeline-ai.json para simular projeto válido
	cfg := `{"version": 1, "project_type": "api", "protected_paths": ["internal/"]}`
	if err := os.WriteFile(filepath.Join(tmpDir, ".pipeline-ai.json"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		args       []string
		wantExit   int
		wantStdout string
		wantStderr string
		checkState func(t *testing.T, dir string)
	}{
		{
			name:       "delivery open sem argumento falha com erro de uso",
			args:       []string{"delivery", "open", "-dir", tmpDir},
			wantExit:   1,
			wantStderr: "delivery_id é obrigatório",
		},
		{
			name:       "delivery open DEL-001 abre com sucesso",
			args:       []string{"delivery", "open", "DEL-001", "-dir", tmpDir, "-ttl", "1h"},
			wantExit:   0,
			wantStdout: "DEL-001",
			checkState: func(t *testing.T, dir string) {
				stateFile := filepath.Join(dir, ".pipeline-state", "active-delivery.json")
				data, err := os.ReadFile(stateFile)
				if err != nil {
					t.Fatalf("arquivo %s não foi criado: %v", stateFile, err)
				}
				if !strings.Contains(string(data), "DEL-001") {
					t.Errorf("conteúdo de active-delivery.json não contém DEL-001: %s", string(data))
				}
			},
		},
		{
			name:       "delivery status com delivery ativa exibe detalhes",
			args:       []string{"delivery", "status", "-dir", tmpDir},
			wantExit:   0,
			wantStdout: "DEL-001",
		},
		{
			name:       "delivery close fecha a delivery ativa",
			args:       []string{"delivery", "close", "-dir", tmpDir},
			wantExit:   0,
			wantStdout: "fechada",
			checkState: func(t *testing.T, dir string) {
				stateFile := filepath.Join(dir, ".pipeline-state", "active-delivery.json")
				if data, err := os.ReadFile(stateFile); err == nil && len(strings.TrimSpace(string(data))) > 0 {
					if strings.Contains(string(data), "DEL-001") {
						t.Errorf("active-delivery.json ainda contém DEL-001 após close: %s", string(data))
					}
				}
			},
		},
		{
			name:       "delivery status apos close indica nenhuma delivery ativa",
			args:       []string{"delivery", "status", "-dir", tmpDir},
			wantExit:   0,
			wantStdout: "nenhuma delivery ativa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exitCode := run(tt.args, &stdout, &stderr)

			if exitCode != tt.wantExit {
				t.Errorf("run(%v) exitCode = %d, quer %d (stdout: %s, stderr: %s)", tt.args, exitCode, tt.wantExit, stdout.String(), stderr.String())
			}

			if tt.wantStdout != "" {
				combined := strings.ToLower(stdout.String())
				if !strings.Contains(combined, strings.ToLower(tt.wantStdout)) {
					t.Errorf("stdout não contém %q; saída:\n%s", tt.wantStdout, stdout.String())
				}
			}

			if tt.wantStderr != "" {
				combined := strings.ToLower(stderr.String() + stdout.String())
				if !strings.Contains(combined, strings.ToLower(tt.wantStderr)) {
					t.Errorf("stderr não contém %q; saída:\n%s", tt.wantStderr, stderr.String())
				}
			}

			if tt.checkState != nil {
				tt.checkState(t, tmpDir)
			}
		})
	}
}

func TestCLIValidateJSON(t *testing.T) {
	tmpDir := t.TempDir()
	validReport := `---
delivery_id: DEL-999
task_code: TASK-1
title: Teste CLI JSON
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

$ go test ./...
ok   testmod 0.1s
PASS

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

	cfg := `{"version": 1, "project_type": "desktop", "protected_paths": ["internal/"], "editor_agents": ["dev-implementation"]}`
	if err := os.WriteFile(filepath.Join(tmpDir, ".pipeline-ai.json"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	delivDir := filepath.Join(tmpDir, "docs-ai", "deliveries")
	_ = os.MkdirAll(delivDir, 0755)
	indexContent := "| DEL-999 | TASK-1 | Teste CLI JSON | feat | 2026-10-21 | draft |\n"
	if err := os.WriteFile(filepath.Join(delivDir, "INDEX.md"), []byte(indexContent), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"validate", "-json", "-no-git", reportPath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("validate -json falhou com exitCode = %d, stderr: %s", exitCode, stderr.String())
	}

	var res map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("stdout de validate -json não é JSON válido: %v (raw: %s)", err, stdout.String())
	}

	if res["status"] != "PASS" {
		t.Errorf("res[\"status\"] = %v, quer PASS", res["status"])
	}
	if res["ok"] != true {
		t.Errorf("res[\"ok\"] = %v, quer true", res["ok"])
	}
}

func TestCLISensorJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Cria estrutura Go mínima válida no tmpDir
	goMod := "module testcli\n\ngo 1.24\n"
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)
	goCode := "package main\n\nfunc Hello() string { return \"world\" }\n"
	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(goCode), 0644)
	goTest := "package main\n\nimport \"testing\"\n\nfunc TestHello(t *testing.T) {\n\tif Hello() != \"world\" {\n\t\tt.Fatal(\"err\")\n\t}\n}\n"
	_ = os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte(goTest), 0644)

	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"sensor", "-dir", tmpDir, "-json"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("sensor -json falhou com exitCode = %d, stderr: %s", exitCode, stderr.String())
	}

	var res map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("stdout de sensor -json não é JSON válido: %v (raw: %s)", err, stdout.String())
	}

	if res["overall"] != "PASS" {
		t.Errorf("res[\"overall\"] = %v, quer PASS", res["overall"])
	}
	if res["ok"] != true {
		t.Errorf("res[\"ok\"] = %v, quer true", res["ok"])
	}
}

func TestCLIModuleCommands(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := `{
		"version": 1,
		"project_type": "desktop",
		"protected_paths": ["internal/"],
		"editor_agents": ["dev-implementation"],
		"enabled_modules": ["release-ops"]
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, ".pipeline-ai.json"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}

	// 1. pipeline-ai module list
	t.Run("module list", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"module", "list", "-dir", tmpDir}, &stdout, &stderr)
		if exitCode != 0 {
			t.Errorf("module list falhou com exitCode = %d, stderr: %s", exitCode, stderr.String())
		}
		out := stdout.String()
		if !strings.Contains(out, "release-ops") || !strings.Contains(out, "data-model") {
			t.Errorf("module list não listou módulos esperados. Saída:\n%s", out)
		}
		// Deve indicar que release-ops está ativo
		if !strings.Contains(strings.ToLower(out), "ativo") && !strings.Contains(strings.ToLower(out), "enabled") {
			t.Errorf("module list não indicou status de ativação. Saída:\n%s", out)
		}
	})

	// 2. pipeline-ai module enable <modulo>
	t.Run("module enable valido", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"module", "enable", "data-model", "-dir", tmpDir}, &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("module enable falhou com exitCode = %d, stderr: %s", exitCode, stderr.String())
		}

		data, err := os.ReadFile(filepath.Join(tmpDir, ".pipeline-ai.json"))
		if err != nil {
			t.Fatal(err)
		}
		var parsed struct {
			EnabledModules []string `json:"enabled_modules"`
		}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("JSON corrompido após enable: %v", err)
		}
		found := false
		for _, m := range parsed.EnabledModules {
			if m == "data-model" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("módulo data-model não foi persistido em enabled_modules: %v", parsed.EnabledModules)
		}
	})

	// 3. pipeline-ai module disable <modulo>
	t.Run("module disable valido", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"module", "disable", "release-ops", "-dir", tmpDir}, &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("module disable falhou com exitCode = %d, stderr: %s", exitCode, stderr.String())
		}

		data, err := os.ReadFile(filepath.Join(tmpDir, ".pipeline-ai.json"))
		if err != nil {
			t.Fatal(err)
		}
		var parsed struct {
			EnabledModules []string `json:"enabled_modules"`
		}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("JSON corrompido após disable: %v", err)
		}
		for _, m := range parsed.EnabledModules {
			if m == "release-ops" {
				t.Errorf("módulo release-ops ainda consta em enabled_modules após disable")
			}
		}
	})

	// 4. pipeline-ai module enable <modulo-invalido>
	t.Run("module enable invalido", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"module", "enable", "modulo-inexistente", "-dir", tmpDir}, &stdout, &stderr)
		if exitCode == 0 {
			t.Errorf("module enable com módulo inválido deveria falhar, obteve exitCode 0")
		}
		errStr := strings.ToLower(stderr.String())
		if !strings.Contains(errStr, "inválido") && !strings.Contains(errStr, "invalido") && !strings.Contains(errStr, "desconhecido") {
			t.Errorf("stderr esperava mensagem sobre módulo inválido, obteve: %s", stderr.String())
		}
	})

	// 5. subcomandos inválidos ou argumentos ausentes
	t.Run("subcomandos invalidos", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exitCode := run([]string{"module"}, &stdout, &stderr)
		if exitCode == 0 {
			t.Errorf("pipeline-ai module sem argumentos deveria falhar")
		}

		stdout.Reset()
		stderr.Reset()
		exitCode = run([]string{"module", "enable", "-dir", tmpDir}, &stdout, &stderr)
		if exitCode == 0 {
			t.Errorf("module enable sem nome do módulo deveria falhar")
		}

		stdout.Reset()
		stderr.Reset()
		exitCode = run([]string{"module", "comando-invalido"}, &stdout, &stderr)
		if exitCode == 0 {
			t.Errorf("module com comando inválido deveria falhar")
		}
	})
}

func TestCLIRunCommand(t *testing.T) {
	tmpDir := t.TempDir()

	writeMainFile := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir falhou: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write falhou: %v", err)
		}
	}

	cfgContent := `{
  "version": 1,
  "project_type": "api",
  "protected_paths": ["internal/"],
  "editor_agents": ["dev-implementation", "test-writer", "docs-update"]
}`
	writeMainFile(filepath.Join(tmpDir, ".pipeline-ai.json"), cfgContent)
	writeMainFile(filepath.Join(tmpDir, "go.mod"), "module cli_run_test\n\ngo 1.24\n")
	writeMainFile(filepath.Join(tmpDir, "calc.go"), "package calc\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n")
	writeMainFile(filepath.Join(tmpDir, "calc_test.go"), "package calc\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1, 1) != 2 {\n\t\tt.Fatal(\"err\")\n\t}\n}\n")
	writeMainFile(filepath.Join(tmpDir, ".githooks", "pre-push"), "#!/bin/sh\nexit 0\n")
	writeMainFile(filepath.Join(tmpDir, ".git", "HEAD"), "ref: refs/heads/main\n")
	writeMainFile(filepath.Join(tmpDir, ".agents", "hooks.json"), "{}")
	writeMainFile(filepath.Join(tmpDir, ".agents", "skills", "test-writer", "SKILL.md"), "# Test Writer")

	// Abre delivery e cria report válido
	var stdout, stderr bytes.Buffer
	exitCode := run([]string{"delivery", "open", "DEL-017", "-task", "TASK-017", "-dir", tmpDir}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("run(delivery open) falhou: exitCode=%d, stderr:\n%s", exitCode, stderr.String())
	}

	reportContent := `---
delivery_id: DEL-017
task_code: TASK-017
title: Pipeline Runner
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

=== RUN   TestAdd
--- PASS: TestAdd (0.00s)
PASS
ok  	cli_run_test	0.001s

# Code Review

Aprovado sem ressalvas.

# Final Checklist

- [x] Item 1
`
	writeMainFile(filepath.Join(tmpDir, "docs-ai", "deliveries", "DEL-017", "report.md"), reportContent)
	writeMainFile(filepath.Join(tmpDir, "docs-ai", "deliveries", "INDEX.md"), "| DEL-017 | TASK-017 | Pipeline Runner | ready |\n")

	t.Run("modo textual padrao", func(t *testing.T) {
		var out, errOut bytes.Buffer
		code := run([]string{"run", "-dir", tmpDir, "-no-git"}, &out, &errOut)
		if code != 0 {
			t.Errorf("run textual retornou exitCode = %d, esperado 0. stderr: %s", code, errOut.String())
		}
		outStr := out.String()
		if !strings.Contains(strings.ToLower(outStr), "doctor") || !strings.Contains(strings.ToLower(outStr), "sensor") {
			t.Errorf("saída textual não contém steps esperados:\n%s", outStr)
		}
	})

	t.Run("modo json estruturado", func(t *testing.T) {
		var out, errOut bytes.Buffer
		code := run([]string{"run", "-dir", tmpDir, "-no-git", "-json"}, &out, &errOut)
		if code != 0 {
			t.Errorf("run -json retornou exitCode = %d, esperado 0. stderr: %s", code, errOut.String())
		}

		var parsed struct {
			Overall    string `json:"overall"`
			OK         bool   `json:"ok"`
			DeliveryID string `json:"delivery_id"`
			Steps      []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			} `json:"steps"`
		}
		if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
			t.Fatalf("saída não é JSON válido: %v; saída:\n%s", err, out.String())
		}
		if parsed.Overall != "PASS" || !parsed.OK {
			t.Errorf("overall = %q, ok = %v, esperado PASS/true", parsed.Overall, parsed.OK)
		}
		if len(parsed.Steps) == 0 {
			t.Errorf("steps vazio no JSON de saída")
		}
	})

	t.Run("flags -dir, -no-git e -delivery explicita", func(t *testing.T) {
		var out, errOut bytes.Buffer
		code := run([]string{"run", "-dir", tmpDir, "-delivery", "DEL-017", "-no-git", "-json"}, &out, &errOut)
		if code != 0 {
			t.Errorf("run com flag -delivery retornou exitCode = %d, esperado 0. stderr: %s", code, errOut.String())
		}
		var parsed struct {
			DeliveryID string `json:"delivery_id"`
			Overall    string `json:"overall"`
			OK         bool   `json:"ok"`
		}
		if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
			t.Fatalf("JSON inválido: %v", err)
		}
		if parsed.DeliveryID != "DEL-017" {
			t.Errorf("DeliveryID = %q, esperado %q", parsed.DeliveryID, "DEL-017")
		}
	})
}

func TestCLIHandOffValidate(t *testing.T) {
	tmpDir := t.TempDir()

	validHandoff := `DE: architecture-guardrails
PARA: test-writer
DELIVERY_ID: DEL-018
TASK_CODE: TASK-018
STATUS: ok
ESCOPO: internal/handoff, cmd/pipeline-ai, internal/mcp
ARQUIVOS: internal/handoff/handoff.go, internal/handoff/handoff_test.go
CONFIRMADO: go test ./... executado com sucesso e 100% dos testes passando
ASSUNCOES: nenhuma
RISCOS: nenhum
SINAIS:
  SECURITY: nao
  SUPPLY_CHAIN: nao
  MIGRATION: nao
  PERFORMANCE: nao
PROXIMO: test-writer
MOTIVO: implementacao TDD
`
	validFile := filepath.Join(tmpDir, "valid_handoff.txt")
	if err := os.WriteFile(validFile, []byte(validHandoff), 0o644); err != nil {
		t.Fatalf("falha ao criar arquivo de handoff valido: %v", err)
	}

	invalidHandoff := `DE: dev-implementation
PARA: qa-validation
DELIVERY_ID: DEL-018
STATUS: status-invalido
CONFIRMADO: Verifiquei e está ok
PROXIMO: qa-validation
MOTIVO: teste
`
	invalidFile := filepath.Join(tmpDir, "invalid_handoff.txt")
	if err := os.WriteFile(invalidFile, []byte(invalidHandoff), 0o644); err != nil {
		t.Fatalf("falha ao criar arquivo de handoff invalido: %v", err)
	}

	t.Run("handoff valido via arquivo", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"handoff", "validate", validFile}, &stdout, &stderr)
		if code != 0 {
			t.Errorf("handoff validate retornou exitCode = %d, esperado 0; stderr:\n%s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Válido") && !strings.Contains(strings.ToLower(stdout.String()), "ok") {
			t.Errorf("saida de handoff validate nao contem confirmacao de validade:\n%s", stdout.String())
		}
	})

	t.Run("handoff valido com saida json", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"handoff", "validate", "-json", validFile}, &stdout, &stderr)
		if code != 0 {
			t.Errorf("handoff validate -json retornou exitCode = %d, esperado 0; stderr:\n%s", code, stderr.String())
		}

		var res struct {
			Valid    bool           `json:"valid"`
			Errors   []string       `json:"errors"`
			Warnings []string       `json:"warnings"`
			Handoff  map[string]any `json:"handoff"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
			t.Fatalf("saida nao e JSON de ValidationResult valido: %v; raw:\n%s", err, stdout.String())
		}
		if !res.Valid {
			t.Errorf("res.Valid = false no json para handoff valido, erros: %v", res.Errors)
		}
	})

	t.Run("handoff invalido gera exit code 1", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"handoff", "validate", invalidFile}, &stdout, &stderr)
		if code != 1 {
			t.Errorf("handoff validate com handoff invalido retornou exitCode = %d, esperado 1", code)
		}
	})

	t.Run("arquivo inexistente gera erro", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"handoff", "validate", filepath.Join(tmpDir, "inexistente.txt")}, &stdout, &stderr)
		if code == 0 {
			t.Errorf("esperava erro para arquivo inexistente, obteve 0")
		}
	})

	t.Run("handoff inline com barra no escopo valida com sucesso", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"handoff", "validate", validHandoff}, &stdout, &stderr)
		if code != 0 {
			t.Errorf("handoff validate inline retornou exitCode = %d, esperado 0; stderr:\n%s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Válido") && !strings.Contains(strings.ToLower(stdout.String()), "ok") {
			t.Errorf("saida de handoff validate inline nao contem confirmacao de validade:\n%s", stdout.String())
		}
	})
}

func TestCLIMetricsCommand(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("metrics summary em workspace vazio", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"metrics", "summary", "-dir", tmpDir}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("metrics summary falhou: code=%d, stderr:\n%s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Deployment Frequency") {
			t.Errorf("saída esperada contendo 'Deployment Frequency', obteve:\n%s", stdout.String())
		}
	})

	t.Run("metrics summary com flag -json", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"metrics", "summary", "-dir", tmpDir, "-json"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("metrics summary -json falhou: code=%d, stderr:\n%s", code, stderr.String())
		}
		var dora map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &dora); err != nil {
			t.Fatalf("saída não é JSON válido: %v; raw:\n%s", err, stdout.String())
		}
		if _, ok := dora["deployment_frequency"]; !ok {
			t.Errorf("campo deployment_frequency ausente no JSON: %v", dora)
		}
	})

	t.Run("metrics list", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"metrics", "list", "-dir", tmpDir}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("metrics list falhou: code=%d, stderr:\n%s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Entregas registradas") {
			t.Errorf("saída esperada contendo 'Entregas registradas', obteve:\n%s", stdout.String())
		}
	})

	t.Run("metrics list com flag -json", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"metrics", "list", "-dir", tmpDir, "-json"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("metrics list -json falhou: code=%d, stderr:\n%s", code, stderr.String())
		}
		var data map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &data); err != nil {
			t.Fatalf("saída não é JSON válido: %v; raw:\n%s", err, stdout.String())
		}
		if _, ok := data["version"]; !ok {
			t.Errorf("campo version ausente no JSON: %v", data)
		}
	})
}

func TestCLIDashboard(t *testing.T) {
	tmpDir := t.TempDir()

	// Configura ambiente mínimo com INDEX.md
	delivDir := filepath.Join(tmpDir, "docs-ai", "deliveries")
	if err := os.MkdirAll(delivDir, 0755); err != nil {
		t.Fatalf("falha ao criar docs-ai/deliveries: %v", err)
	}
	indexContent := "| ID | Task | Título | Tipo | Data | Status |\n|---|---|---|---|---|---|\n| DEL-001 | TASK-001 | Teste Dashboard | feat | 2026-10-28 | merged |\n"
	if err := os.WriteFile(filepath.Join(delivDir, "INDEX.md"), []byte(indexContent), 0644); err != nil {
		t.Fatalf("falha ao criar INDEX.md: %v", err)
	}

	t.Run("dashboard saída padrão stdout", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"dashboard", "-dir", tmpDir}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("dashboard falhou: code=%d, stderr:\n%s", code, stderr.String())
		}
		out := stdout.String()
		if !strings.Contains(out, "<!DOCTYPE html>") && !strings.Contains(out, "<html") {
			t.Errorf("saída esperada contendo HTML, obteve:\n%s", out)
		}
	})

	t.Run("dashboard com flag -output", func(t *testing.T) {
		outFile := filepath.Join(tmpDir, "relatorio-dashboard.html")
		var stdout, stderr bytes.Buffer
		code := run([]string{"dashboard", "-dir", tmpDir, "-output", outFile}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("dashboard -output falhou: code=%d, stderr:\n%s", code, stderr.String())
		}
		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("falha ao ler arquivo gerado %s: %v", outFile, err)
		}
		if !strings.Contains(string(content), "<html") {
			t.Errorf("arquivo gerado não contém tags html: %s", string(content))
		}
	})

	t.Run("dashboard com diretório inválido para -output", func(t *testing.T) {
		invalidOutput := filepath.Join(tmpDir, "inexistente", "sub", "dashboard.html")
		var stdout, stderr bytes.Buffer
		code := run([]string{"dashboard", "-dir", tmpDir, "-output", invalidOutput}, &stdout, &stderr)
		if code == 0 {
			t.Errorf("esperava código de erro para caminho inválido de saída, obteve 0")
		}
	})
}

func TestCLIPluginCommand(t *testing.T) {
	// Configura mock de bundle válido completo em tempDir
	bundleDir := t.TempDir()
	manifestJSON := `{
		"name": "pipeline-ai",
		"description": "Governança determinística de esteiras com múltiplos agentes em IA",
		"displayName": "Pipeline AI",
		"version": "0.9.0",
		"suggestedPrompts": ["Triage", "Delivery"],
		"repository": "https://github.com/rodrigodosanjosoliveira/pipeline-ai"
	}`
	if err := os.WriteFile(filepath.Join(bundleDir, "plugin.json"), []byte(manifestJSON), 0644); err != nil {
		t.Fatalf("falha ao criar plugin.json: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(bundleDir, "rules"), 0755); err != nil {
		t.Fatalf("falha ao criar pasta rules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "rules", "AGENTS.md"), []byte("# Regras\n"), 0644); err != nil {
		t.Fatalf("falha ao criar rules/AGENTS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "mcp_config.json"), []byte(`{"mcpServers":{}}`), 0644); err != nil {
		t.Fatalf("falha ao criar mcp_config.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "hooks.json"), []byte(`{"hooks":{}}`), 0644); err != nil {
		t.Fatalf("falha ao criar hooks.json: %v", err)
	}
	skillDir := filepath.Join(bundleDir, "skills", "pm-triage")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("falha ao criar pasta skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: pm-triage\n---\n"), 0644); err != nil {
		t.Fatalf("falha ao criar SKILL.md: %v", err)
	}

	t.Run("plugin verify em bundle válido", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"plugin", "verify", "-dir", bundleDir}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("plugin verify falhou: code=%d, stderr:\n%s", code, stderr.String())
		}
		if !strings.Contains(strings.ToLower(stdout.String()), "válido") && !strings.Contains(strings.ToLower(stdout.String()), "ok") {
			t.Errorf("saída esperada confirmando bundle válido, obteve:\n%s", stdout.String())
		}
	})

	t.Run("plugin verify em bundle com arquivo faltando", func(t *testing.T) {
		invalidBundle := t.TempDir()
		var stdout, stderr bytes.Buffer
		code := run([]string{"plugin", "verify", "-dir", invalidBundle}, &stdout, &stderr)
		if code == 0 {
			t.Errorf("plugin verify deveria falhar em diretório vazio sem arquivos de bundle")
		}
	})

	t.Run("plugin install com target válido", func(t *testing.T) {
		targetDir := filepath.Join(t.TempDir(), "installed")
		var stdout, stderr bytes.Buffer
		code := run([]string{"plugin", "install", "-dir", bundleDir, "-target", targetDir}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("plugin install falhou: code=%d, stderr:\n%s", code, stderr.String())
		}
		if _, err := os.Stat(filepath.Join(targetDir, "plugin.json")); err != nil {
			t.Fatalf("plugin.json não encontrado no diretório de destino: %v", err)
		}
	})

	t.Run("plugin install sem flag -target", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"plugin", "install", "-dir", bundleDir}, &stdout, &stderr)
		if code == 0 {
			t.Errorf("plugin install sem -target deveria retornar erro, obteve code=0")
		}
	})

	t.Run("plugin subcomando desconhecido", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"plugin", "inexistente"}, &stdout, &stderr)
		if code == 0 {
			t.Errorf("subcomando desconhecido de plugin deveria retornar erro")
		}
	})
}

func TestCLIEval(t *testing.T) {
	t.Run("eval sem argumentos", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"eval"}, &stdout, &stderr)
		if code == 0 {
			t.Errorf("eval sem argumentos deveria falhar com código != 0")
		}
		out := stdout.String() + stderr.String()
		if !strings.Contains(strings.ToLower(out), "state") && !strings.Contains(strings.ToLower(out), "uso") && !strings.Contains(strings.ToLower(out), "obrigatório") {
			t.Errorf("esperada mensagem de erro de parâmetros, obteve: %s", out)
		}
	})

	t.Run("eval com state ou question vazios", func(t *testing.T) {
		tests := []struct {
			name string
			args []string
		}{
			{name: "state vazio", args: []string{"eval", "-state", "", "-question", "ok?"}},
			{name: "question vazia", args: []string{"eval", "-state", "algo", "-question", ""}},
			{name: "ambos vazios", args: []string{"eval", "-state", "", "-question", ""}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				code := run(tt.args, &stdout, &stderr)
				if code == 0 {
					t.Errorf("esperava código != 0 para parâmetros inválidos em %s", tt.name)
				}
			})
		}
	})

	t.Run("eval quando sensor nao configurado", func(t *testing.T) {
		t.Setenv("PIPELINE_SEMANTIC_ENDPOINT", "")
		t.Setenv("PIPELINE_SEMANTIC_API_KEY", "")
		t.Setenv("JEV_ENDPOINT", "")
		t.Setenv("JEV_API_KEY", "")

		var stdout, stderr bytes.Buffer
		code := run([]string{"eval", "-state", "var a = 1", "-question", "tem segredos?"}, &stdout, &stderr)
		if code == 0 {
			t.Errorf("esperava código != 0 quando sensor não configurado")
		}
		out := stdout.String() + stderr.String()
		if !strings.Contains(strings.ToLower(out), "configur") && !strings.Contains(strings.ToLower(out), "endpoint") {
			t.Errorf("esperava mensagem indicando que o sensor não está configurado, obteve: %s", out)
		}
	})

	t.Run("eval com mock server HTTP sucesso e saida formatada", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"decision": true,
				"confidence": 0.96,
				"reason": "Código sem credenciais hardcoded e bem estruturado.",
				"latency_ms": 35
			}`))
		}))
		defer mockServer.Close()

		var stdout, stderr bytes.Buffer
		args := []string{
			"eval",
			"-endpoint", mockServer.URL,
			"-api-key", "test-token",
			"-state", "const a = 123;",
			"-question", "Há credenciais expostas?",
			"-threshold", "0.8",
		}
		code := run(args, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("eval falhou com código %d, stderr: %s", code, stderr.String())
		}
		out := stdout.String()
		if !strings.Contains(out, "true") && !strings.Contains(strings.ToLower(out), "sim") && !strings.Contains(strings.ToLower(out), "decisão") {
			t.Errorf("saída formatada não contém indicação da decisão: %s", out)
		}
		if !strings.Contains(out, "0.96") && !strings.Contains(out, "96%") {
			t.Errorf("saída formatada não contém confiança: %s", out)
		}
		if !strings.Contains(out, "Código sem credenciais hardcoded") {
			t.Errorf("saída formatada não contém justificativa: %s", out)
		}
	})

	t.Run("eval com flag -json", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"decision": false,
				"confidence": 0.88,
				"reason": "Encontrada vulnerabilidade potencial.",
				"latency_ms": 50
			}`))
		}))
		defer mockServer.Close()

		var stdout, stderr bytes.Buffer
		args := []string{
			"eval",
			"-endpoint", mockServer.URL,
			"-api-key", "test-token",
			"-state", "auth_token = '1234'",
			"-question", "Código seguro?",
			"-json",
		}
		code := run(args, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("eval -json falhou com código %d, stderr: %s", code, stderr.String())
		}
		var res struct {
			Decision   bool    `json:"decision"`
			Confidence float64 `json:"confidence"`
			Reason     string  `json:"reason"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
			t.Fatalf("saída com -json não é JSON válido: %v; saída: %s", err, stdout.String())
		}
		if res.Decision != false {
			t.Errorf("res.Decision esperada false, obteve %v", res.Decision)
		}
		if res.Confidence != 0.88 {
			t.Errorf("res.Confidence esperada 0.88, obteve %v", res.Confidence)
		}
		if res.Reason != "Encontrada vulnerabilidade potencial." {
			t.Errorf("res.Reason esperada 'Encontrada vulnerabilidade potencial.', obteve %q", res.Reason)
		}
	})

	t.Run("eval com -state apontando para arquivo no disco", func(t *testing.T) {
		tmpDir := t.TempDir()
		stateFile := filepath.Join(tmpDir, "state.txt")
		stateContent := "conteúdo lido a partir do arquivo para teste de sensoriamento"
		if err := os.WriteFile(stateFile, []byte(stateContent), 0644); err != nil {
			t.Fatal(err)
		}

		var receivedBody bytes.Buffer
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(&receivedBody, r.Body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"decision": true, "confidence": 0.9, "reason": "ok"}`))
		}))
		defer mockServer.Close()

		var stdout, stderr bytes.Buffer
		args := []string{
			"eval",
			"-endpoint", mockServer.URL,
			"-api-key", "dummy",
			"-state", stateFile,
			"-question", "conteúdo válido?",
		}
		code := run(args, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("eval com arquivo falhou: código %d, stderr: %s", code, stderr.String())
		}
		if !strings.Contains(receivedBody.String(), stateContent) {
			t.Errorf("o servidor mock não recebeu o conteúdo lido do arquivo; payload: %s", receivedBody.String())
		}
	})

	t.Run("eval seguranca SEC-01 token redaction em caso de erro", func(t *testing.T) {
		secretToken := "sk-super-secret-token-eval-998877"
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal server failure with " + secretToken))
		}))
		defer mockServer.Close()

		var stdout, stderr bytes.Buffer
		args := []string{
			"eval",
			"-endpoint", mockServer.URL,
			"-api-key", secretToken,
			"-state", "test",
			"-question", "test?",
		}
		code := run(args, &stdout, &stderr)
		if code == 0 {
			t.Errorf("esperava código de erro para servidor com 500")
		}
		combined := stdout.String() + stderr.String()
		if strings.Contains(combined, secretToken) {
			t.Fatalf("SEC-01 violação: token secreto refletido na saída:\n%s", combined)
		}
	})

	t.Run("eval seguranca SEC-02 limitacao de tamanho do state max 512KB", func(t *testing.T) {
		tmpDir := t.TempDir()
		largeFile := filepath.Join(tmpDir, "large_state.txt")
		largeData := make([]byte, 513*1024)
		for i := range largeData {
			largeData[i] = 'A'
		}
		if err := os.WriteFile(largeFile, largeData, 0644); err != nil {
			t.Fatal(err)
		}

		var stdout, stderr bytes.Buffer
		args := []string{
			"eval",
			"-endpoint", "http://127.0.0.1:9999",
			"-api-key", "token",
			"-state", largeFile,
			"-question", "muito grande?",
		}
		code := run(args, &stdout, &stderr)
		if code == 0 {
			t.Errorf("SEC-02: esperava erro ao ler state que excede 512KB")
		}
		combined := stdout.String() + stderr.String()
		if !strings.Contains(strings.ToLower(combined), "tamanho") && !strings.Contains(strings.ToLower(combined), "limite") && !strings.Contains(strings.ToLower(combined), "large") && !strings.Contains(strings.ToLower(combined), "512") {
			t.Errorf("SEC-02: esperava mensagem indicando limite de tamanho do state excedido, obteve: %s", combined)
		}
	})

	t.Run("eval seguranca SEC-03 timeout mandatorio via context", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(300 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"decision": true, "confidence": 0.9, "reason": "ok"}`))
		}))
		defer mockServer.Close()

		var stdout, stderr bytes.Buffer
		args := []string{
			"eval",
			"-endpoint", mockServer.URL,
			"-api-key", "token",
			"-state", "teste",
			"-question", "timeout?",
			"-timeout", "50ms",
		}
		code := run(args, &stdout, &stderr)
		if code == 0 {
			t.Errorf("SEC-03: esperava código de erro por timeout")
		}
		combined := stdout.String() + stderr.String()
		if !strings.Contains(strings.ToLower(combined), "timeout") && !strings.Contains(strings.ToLower(combined), "context") && !strings.Contains(strings.ToLower(combined), "deadline") {
			t.Errorf("SEC-03: esperava menção a timeout ou cancelamento, obteve: %s", combined)
		}
	})
}
