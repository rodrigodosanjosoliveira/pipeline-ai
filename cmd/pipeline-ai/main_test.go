package main

import (
	"bytes"
	"encoding/json"
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
			wantStdout: "pipeline-ai v0.6.0",
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
