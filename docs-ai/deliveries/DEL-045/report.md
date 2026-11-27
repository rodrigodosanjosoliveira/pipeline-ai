---
delivery_id: DEL-045
task_code: TASK-045
title: "Sensor Nativo de Supply Chain (govulncheck) com StatusSkip e Bounded Output"
status: merged
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

A auditoria de vulnerabilidades de supply chain via `govulncheck ./...` era executada apenas como gate externo em `Taskfile.yml` / CI, sem integração estruturada nos sensores (`pipeline-ai sensor`), no pipeline runner (`pipeline-ai run`) nem nas ferramentas do servidor MCP (`pipeline_run_sensors` e `pipeline_run_pipeline`).

Esta delivery implementa o 4º sensor nativo `"vuln"` (`D-45`), opt-in via `-vuln` na CLI e `include_vuln` no MCP, aplicando `BoundOutput` à saída, `StatusSkip` gracioso quando `govulncheck` não estiver instalado no ambiente e suporte a `VulnBinary` injetável para testes determinísticos offline em 100% Go stdlib.

# Escopo

- **Sensor de Supply Chain (`internal/sensor/sensor.go`, `internal/sensor/sensor_test.go`):**
  - Adição do status `StatusSkip`, das opções `IncludeVuln` e `VulnBinary` em `Options`, e da execução do 4º sensor `"vuln"` (`govulncheck ./...`) com `BoundOutput` aplicado à saída.
  - Retorno gracioso `StatusSkip` quando o binário `govulncheck` não estiver instalado (sem falhar o `Overall`), e `StatusFail` quando vulnerabilidades forem reportadas.
  - Testes determinísticos offline utilizando `VulnBinary` injetável (binário ausente → `StatusSkip`, execução limpa → `StatusPass`, falha → `StatusFail`, truncamento via `BoundOutput`).
- **Integração no Runner (`internal/runner/runner.go`, `internal/runner/runner_test.go`):**
  - Propagação de `IncludeVuln` e `VulnBinary` nas opções do runner para o estágio `Sensor`.
- **CLI e Catálogo de Schema (`cmd/pipeline-ai/main.go`, `cmd/pipeline-ai/schema.go`, `cmd/pipeline-ai/main_test.go`):**
  - Adição da flag `-vuln` nos subcomandos `pipeline-ai sensor` e `pipeline-ai run`.
  - Registro declarativo da flag `vuln` no catálogo de introspecção `cmd/pipeline-ai/schema.go`.
- **Servidor MCP (`internal/mcp/server.go`, `internal/mcp/server_test.go`):**
  - Adição do parâmetro booleano `include_vuln` nas tools `pipeline_run_sensors` e `pipeline_run_pipeline`.
- **Documentação e Governança:**
  - `docs-ai/DECISOES.md`: Registro da decisão arquitetural D-45 (Seção 19).
  - `README.md`: Documentação da flag `-vuln` em `pipeline-ai sensor` e `pipeline-ai run`, e do parâmetro `include_vuln` nas tools MCP `pipeline_run_sensors` e `pipeline_run_pipeline`.
  - `docs-ai/deliveries/DEL-044/report.md`: Atualização de `status: draft` para `status: merged`.
  - `docs-ai/deliveries/INDEX.md`: Atualização de `DEL-044` para `merged` e registro de `DEL-045`.
  - `docs-ai/deliveries/DEL-045/report.md`: Relatório completo desta delivery.

**Fora de escopo:**
- Tornar `govulncheck` obrigatório/bloqueante sem opt-in em ambientes offline ou sem o binário instalado.
- Adição de dependências externas ao `go.mod`.

# Implementação

1. **Sensor (`internal/sensor/sensor.go`):**
   - Implementação do 4º sensor `"vuln"` acionado quando `opts.IncludeVuln` é verdadeiro, resolvendo o binário via `opts.VulnBinary` (padrão `"govulncheck"`), retornando `StatusSkip` quando não encontrado no `PATH` e aplicando `BoundOutput` à saída.
2. **Runner (`internal/runner/runner.go`):**
   - Repasse de `IncludeVuln` e `VulnBinary` nas opções passadas ao `sensor.RunWithOptions`.
3. **CLI & Schema (`cmd/pipeline-ai/main.go`, `cmd/pipeline-ai/schema.go`):**
   - Adição da flag `-vuln` em `sensor` e `run`, e exposição no schema declarativo em `schema.go`.
4. **Servidor MCP (`internal/mcp/server.go`):**
   - Adição do argumento `include_vuln` no schema e handlers de `pipeline_run_sensors` e `pipeline_run_pipeline`.

**Arquivos alterados:**
- `internal/sensor/sensor.go` — Sensor `"vuln"` (`govulncheck ./...`), `StatusSkip`, `IncludeVuln`, `VulnBinary` e `BoundOutput`
- `internal/sensor/sensor_test.go` — Testes determinísticos offline do sensor `"vuln"`
- `internal/runner/runner.go` — Propagação de `IncludeVuln` e `VulnBinary` para o estágio `Sensor`
- `internal/runner/runner_test.go` — Cobertura de `IncludeVuln` no runner
- `cmd/pipeline-ai/main.go` — Flag `-vuln` em `pipeline-ai sensor` e `pipeline-ai run`
- `cmd/pipeline-ai/schema.go` — Documentação declarativa da flag `vuln` no catálogo de introspecção
- `cmd/pipeline-ai/main_test.go` — Testes da flag `-vuln` na CLI e schema
- `internal/mcp/server.go` — Parâmetro `include_vuln` em `pipeline_run_sensors` e `pipeline_run_pipeline`
- `internal/mcp/server_test.go` — Testes de `include_vuln` nas tools MCP
- `README.md` — Documentação de `-vuln` e `include_vuln`
- `docs-ai/DECISOES.md` — Registro da decisão arquitetural D-45 (Seção 19)
- `docs-ai/deliveries/INDEX.md` — Atualização de `DEL-044` para `merged` e registro de `DEL-045`
- `docs-ai/deliveries/DEL-044/report.md` — Atualização de `status: draft` para `status: merged`
- `docs-ai/deliveries/DEL-045/report.md` — Relatório completo desta delivery

# Testes

**Gates:**

```text
$ gofmt -l .
(saída limpa)

$ go build ./...
(saída limpa, exit code 0)

$ go vet ./...
(saída limpa, exit code 0)

$ go test -race ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor
PASS
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Sensor `"vuln"` (`StatusSkip`, `StatusPass`, `StatusFail`, `BoundOutput`) | `internal/sensor/sensor_test.go` |
| Propagação de `IncludeVuln` no `runner` | `internal/runner/runner_test.go` |
| Flags `-vuln` e schema em `cmd/pipeline-ai` | `cmd/pipeline-ai/main_test.go` |
| Parâmetro `include_vuln` em `pipeline_run_sensors` e `pipeline_run_pipeline` | `internal/mcp/server_test.go` |
| Integridade da delivery e registro no `INDEX.md` | `go run ./cmd/pipeline-validate docs-ai/deliveries/DEL-045/report.md` |

**Validação (QA):**
`gofmt -l .`, `go build ./...`, `go vet ./...` e `go test -race ./...` aprovados.

**Segurança:**
Sinal `SECURITY: nao`. Nenhuma alteração em criptografia, segredos ou controle de acesso.

**Supply Chain:**
Sinal `SUPPLY_CHAIN: nao`. Nenhuma nova dependência externa adicionada em `go.mod`.

**Pendências:**
Nenhuma pendência técnica identificada.

**Handoffs:**
Esteira executada continuamente através dos papéis `pm-triage` → `architecture-guardrails` → `test-writer` → `dev-implementation` → `code-reviewer` → `qa-validation` → `docs-update`.

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** opt-in explícito via `-vuln` / `include_vuln`, `StatusSkip` gracioso na ausência do binário `govulncheck` e injeção de `VulnBinary` para testes determinísticos offline em Go stdlib pura.

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
