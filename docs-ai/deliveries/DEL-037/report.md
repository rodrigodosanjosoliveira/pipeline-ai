---
delivery_id: DEL-037
task_code: TASK-037
title: "Bounded Output, Smart Test Filtering e marcação de resultados parciais no sensor e runner (CLI e MCP)"
status: merged
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Em projetos Go com dezenas ou centenas de testes, a execução de `go test -v ./...` produz dezenas de kilobytes de logs repetitivos (`=== RUN`, `--- PASS`), degradando a eficiência da janela de contexto dos agentes de IA (LLMs), elevando o consumo de tokens e podendo estourar buffers de ferramentas CLI e MCP.

Seguindo o padrão de *Dual-Audience CLI Design* (ferramentas legíveis por humanos e econômicas/estruturadas para agentes de IA), esta delivery implementa **Bounded Output**, **Smart Test Filtering** e **Marcação Explícita de Resultados Parciais** (`truncated`, `total_bytes`, `omitted_bytes`, `failed_tests`) no motor de sensores (`internal/sensor`), no Pipeline Runner (`internal/runner`), na CLI unificada (`cmd/pipeline-ai`) e no servidor MCP (`internal/mcp`) (D-39).

# Escopo

- **`internal/sensor/sensor.go` e `internal/sensor/sensor_test.go`:**
  - Constante `DefaultMaxOutputBytes = 8192` (8 KB) e struct `Options{MaxOutputBytes int, VerboseOutput bool}`.
  - Expansão de `CheckResult` com metadados `Truncated bool` (`json:"truncated,omitempty"`), `TotalBytes int` (`json:"total_bytes,omitempty"`), `OmittedBytes int` (`json:"omitted_bytes,omitempty"`) e `FailedTests []string` (`json:"failed_tests,omitempty"`).
  - Funções puras `BoundOutput(raw string, maxBytes int)` e `BoundTestOutput(raw string, opts Options)` com *Smart Test Filtering* (suprimindo blocos `=== RUN` / `--- PASS` e `--- SKIP` quando `VerboseOutput == false`, preservando integralmente blocos `--- FAIL`, erros de compilação, panics e sumários de pacotes `ok`/`FAIL`), extração determinística sem duplicatas de `failed_tests` e truncamento seguro em fronteira de runa UTF-8 (`utf8.RuneStart`).
  - `RunWithOptions(dir string, opts Options)` e manutenção da assinatura `Run(dir string)` delegando com `DefaultMaxOutputBytes`.
  - Testes unitários table-driven e alvo de fuzzing `FuzzBoundTestOutput`.
- **`internal/runner/runner.go` e `internal/runner/runner_test.go`:**
  - Adição de `MaxOutputBytes int` e `VerboseOutput bool` em `runner.Options`, repassados para `sensor.RunWithOptions`.
- **`cmd/pipeline-ai/main.go` e `cmd/pipeline-ai/main_test.go`:**
  - Adição das flags `-max-output-bytes` (default `8192`) e `-verbose-output` (default `false`) nos subcomandos `pipeline-ai sensor` e `pipeline-ai run`.
  - Exibição dos testes que falharam e aviso de saída truncada no modo texto.
- **`internal/mcp/server.go` e `internal/mcp/server_test.go`:**
  - Suporte aos argumentos opcionais `max_output_bytes` e `verbose_output` nas tools `pipeline_run_sensors` e `pipeline_run_pipeline`.
- **Documentação e Governança:**
  - `README.md`, `docs-ai/DECISOES.md` (D-39), `docs-ai/deliveries/INDEX.md` e `docs-ai/deliveries/DEL-037/report.md`.

**Fora de escopo:**
- Alteração na lógica de execução do `go test -v ./...` para contagem de `tests_run` (a contagem sempre ocorre sobre a saída bruta completa antes da filtragem/truncamento).
- Adição de dependências externas em `go.mod`.

# Implementação

1. **Motor de Sensores (`internal/sensor/sensor.go`):**
   - Implementada `BoundOutput(raw string, maxBytes int)` com recuo seguro até o início de runa UTF-8 (`utf8.RuneStart`) quando o corte ocorre no meio de um caractere multi-byte, anexando rodapé `\n... [TRUNCATED: N bytes omitted]`.
   - Implementada `BoundTestOutput(raw string, opts Options)` que varre a saída do `go test -v`, identifica blocos de testes por `=== RUN`, descarta blocos finalizados com `--- PASS:` ou `--- SKIP:` quando `!opts.VerboseOutput`, retém integralmente blocos `--- FAIL:`, linhas de pacote (`ok`, `FAIL`) e erros de compilação/panics, extrai os nomes únicos em `failedTests` e aplica `BoundOutput` ao resultado filtrado.
   - Adicionada `RunWithOptions(dir string, opts Options)` preservando `countTestsRun(out)` na saída bruta original para garantir que a salvaguarda contra zero testes (`tests_run == 0`) permaneça 100% exata.
2. **Pipeline Runner (`internal/runner/runner.go`):**
   - Expandida `runner.Options` com `MaxOutputBytes` e `VerboseOutput`, encaminhando-os ao estágio `sensor.RunWithOptions`.
3. **CLI Unificada (`cmd/pipeline-ai/main.go`) e Servidor MCP (`internal/mcp/server.go`):**
   - Adicionadas flags `-max-output-bytes` e `-verbose-output` em `runSensor` e `runPipeline`.
   - Atualizadas as definições JSON Schema e handlers de `pipeline_run_sensors` e `pipeline_run_pipeline` no servidor MCP para aceitar `max_output_bytes` e `verbose_output`.
4. **Documentação (`README.md` e `docs-ai/DECISOES.md`):**
   - Documentada a decisão arquitetural `D-39` em `docs-ai/DECISOES.md` e atualizadas as tabelas e exemplos de uso em `README.md`.

**Arquivos alterados:**
- `internal/sensor/sensor.go` — Implementação de `Options`, `BoundOutput`, `BoundTestOutput` e `RunWithOptions`
- `internal/sensor/sensor_test.go` — Testes unitários table-driven e alvo de fuzzing `FuzzBoundTestOutput`
- `internal/runner/runner.go` — Propagação de `MaxOutputBytes` e `VerboseOutput` para o sensor
- `internal/runner/runner_test.go` — Teste de propagação `TestRun_PropagatesMaxOutputBytes`
- `cmd/pipeline-ai/main.go` — Flags `-max-output-bytes` e `-verbose-output` em `sensor` e `run`
- `cmd/pipeline-ai/main_test.go` — Testes de integração CLI para flags de bounded output
- `internal/mcp/server.go` — Parâmetros `max_output_bytes` e `verbose_output` em `pipeline_run_sensors` e `pipeline_run_pipeline`
- `internal/mcp/server_test.go` — Testes unitários MCP validando `max_output_bytes` e `verbose_output`
- `README.md` — Documentação de Bounded Output e Smart Test Filtering na CLI e MCP
- `docs-ai/DECISOES.md` — Registro da decisão D-39 (Seção 14)
- `docs-ai/deliveries/INDEX.md` — Registro de `DEL-037` e atualização de `DEL-036`
- `docs-ai/deliveries/DEL-036/report.md` — Sincronização de status para `merged`
- `docs-ai/deliveries/DEL-037/report.md` — Relatório desta delivery

# Testes

**Gates:**

```
$ gofmt -l .
(saída limpa)

$ go build ./...
(saída limpa, exit code 0)

$ go vet ./...
(saída limpa, exit code 0)

$ go test -race ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	3.439s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	1.471s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-init	1.747s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.855s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	1.281s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	1.930s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/debt	1.341s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	2.111s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	1.464s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/hook	1.123s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	2.968s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	1.572s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	1.648s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	5.605s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	2.299s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	4.576s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	2.142s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	1.366s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	2.497s

$ go test -fuzz=FuzzBoundTestOutput -fuzztime=5s ./internal/sensor
fuzz: elapsed: 0s, gathering baseline coverage: 0/4 completed
fuzz: elapsed: 3s, execs: 486259 (162077/sec), new interesting: 14 (total: 18)
fuzz: elapsed: 5s, execs: 789598 (157890/sec), new interesting: 14 (total: 18)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	5.224s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| `internal/sensor/sensor.go` (`BoundOutput` e truncamento UTF-8) | `TestBoundOutput` (`internal/sensor/sensor_test.go`) |
| `internal/sensor/sensor.go` (`BoundTestOutput` e Smart Test Filtering) | `TestBoundTestOutput` (`internal/sensor/sensor_test.go`) |
| `internal/sensor/sensor.go` (`RunWithOptions`, `failed_tests` e `tests_run`) | `TestRunWithOptions_BoundedAndFailedTests` (`internal/sensor/sensor_test.go`) |
| `internal/sensor/sensor.go` (Fuzzing de `BoundTestOutput` e `utf8.ValidString`) | `FuzzBoundTestOutput` (`internal/sensor/sensor_test.go`) |
| `internal/runner/runner.go` (`Options.MaxOutputBytes` e `VerboseOutput`) | `TestRun_PropagatesMaxOutputBytes` (`internal/runner/runner_test.go`) |
| `cmd/pipeline-ai/main.go` (CLI `sensor` e `run` com `-max-output-bytes` e `-verbose-output`) | `TestCLISensorJSON`, `TestCLIRunCommand` (`cmd/pipeline-ai/main_test.go`) |
| `internal/mcp/server.go` (MCP `pipeline_run_sensors` e `pipeline_run_pipeline`) | `TestMCPToolsRunSensors`, `TestMCPToolsRunPipeline` (`internal/mcp/server_test.go`) |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** preservação integral da contagem `countTestsRun` sobre a saída bruta antes de aplicar `BoundTestOutput` e recuo seguro em fronteira de runa (`utf8.RuneStart`) validado por fuzz testing.

**Sugestões recusadas:** nenhuma

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
