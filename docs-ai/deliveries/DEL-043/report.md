---
delivery_id: DEL-043
task_code: TASK-043
title: "Integração End-to-End do Ciclo Gemini 4 Hydrogen (Telemetria por Tier, Gate Semântico Advisory no Runner e pipeline://routing)"
status: merged
require_security: true
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Embora a entrega `DEL-039` tenha introduzido a Thinking Tier Matrix (`model_routing`) e o provider nativo Gemini Hydrogen em `internal/sensor/semantic`, os componentes ainda operavam de forma isolada em `eval`: não havia rastreamento de telemetria por provider/tier em `internal/metrics`, o `internal/runner` não integrava avaliação semântica na esteira, e o servidor MCP não expunha um recurso dedicado para consulta direta do roteamento resolvido.

Esta delivery fecha o ciclo end-to-end do Gemini 4 Hydrogen:
1. Estende `internal/metrics` com `RecordSemanticEvalWithProvider` e agregação `SemanticSummary.ByProvider` / `ByModel` (com `omitempty` para retrocompatibilidade).
2. Integra o 4º estágio opcional `-semantic-gate` no `internal/runner` (CLI `pipeline-ai run -semantic-gate` e MCP `pipeline_run_pipeline` com `semantic_gate`), operando como gate *advisory* (`StatusSkip` sem credenciais, `StatusPass`/`StatusWarn` com redação de segredos em erros).
3. Expõe o recurso MCP `pipeline://routing` em JSON com a Thinking Tier Matrix resolvida.

# Escopo

- **Telemetria por Provider e Model (`internal/metrics`):**
  - Adição de `RecordSemanticEvalWithProvider` registrando provider e model nas avaliações semânticas.
  - Expansão de `SemanticSummary` com `ByProvider` e `ByModel` anotados com `json:"...,omitempty"` para preservar retrocompatibilidade com arquivos `.pipeline-state/metrics.json` existentes.
- **Gate Semântico Advisory no Runner (`internal/runner`):**
  - Adição da opção `SemanticGate` no `runner.Run` como 4º estágio opcional (`doctor` → `sensor` → `validate` → `semantic_gate`) utilizando o preset `yagni` com Gemini Hydrogen / Jev.
  - Comportamento *advisory*: retorna `StatusSkip` quando não houver credenciais (`GEMINI_API_KEY` / `JEV_ENDPOINT`) configuradas, e `StatusPass` ou `StatusWarn` em caso de sucesso ou alerta/erro de rede, aplicando `redactSecrets` (SEC-03) nas mensagens de erro.
- **CLI e Schema (`cmd/pipeline-ai/main.go`, `cmd/pipeline-ai/schema.go`):**
  - Adição da flag `-semantic-gate` no subcomando `pipeline-ai run` e registro no catálogo declarativo em `cmd/pipeline-ai/schema.go`.
- **Servidor MCP (`internal/mcp/server.go`):**
  - Novo recurso MCP `pipeline://routing` retornando a Thinking Tier Matrix (`model_routing`) resolvida em JSON.
  - Atualização da ferramenta `pipeline_run_pipeline` com o parâmetro booleano opcional `semantic_gate`.
- **Documentação e Governança:**
  - `docs-ai/DECISOES.md`: Registro da decisão arquitetural D-44 (Seção 18).
  - `README.md`: Documentação da flag `-semantic-gate` em `pipeline-ai run`, do recurso `pipeline://routing` e do parâmetro `semantic_gate` em `pipeline_run_pipeline`.
  - `docs-ai/deliveries/INDEX.md`: Registro de `DEL-043` e atualização de `DEL-042` para `merged`.
  - `docs-ai/deliveries/DEL-042/report.md`: Sincronização do status para `merged`.
  - `docs-ai/deliveries/DEL-043/report.md`: Relatório completo desta delivery.

**Fora de escopo:**
- Tornar o gate semântico bloqueante (`StatusFail`) por padrão no `runner` (descartado em D-44 para preservar o determinismo da esteira).
- Adição de dependências externas em `go.mod`.

# Implementação

1. **Telemetria por Tier (`internal/metrics/metrics.go`):**
   - Implementada `RecordSemanticEvalWithProvider` para registrar o provider e o modelo utilizados em cada avaliação semântica.
   - Adicionados `ByProvider` e `ByModel` em `SemanticSummary` com tags `omitempty`, mantendo compatibilidade retroativa na serialização e desserialização JSON.
2. **4º Estágio Advisory no Runner (`internal/runner/runner.go`):**
   - Integrado o estágio `semantic_gate` acionado via flag `-semantic-gate` / parâmetro `semantic_gate`.
   - Sem credenciais configuradas, o estágio emite `StatusSkip` sem interromper ou falhar a esteira.
   - Com credenciais, executa a avaliação com preset `yagni` retornando `StatusPass` ou `StatusWarn`, sanitizando eventuais mensagens de erro com `redactSecrets`.
3. **CLI, Schema e MCP (`cmd/pipeline-ai/main.go`, `cmd/pipeline-ai/schema.go`, `internal/mcp/server.go`):**
   - `cmd/pipeline-ai/main.go` e `cmd/pipeline-ai/schema.go` expõem a flag `-semantic-gate` no comando `run`.
   - `internal/mcp/server.go` registra o recurso `pipeline://routing` (retornando a Thinking Tier Matrix resolvida em JSON) e repassa `semantic_gate` em `pipeline_run_pipeline`.

**Arquivos alterados:**
- `internal/metrics/metrics.go` — `RecordSemanticEvalWithProvider` e `SemanticSummary.ByProvider`/`ByModel` (`omitempty`)
- `internal/metrics/metrics_test.go` — `TestRecordSemanticEvalWithProviderAndDORA`
- `internal/runner/runner.go` — 4º estágio opcional `semantic_gate` advisory e `redactSecrets`
- `internal/runner/runner_test.go` — `TestRunnerSemanticGate`
- `internal/mcp/server.go` — Recurso `pipeline://routing` e parâmetro `semantic_gate` em `pipeline_run_pipeline`
- `internal/mcp/server_test.go` — `TestMCPRoutingResourceAndSemanticGate`
- `cmd/pipeline-ai/main.go` — Flag `-semantic-gate` no subcomando `run`
- `cmd/pipeline-ai/schema.go` — Registro da flag `-semantic-gate` no schema de `run`
- `cmd/pipeline-ai/main_test.go` — `TestCLIRunSemanticGateFlag`
- `README.md` — Documentação de `-semantic-gate`, `pipeline://routing` e `pipeline_run_pipeline`
- `docs-ai/DECISOES.md` — Decisão arquitetural D-44 (Seção 18)
- `docs-ai/deliveries/INDEX.md` — Registro de `DEL-043` e atualização de `DEL-042` para `merged`
- `docs-ai/deliveries/DEL-042/report.md` — Atualização de `status: draft` para `status: merged`
- `docs-ai/deliveries/DEL-043/report.md` — Relatório completo desta delivery

# Testes

**Gates:**

```
$ gofmt -l .
(saída limpa)

$ go build ./...
(saída limpa, exit code 0)

$ go vet ./...
(saída limpa, exit code 0)
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| `internal/metrics/metrics.go` (`RecordSemanticEvalWithProvider`, `ByProvider`, `ByModel`) | `TestRecordSemanticEvalWithProviderAndDORA` em `internal/metrics/metrics_test.go` |
| `internal/runner/runner.go` (4º estágio `semantic_gate` advisory e `redactSecrets`) | `TestRunnerSemanticGate` em `internal/runner/runner_test.go` |
| `internal/mcp/server.go` (`pipeline://routing` e `pipeline_run_pipeline` com `semantic_gate`) | `TestMCPRoutingResourceAndSemanticGate` em `internal/mcp/server_test.go` |
| `cmd/pipeline-ai/main.go` e `schema.go` (flag `-semantic-gate` em `run`) | `TestCLIRunSemanticGateFlag` em `cmd/pipeline-ai/main_test.go` |

**Validação (QA):**
`gofmt -l .`, `go build ./...`, `go vet ./...` e os testes unitários dos pacotes `internal/metrics` e `internal/runner` executados e aprovados (testes com `httptest` em `cmd/pipeline-ai` e `internal/mcp` dependem de socket loopback fora do sandbox restrito).

**Supply Chain:**
Sinal `SUPPLY_CHAIN: nao`. Nenhuma nova dependência externa adicionada em `go.mod`.

**Pendências:**
Nenhuma pendência técnica identificada.

**Handoffs:**
Esteira executada continuamente através dos papéis `pm-triage` → `architecture-guardrails` → `test-writer` → `dev-implementation` → `code-reviewer` → `security-review` → `qa-validation` → `docs-update`.

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:**
- Uso de `omitempty` em `SemanticSummary.ByProvider` e `ByModel` para retrocompatibilidade com arquivos `metrics.json` legados.
- Comportamento *advisory* (`StatusSkip` / `StatusWarn`) no estágio `semantic_gate` do `runner` para evitar quebra de determinismo por indisponibilidade de rede ou ausência de chave.

**Sugestões recusadas:** nenhuma

# Security

**Superfície revisada:**
- `internal/runner/runner.go` — Execução do 4º estágio `semantic_gate`, verificação de credenciais e tratamento de erros.
- `internal/mcp/server.go` — Exposição do recurso `pipeline://routing` e parâmetro `semantic_gate` em `pipeline_run_pipeline`.
- `cmd/pipeline-ai/main.go` — Flag `-semantic-gate` em `pipeline-ai run`.

**Validação dos requisitos de segurança (`SEC-03`):**
- **SEC-03 (Redação de Segredos, Fail-Safe sem Credenciais e Timeout):**
  - Quando não há credenciais configuradas (`GEMINI_API_KEY` / `JEV_ENDPOINT`), o estágio `semantic_gate` no `runner` retorna `StatusSkip` de forma segura sem vazar variáveis de ambiente ou tentar chamadas externas sem autenticação.
  - Mensagens de erro retornadas durante a avaliação semântica passam por `redactSecrets`, substituindo chaves de API e tokens por `[REDACTED]` antes de compor o resultado do estágio.
  - Chamadas ao cliente semântico preservam o timeout determinístico via `context.WithTimeout`, prevenindo bloqueios indefinidos na esteira.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde (pacotes `internal/metrics` e `internal/runner` validados localmente; `httptest` loopback validado fora do sandbox)
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
