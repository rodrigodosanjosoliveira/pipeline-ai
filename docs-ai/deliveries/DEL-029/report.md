---
delivery_id: DEL-029
task_code: TASK-029
title: "Expansão do sensor semântico Jev (EvalChoice e EvalScore), CLI eval -mode, MCP pipeline_eval_semantic e telemetria semântica"
status: draft
require_security: true
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

As entregas anteriores (`DEL-025`, `DEL-026` e `DEL-027`) introduziram o pacote `internal/sensor/semantic` com avaliação binária calibrada (`EvalNoul` em `/v1/noul`), os structs `ChoiceDecision` e `ScoreDecision` em `types.go`, o subcomando CLI `pipeline-ai eval` e a ferramenta MCP `pipeline_eval_semantic`.

Para completar o ciclo de decisões calibradas estilo Jev ("System One"), o pipeline precisava operacionalizar:
1. **Roteamento de múltipla escolha (`EvalChoice`)**: permitindo que agentes e automações escolham uma opção dentre um conjunto fechado de `choices` com verificação anti-alucinação e limiar de confiança.
2. **Pontuação contínua (`EvalScore`)**: permitindo avaliação em escala numérica (`0..max_score`) com validação de fronteiras e calibração normalizada.
3. **Telemetria semântica (`internal/metrics`)**: rastreando o volume, a taxa de aprovação, a confiança média e a distribuição por modo (`noul`, `choice`, `score`) das avaliações semânticas ao lado das métricas DORA.

# Escopo

- Implementação de `ChoiceInput`, `ScoreInput`, `ErrInvalidChoice`, `ErrInvalidScore`, `Client.EvalChoice`, `Client.EvalScore`, `ParseChoiceResponse` e `ParseScoreResponse` em `internal/sensor/semantic`.
- Alvos de fuzzing `FuzzParseChoiceResponse` e `FuzzParseScoreResponse` em `internal/sensor/semantic/fuzz_test.go`.
- Expansão do subcomando CLI `pipeline-ai eval` com as flags `-mode` (`noul` | `choice` | `score`, default `noul`), `-choices` e `-max-score`.
- Expansão da ferramenta MCP `pipeline_eval_semantic` com `mode`, `choices` e `max_score`.
- Registro e agregação de telemetria semântica (`SemanticEvalEvent`, `SemanticSummary`, `RecordSemanticEval`) em `internal/metrics` integrado ao resumo DORA (`pipeline-ai metrics summary`).
- Atualização da documentação (`README.md`, `docs-ai/DECISOES.md` com `D-35`) e sincronização de `DEL-028` para `merged`.

**Fora de escopo:**
- Adição de dependências externas em `go.mod` (100% Go 1.24+ standard library).
- Mudanças quebradoras no comportamento default (`noul`) da CLI ou MCP.

# Implementação

1. **Sensor Semântico (`internal/sensor/semantic/types.go` e `client.go`):**
   - Adicionados os erros sentinela `ErrInvalidChoice` e `ErrInvalidScore`, bem como os tipos `ChoiceInput` e `ScoreInput`.
   - Extraído o helper HTTP compartilhado `Client.doPost` concentrando `context.WithTimeout`, `io.LimitReader` (`MaxBodyBytes` 10MB) e sanitização de `APIKey` (`[REDACTED]`).
   - Implementados `EvalChoice` (`/v1/choice`) e `EvalScore` (`/v1/score`), além das funções de parsing `ParseChoiceResponse` e `ParseScoreResponse`.
2. **Telemetria Semântica (`internal/metrics/metrics.go`):**
   - Adicionados `SemanticEvalEvent` e `SemanticSummary` (contendo `TotalEvals`, `ApprovedRate`, `AvgConfidence` e `ByMode`).
   - Adicionado `Store.RecordSemanticEval` com persistência atômica e integração em `CalculateDORA`.
3. **CLI `pipeline-ai eval` e `metrics summary` (`cmd/pipeline-ai/main.go`):**
   - Adicionadas as flags `-mode`, `-choices` e `-max-score` em `runEval`, preservando `-mode noul` como default, limite de 512KB para `state` e redação de credenciais.
   - Exibição da seção de telemetria semântica em `pipeline-ai metrics summary` quando houver eventos registrados.
4. **Servidor MCP (`internal/mcp/server.go`):**
   - Atualizado o schema e o handler `toolEvalSemantic` de `pipeline_eval_semantic` para aceitar `mode`, `choices` e `max_score`.

**Arquivos alterados:**
- `internal/sensor/semantic/types.go` — Novos tipos `ChoiceInput`, `ScoreInput` e erros `ErrInvalidChoice`, `ErrInvalidScore`
- `internal/sensor/semantic/client.go` — Implementação de `EvalChoice`, `EvalScore`, `ParseChoiceResponse`, `ParseScoreResponse` e helper `doPost`
- `internal/sensor/semantic/semantic_test.go` — Testes unitários e adversariais para `EvalChoice` e `EvalScore`
- `internal/sensor/semantic/fuzz_test.go` — Alvos de fuzzing `FuzzParseChoiceResponse` e `FuzzParseScoreResponse`
- `internal/metrics/metrics.go` — Telemetria semântica (`SemanticEvalEvent`, `SemanticSummary`, `RecordSemanticEval`)
- `internal/metrics/metrics_test.go` — Testes de persistência e agregação de telemetria semântica
- `cmd/pipeline-ai/main.go` — Suporte a `-mode`, `-choices` e `-max-score` em `pipeline-ai eval` e exibição em `metrics summary`
- `cmd/pipeline-ai/main_test.go` — Testes de CLI para `-mode choice` e `-mode score`
- `internal/mcp/server.go` — Suporte a `mode`, `choices` e `max_score` em `pipeline_eval_semantic`
- `internal/mcp/server_test.go` — Testes MCP para `mode: "choice"` e `mode: "score"`
- `README.md` — Documentação dos novos modos do sensor semântico na CLI e MCP
- `docs-ai/DECISOES.md` — Registro da decisão arquitetural `D-35`
- `docs-ai/deliveries/DEL-028/report.md` — Atualização de status para `merged`
- `docs-ai/deliveries/INDEX.md` — Atualização de `DEL-028` para `merged` e registro de `DEL-029`
- `docs-ai/deliveries/DEL-029/report.md` — Relatório desta delivery

# Testes

**Gates:**

```
$ gofmt -l .
(saída vazia, exit code 0)

$ go build ./...
(saída vazia, exit code 0)

$ go vet ./...
(saída vazia, exit code 0)

$ go test -race -count=1 ./internal/sensor/semantic ./cmd/pipeline-ai ./internal/mcp
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	1.674s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	3.227s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	2.713s

$ go test -count=1 ./internal/metrics
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	0.225s

$ go test -run="^$" -fuzz=FuzzParseChoiceResponse -fuzztime=3s ./internal/sensor/semantic
fuzz: elapsed: 0s, gathering baseline coverage: 0/6 completed
fuzz: elapsed: 0s, gathering baseline coverage: 6/6 completed, now fuzzing with 10 workers
fuzz: elapsed: 3s, execs: 421119 (140363/sec), new interesting: 2 (total: 8)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	3.188s

$ go test -run="^$" -fuzz=FuzzParseScoreResponse -fuzztime=3s ./internal/sensor/semantic
fuzz: elapsed: 0s, gathering baseline coverage: 0/6 completed
fuzz: elapsed: 0s, gathering baseline coverage: 6/6 completed, now fuzzing with 10 workers
fuzz: elapsed: 3s, execs: 410163 (136694/sec), new interesting: 3 (total: 9)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	3.189s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| `EvalChoice` e anti-hallucination (`SEC-01`, `ErrInvalidChoice`, `ErrInsufficientConfidence`) | `TestEvalChoice` em `internal/sensor/semantic/semantic_test.go` |
| `EvalScore` e bounds/calibração (`SEC-02`, `ErrInvalidScore`, `NaN`/`Inf`, normalização) | `TestEvalScore` em `internal/sensor/semantic/semantic_test.go` |
| Fuzzing de `ParseChoiceResponse` e `ParseScoreResponse` (`SEC-03`) | `FuzzParseChoiceResponse` e `FuzzParseScoreResponse` em `internal/sensor/semantic/fuzz_test.go` |
| Telemetria semântica (`RecordSemanticEval` e `SemanticSummary`) | `TestRecordSemanticEval` em `internal/metrics/metrics_test.go` |
| Subcomando CLI `pipeline-ai eval -mode choice|score` | `TestRunEvalChoiceAndScore` em `cmd/pipeline-ai/main_test.go` |
| Ferramenta MCP `pipeline_eval_semantic` com `mode: choice|score` | `TestMCPToolEvalSemanticModes` em `internal/mcp/server_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Extração do helper privado `Client.doPost` em `internal/sensor/semantic/client.go` para evitar duplicação de lógica HTTP, `io.LimitReader` e sanitização de credenciais entre `EvalNoul`, `EvalChoice` e `EvalScore`.

**Sugestões recusadas:** nenhuma

# Security

**Superfície revisada:**
- `internal/sensor/semantic/client.go` e `types.go` — Validação de entradas e respostas de modelos externos em `/v1/choice` e `/v1/score`.
- `cmd/pipeline-ai/main.go` e `internal/mcp/server.go` — Superfície de entrada CLI/MCP, limites de payload e redação de tokens.

**Validação dos requisitos de segurança (`REQUISITOS_SEGURANCA`):**
- `SEC-01` (Validação Estrita de Escolhas / Anti-Hallucination): Aprovado. `EvalChoice` exige no mínimo 2 opções válidas e verifica se a escolha retornada pertence estritamente à lista `ChoiceInput.Choices`, retornando `ErrInvalidChoice` em caso de divergência ou alucinação.
- `SEC-02` (Validação de Fronteiras Numéricas e Calibração em `EvalScore`): Aprovado. `ParseScoreResponse` e `EvalScore` rejeitam `NaN`, `+Inf`, `-Inf` e valores fora de `[0.0, MaxScore]` com `ErrInvalidScore`, e exigem que tanto `Confidence >= Threshold` quanto `(Score / MaxScore) >= Threshold`.
- `SEC-03` (Preservação de Redação de Credenciais e Limites de Payload): Aprovado. O helper `doPost` preserva a redação de `APIKey` para `[REDACTED]` e o limite de `MaxBodyBytes` (10MB), enquanto CLI e MCP mantêm o limite de 512KB para `state`.

# Security Review

Todos os requisitos `SEC-01`, `SEC-02` e `SEC-03` especificados no `architecture-guardrails` foram auditados no diff e comprovados por testes unitários, adversariais e de fuzzing, sem riscos residuais.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
