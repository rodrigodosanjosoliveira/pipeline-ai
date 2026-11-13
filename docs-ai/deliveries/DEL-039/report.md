---
delivery_id: DEL-039
task_code: TASK-039
title: "Thinking Tier Matrix (model_routing) e Provider Nativo Gemini Hydrogen no Sensor Semântico"
status: draft
require_security: true
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Orquestrar todas as skills de uma esteira multiagente com um único tier de raciocínio resulta em desperdício de cota de inferência e latência desnecessária em etapas mecânicas (`docs-update`, `git-delivery`, `release-ops`), ao mesmo tempo em que subdimensiona etapas que exigem auditoria rigorosa e raciocínio analítico profundo (`architecture-guardrails`, `code-reviewer`, `security-review`).

Adicionalmente, o sensoriamento semântico calibrado (`pipeline-ai eval` e a ferramenta MCP `pipeline_eval_semantic`) dependia exclusivamente de endpoints de compatibilidade do provedor externo Jev (`JEV_ENDPOINT`), inviabilizando o uso direto de modelos nativos Google Gemini (família Hydrogen / Gemini 2.5) que fornecem suporte estruturado a `responseSchema`, `responseMimeType: "application/json"` e `thinkingConfig.thinkingLevel`.

Esta delivery implementa:
1. A matriz de roteamento de modelos por skill (`model_routing`) em `config.Config` (`.pipeline-ai.json`), distribuindo deterministicamente as 16 skills em 3 tiers (`gemini-4-hydrogen-low`, `gemini-4-hydrogen-medium`, `gemini-4-hydrogen-high`) e expondo-a no recurso MCP `pipeline://catalog`.
2. O provider nativo `"gemini"` em `internal/sensor/semantic` em Go 1.24 puro (100% standard library `net/http`, zero dependências externas em `go.mod`), com suporte a `generateContent`, `responseSchema` estrito por modo (`noul`, `choice`, `score`), `thinkingConfig.thinkingLevel` (padrão `"low"`), autenticação exclusiva via header `x-goog-api-key` (SEC-01) e suporte em CLI e MCP.

# Escopo

- **Thinking Tier Matrix (`model_routing` em `internal/config`):**
  - Adição do campo `ModelRouting map[string]string` na struct `config.Config` (`.pipeline-ai.json`).
  - Funções `DefaultModelRouting()` e `ResolveSkillModel(skill string) string` mapeando as 16 skills em 3 tiers de raciocínio.
  - Validação estrita no carregamento do `.pipeline-ai.json` rejeitando chaves ou valores vazios/em branco.
- **Provider Nativo Gemini no Sensor Semântico (`internal/sensor/semantic`):**
  - Implementação do provider nativo `"gemini"` utilizando a biblioteca padrão de Go (`net/http`, `encoding/json`, `context`).
  - Chamadas REST ao endpoint `v1beta/models/{model}:generateContent`.
  - Suporte a `responseMimeType: "application/json"` e `responseSchema` tipado para os 3 modos de inferência:
    - `noul`: esquema com `decision` (boolean), `confidence` (float 0.0..1.0) e `reasoning` (string).
    - `choice`: esquema com `selected_option` (enum restrito aos itens de `choices`), `confidence` e `reasoning`.
    - `score`: esquema com `score` (float), `confidence` e `reasoning`.
  - Suporte a `thinkingConfig.thinkingLevel` (padrão `"low"`).
  - Salvaguardas de segurança mandatórias:
    - **SEC-01:** Autenticação exclusiva através do header HTTP `x-goog-api-key` (rejeitando query params `?key=` na URL). Redação de `api_key` e `GEMINI_API_KEY` em mensagens de erro via `sanitizeErr` e `redactSecrets`. Validação anti-hallucination em escolhas.
    - **SEC-02:** Limitação estrita de leitura com `io.LimitReader` e `MaxBodyBytes` (10MB), retornando `ErrResponseTooLarge`. Bounds checking normalizado em pontuações contínuas.
    - **SEC-03:** Deadline e cancelamento de sockets com `context.WithTimeout` (default 10s).
  - Fuzz testing contínuo com `FuzzParseGeminiResponse`.
- **CLI Unificada (`cmd/pipeline-ai/main.go`):**
  - Subcomando `pipeline-ai eval` expandido com as flags `-provider` (padrão `"jev"`), `-model` (padrão `"gemini-4-hydrogen-low"` para gemini), `-thinking-level` (padrão `"low"`).
  - Resolução de credencial com fallback para a variável de ambiente `GEMINI_API_KEY` quando o provider for `gemini`.
  - Redação segura de mensagens de erro impedindo vazamento de tokens.
- **Servidor MCP Nativo (`internal/mcp/server.go`):**
  - Ferramenta MCP `pipeline_eval_semantic` enriquecida com os parâmetros opcionais `provider`, `model` e `thinking_level`.
  - Recurso MCP `pipeline://catalog` atualizado incluindo o bloco `model_routing` com o mapeamento de modelos por skill.
- **Documentação e Governança:**
  - `docs-ai/DECISOES.md`: Registro da decisão arquitetural D-41.
  - `README.md`: Documentação das flags na CLI `eval`, parâmetros MCP e atualização do Roadmap.
  - `docs-ai/deliveries/INDEX.md`: Registro de DEL-039 e atualização de DEL-038 para `merged`.
  - `docs-ai/deliveries/DEL-038/report.md`: Sincronização do status para `merged`.
  - `docs-ai/deliveries/DEL-039/report.md`: Relatório completo desta delivery.

**Fora de escopo:**
- Adição de dependências externas (`google.golang.org/genai`) ao `go.mod` (preservando Go puro e stdlib).
- Passagem de chave de API em query parameter `?key=` na URL (bloqueado por segurança).

# Implementação

A implementação foi realizada em camadas com estrita preservação das invariantes arquiteturais:

1. **Configuração e Matriz de Roteamento (`internal/config`):**
   - O struct `Config` em `internal/config/config.go` recebeu o campo `ModelRouting map[string]string` (`json:"model_routing,omitempty"`).
   - `DefaultModelRouting()` define a distribuição determinística das 16 skills em 3 tiers:
     - `gemini-4-hydrogen-low`: skills mecânicas (`docs-update`, `git-delivery`, `release-ops`).
     - `gemini-4-hydrogen-medium`: skills de execução e triagem (`pm-triage`, `test-writer`, `dev-implementation`, `data-model`, `context-loader`, `qa-validation`).
     - `gemini-4-hydrogen-high`: skills de auditoria, arquitetura e revisão analítica (`architecture-guardrails`, `code-reviewer`, `migration-reviewer`, `query-performance`, `security-review`, `supply-chain-review`, `agent-auditor`).
   - `ResolveSkillModel(skill string)` consulta o mapa e retorna o modelo configurado, recorrendo ao padrão de forma resiliente.
   - O validador do `config.Load` rejeita entradas malformadas contendo chaves ou valores vazios em `model_routing`.

2. **Provider Nativo Gemini (`internal/sensor/semantic`):**
   - Em `internal/sensor/semantic/types.go` e `client.go`, definidos os tipos REST do Gemini (`geminiGenerateContentRequest`, `geminiContent`, `geminiPart`, `geminiGenerationConfig`, `geminiThinkingConfig`, `geminiSchema`, `geminiResponse`).
   - `evalGemini` constrói o payload estruturado especificando `responseMimeType: "application/json"`, schemas JSON estritos de acordo com o modo (`noul`, `choice`, `score`), e repassa `thinkingConfig.thinkingLevel`.
   - A autenticação é realizada unicamente injetando o header HTTP `req.Header.Set("x-goog-api-key", client.apiKey)`.
   - As funções `sanitizeErr` e `redactSecrets` foram expandidas para incluir a variável de ambiente `GEMINI_API_KEY` na substituição por `[REDACTED]`.
   - Adicionada a função `FuzzParseGeminiResponse` em `fuzz_test.go` para validação contínua contra payloads corrompidos.

3. **Integração CLI e Servidor MCP:**
   - Em `cmd/pipeline-ai/main.go`, `runEval` adicionou flags `-provider`, `-model`, `-thinking-level`, resolvendo automaticamente a chave `GEMINI_API_KEY` quando o provider for `gemini`.
   - Em `internal/mcp/server.go`, `handleEvalSemantic` extrai `provider`, `model`, `thinking_level` da chamada JSON-RPC, repassando ao cliente semântico.
   - Em `pipeline://catalog`, o JSON emitido agrega a chave `"model_routing"` mapeando cada skill ao seu tier de inferência.

**Arquivos alterados:**
- `internal/config/config.go` — Campo `ModelRouting`, funções `DefaultModelRouting` e `ResolveSkillModel`
- `internal/config/config_test.go` — Testes unitários para resolução e validação de `model_routing`
- `internal/sensor/semantic/types.go` — Enumeração de providers e structs REST da API Gemini
- `internal/sensor/semantic/client.go` — Provider nativo Gemini com `generateContent`, `responseSchema`, header `x-goog-api-key` e redação de `GEMINI_API_KEY`
- `internal/sensor/semantic/semantic_test.go` — Testes dos modos `noul`, `choice`, `score` e salvaguardas SEC-01, SEC-02 e SEC-03
- `internal/sensor/semantic/fuzz_test.go` — Fuzz testing para `ParseGeminiResponse` (+578k execuções)
- `cmd/pipeline-ai/main.go` — Flags `-provider`, `-model`, `-thinking-level` e fallback para `GEMINI_API_KEY` na CLI `eval`
- `cmd/pipeline-ai/main_test.go` — Testes de integração da CLI `eval` com provider Gemini e validação de redação de segredos
- `internal/mcp/server.go` — Suporte a `provider`, `model`, `thinking_level` em `pipeline_eval_semantic` e `model_routing` em `pipeline://catalog`
- `internal/mcp/server_test.go` — Testes das tools e resources MCP com Gemini e catálogo
- `README.md` — Exemplos da CLI `eval` com Gemini, catálogo MCP, `pipeline_eval_semantic` e Roadmap
- `docs-ai/DECISOES.md` — Decisão arquitetural D-41
- `docs-ai/deliveries/INDEX.md` — Registro de DEL-039 e atualização de DEL-038 para merged
- `docs-ai/deliveries/DEL-038/report.md` — Atualização de status para merged
- `docs-ai/deliveries/DEL-039/report.md` — Relatório de entrega desta delivery

# Testes

**Gates:**

```
$ gofmt -l .
(saída limpa)

$ go build ./...
(saída limpa, exit code 0)

$ go vet ./...
(saída limpa, exit code 0)

$ go test ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	2.017s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/debt	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	1.188s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	0.437s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	(cached)
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	(cached)
```

**Fuzz Testing:**

```
$ go test -fuzz=FuzzParseGeminiResponse -fuzztime=5s ./internal/sensor/semantic
fuzz: elapsed: 0s, gathering baseline coverage: 0/354 completed
fuzz: elapsed: 0s, gathering baseline coverage: 354/354 completed, now fuzzing with 10 workers
fuzz: elapsed: 3s, execs: 368144 (122711/sec), new interesting: 26 (total: 380)
fuzz: elapsed: 6s, execs: 578428 (85914/sec), new interesting: 48 (total: 402)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	6.029s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| `internal/config/config.go` (`DefaultModelRouting`, `ResolveSkillModel`) | `TestModelRoutingAndResolveSkillModel` em `internal/config/config_test.go` |
| `internal/sensor/semantic/client.go` (Gemini REST, headers, schemas) | `TestGeminiProvider_EvalModesAndSecurity` em `internal/sensor/semantic/semantic_test.go` |
| `internal/sensor/semantic/client.go` (SEC-01 Anti-Hallucination e header seguro) | `TestGeminiProvider_EvalModesAndSecurity` em `internal/sensor/semantic/semantic_test.go` |
| `internal/sensor/semantic/client.go` (SEC-02 Bounds checking e MaxBodyBytes) | `TestGeminiProvider_EvalModesAndSecurity` em `internal/sensor/semantic/semantic_test.go` |
| `internal/sensor/semantic/client.go` (SEC-03 Redação de tokens e timeout) | `TestGeminiProvider_EvalModesAndSecurity` em `internal/sensor/semantic/semantic_test.go` |
| `internal/sensor/semantic/fuzz_test.go` (FuzzParseGeminiResponse) | `FuzzParseGeminiResponse` em `internal/sensor/semantic/fuzz_test.go` |
| `cmd/pipeline-ai/main.go` (CLI eval flags provider, model, GEMINI_API_KEY) | `TestCLIEvalGeminiProvider` em `cmd/pipeline-ai/main_test.go` |
| `cmd/pipeline-ai/main.go` (Redação de GEMINI_API_KEY em erro na CLI) | `TestCLIEvalGeminiProvider` em `cmd/pipeline-ai/main_test.go` |
| `internal/mcp/server.go` (Tool pipeline_eval_semantic com Gemini) | `TestMCPGeminiProviderAndCatalogRouting` em `internal/mcp/server_test.go` |
| `internal/mcp/server.go` (Resource pipeline://catalog com model_routing) | `TestMCPGeminiProviderAndCatalogRouting` em `internal/mcp/server_test.go` |

**Validação (QA):**
Suíte completa de testes aprovada sem regressões em todos os 18 pacotes.

**Supply Chain:**
Sinal `SUPPLY_CHAIN: nao`. Nenhuma nova dependência externa adicionada em `go.mod`.

**Pendências:**
Nenhuma pendência técnica identificada.

**Handoffs:**
Esteira executada continuamente através dos papéis `pm-triage` → `architecture-guardrails` → `test-writer` → `dev-implementation` → `code-reviewer` → `security-review` → `qa-validation` → `release-ops` → `docs-update`.

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:**
- Rejeição expressa de query parameter `?key=` na URL para eliminar qualquer risco de vazamento de credenciais em traces ou logs de rede HTTP.
- Inclusão da variável de ambiente `GEMINI_API_KEY` na lista de redação determinística de segredos (`sanitizeErr` / `redactSecrets`).
- Definição de `responseSchema` estrito validado por modo (`noul`, `choice`, `score`).

**Sugestões recusadas:** nenhuma

# Security

**Superfície revisada:**
- `internal/sensor/semantic/client.go` e `types.go` — Provider nativo Gemini, serialização REST, schemas estritos e headers de autenticação.
- `cmd/pipeline-ai/main.go` — Flags de CLI, leitura de variável de ambiente `GEMINI_API_KEY` e tratamento de erros.
- `internal/mcp/server.go` — Parâmetros `provider`, `model`, `thinking_level` na ferramenta MCP e exposição do catálogo.

**Validação dos requisitos de segurança (`REQUISITOS_SEGURANCA`):**
- **SEC-01 (Autenticação Segura via Header e Redação de Segredos):**
  - Chaves de API são transmitidas exclusivamente via header HTTP `x-goog-api-key`, nunca como query parameter `?key=` na URL.
  - Funções `sanitizeErr` e `redactSecrets` substituem tokens e chaves de API (`api_key`, `PIPELINE_AI_SEMANTIC_API_KEY`, `GEMINI_API_KEY`) por `[REDACTED]` em saídas de erro, logs, stdout e stderr.
  - Em modo `choice`, validação anti-hallucination estrita rejeita deterministicamente opções que não constem na lista `choices`.
- **SEC-02 (Mitigação de DoS e Limites de Leitura):**
  - O tamanho do `state` fornecido via flag, arquivo ou `stdin` é validado e limitado a 512KB (524.288 bytes) antes de qualquer transmissão.
  - As respostas da API HTTP são consumidas através de `io.LimitReader` limitado a `MaxBodyBytes` (10MB), retornando `ErrResponseTooLarge` caso o servidor retorne payload excessivo.
  - Em modo `score`, validação normalizada de limites rejeita `NaN`, `+Inf`, `-Inf` e notas fora de `[0.0, MaxScore]`.
- **SEC-03 (Timeout Determinístico e Cancelamento):**
  - Todas as chamadas de rede utilizam `context.WithTimeout` com cancelamento automático de sockets e goroutines, impedindo travamentos ou hang de processos e agentes.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
