---
delivery_id: DEL-017
task_code: TASK-017
title: "Pipeline Runner CLI (pipeline-ai run) e MCP Tool para orquestração de gates em lote"
type: feat
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Após a introdução dos sensores com saída estruturada (DEL-013) e da regra formal de orquestração autônoma contínua (DEL-016), agentes de IA e desenvolvedores ainda precisavam disparar comandos separados para diagnosticar o ambiente (`pipeline-ai doctor`), rodar a suíte de sensores (`pipeline-ai sensor`) e validar o relatório da entrega (`pipeline-validate`).

Esta entrega introduz o **Pipeline Runner**:
- Um motor integrado (`internal/runner`) que executa sequencialmente todos os gates locais determinísticos (`doctor` ➔ `sensor` ➔ `validate`).
- Novo subcomando CLI `pipeline-ai run` com suporte a execução textual interativa ou saída estruturada em JSON (`-json`).
- Nova ferramenta MCP `pipeline_run_pipeline` permitindo a execução remota de toda a esteira por agentes MCP.
- Descoberta automática da delivery ativa e relatório associado via `state.Get`. Se não houver delivery ativa ou relatório indicado, o gate de validação é marcado como `SKIP` sem quebrar o pipeline.

# Escopo

- Novo pacote `internal/runner`:
  - Tipos: `Status`, `StepResult`, `Result`, `Options`.
  - Função `Run(opts Options) (*Result, error)`.
  - Orquestração dos gates:
    - Gate 1: `doctor` (diagnóstico do ambiente, toolchain, git e configuração).
    - Gate 2: `sensor` (formatação `fmt`, análise estática `vet`, e suíte de testes com autoteste negativo).
    - Gate 3: `validate` (validação de `report.md`, checklists, índices e superfícies git).
- CLI `pipeline-ai run` em `cmd/pipeline-ai/main.go`:
  - Flags: `-dir`, `-delivery`, `-report`, `-no-git`, `-base`, `-json`.
  - Visualização formatada em texto ou JSON estruturado indentado.
  - Códigos de saída: `0` para aprovação de todos os gates, `1` para falhas.
- Ferramenta MCP `pipeline_run_pipeline` em `internal/mcp/server.go`:
  - Registro em `tools/list` e manipulador em `tools/call`.
- Testes automatizados unitários, CLI e MCP (404 testes passando).

**Fora de escopo:**
- Pushes ou interações com repositórios remotos (o runner é estritamente local e determinístico).
- Dependências externas de terceiros (100% Go 1.24+ standard library).

# Implementação

1. **`internal/runner/runner.go`:**
   - Implementado o ciclo sequencial de execução dos gates.
   - Resolução inteligente do relatório a validar (argumento `-report`, argumento `-delivery`, ou detecção da delivery ativa em `.pipeline-state/active-delivery.json`).
   - Se nenhuma delivery for detectada, marca o step de validação como `SKIP`.
2. **`cmd/pipeline-ai/main.go`:**
   - Adicionada a função `runRun` processando as flags e renderizando os resultados no terminal ou serializando JSON estruturado.
   - Atualizado o roteador `runWithStdin` e as instruções de ajuda em `printUsage`.
3. **`internal/mcp/server.go`:**
   - Registrada a tool `pipeline_run_pipeline` no catálogo MCP e configurado o tratamento da requisição em `handleToolsCall`.

**Arquivos alterados:**
- `internal/runner/runner.go` — Implementação do pipeline runner integrado
- `internal/runner/runner_test.go` — Testes unitários do runner
- `cmd/pipeline-ai/main.go` — Subcomando run
- `cmd/pipeline-ai/main_test.go` — Testes do subcomando run
- `internal/mcp/server.go` — Tool MCP pipeline_run_pipeline
- `internal/mcp/server_test.go` — Testes da tool MCP
- `docs-ai/deliveries/INDEX.md` — Registro de DEL-017
- `docs-ai/deliveries/DEL-017/report.md` — Relatório de entrega da delivery

# Testes

**Gates:**

```
$ gofmt -l .
(nenhum arquivo fora do padrão)

$ go build ./...
(compilação com sucesso)

$ go vet ./...
(análise estática sem avisos)

$ go test -count=1 ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	4.151s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.149s
?   	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-init	[no test files]
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.980s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.277s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.109s
?   	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/hook	[no test files]
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	2.715s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.029s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	6.227s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	1.125s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	3.819s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.165s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.696s
PASS
```

**Execução Integrada (`pipeline-ai run`):**

```
=== pipeline-ai runner ===
Delivery: DEL-017

 [PASS] doctor     : Ambiente validado com sucesso
 [PASS] sensor     : sensores aprovados (404 testes executados)
 [PASS] validate   : validação aprovada (0 erros, 0 avisos)

Status geral: PASS
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Orquestração dos gates e skips condicionais | `TestRunner_SuccessWithActiveDelivery`, `TestRunner_NoActiveDelivery_SensorsAndDoctorOnly`, `TestRunner_SensorFailure`, `TestRunner_ValidateFailure` em `internal/runner/runner_test.go` |
| CLI `pipeline-ai run` (textual e `-json`) | `TestCLIRunCommand` em `cmd/pipeline-ai/main_test.go` |
| Tool MCP `pipeline_run_pipeline` | `TestMCPToolsRunPipeline` em `internal/mcp/server_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Detecção automática de delivery a partir do diretório pai (`DEL-NNN`) caso um caminho relativo de relatório seja informado sem `-delivery`.

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
