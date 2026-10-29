---
delivery_id: DEL-025
task_code: TASK-025
title: "Sensoriamento Semântico e Decisões Calibradas com Jev (pacote internal/sensor/semantic em Go puro)"
status: merged
require_security: true
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

A automação de decisões e a inferência de conformidade em pipelines de engenharia de software assistidos por inteligência artificial frequentemente esbarram no não-determinismo, latência e custo computacional de Large Language Models (LLMs) generativos quando aplicados a problemas de classificação binária ou avaliação estrita de critérios de qualidade.

A integração de modelos tipo "System One" (Jev) fornece uma abordagem de inferência rápida, calibrada e estritamente tipada. O sensoriamento semântico permite consultar modelos especializados para verificação de hipóteses, conformidade de critérios e decisões booleanas (Noul), sem geração estocástica de texto livre. 

Esta entrega introduz o pacote `internal/sensor/semantic` no framework Pipeline-AI, fornecendo a base arquitetural para sensoriamento semântico calibrado em Go 1.24+ puro, mantendo as invariantes do projeto: zero dependências externas em `go.mod`, comportamento fail-closed seguro, proteção ativa de credenciais e resiliência a payloads malformados via Fuzz testing contínuo.

# Escopo

- Implementação do pacote `internal/sensor/semantic` na biblioteca padrão de Go:
  - Definição dos tipos estruturados e contratos de domínio em `types.go` (`NoulDecision`, `ChoiceDecision`, `ScoreDecision`, `EvalRequest`, `ClientConfig`).
  - Definição de erros canônicos do domínio (`ErrInvalidRequest`, `ErrInsufficientConfidence`, `ErrResponseTooLarge`, `ErrUnauthorized`, `ErrServerFailure`).
  - Implementação do cliente HTTP nativo em `client.go` com os métodos `NewClient`, `IsConfigured`, `CheckAvailable`, `EvalNoul` e `ParseNoulResponse`.
  - Redação e sanitização automática de chaves de API (`APIKey`) em mensagens de erro (`SEC-01`).
  - Limite estrito de leitura do corpo de respostas HTTP (`MaxBodyBytes` com default de 10MB) via `io.LimitReader` para mitigação de exaustão de memória e DoS (`SEC-02`).
  - Respeito integral ao cancelamento e timeouts via `context.Context` (`SEC-03`).
- Suíte de testes abrangente em `semantic_test.go`:
  - Testes de happy path e cenários com thresholds de confiança.
  - Testes de validação de parâmetros de requisição.
  - Testes de segurança cobrindo redação de tokens, corte de body excessivo e cancelamento por contexto.
  - Testes de mapeamento de status codes HTTP (401, 500, etc.).
- Bateria de testes de robustez por Fuzzing em `fuzz_test.go` (`FuzzParseNoulResponse`).
- Formalização da decisão arquitetural D-33 em `docs-ai/DECISOES.md`.
- Registro formal de DEL-025 no índice central `docs-ai/deliveries/INDEX.md`.

**Fora de escopo:**
- Adição de dependências de terceiros no `go.mod` (100% Go standard library preservado).
- Implementação de mock/servidor Jev real embarcado no binário CLI (o pacote atua como cliente integrador com suporte a endpoints remotos ou locais configuráveis).
- Alteração nos contratos existentes de outros sensores determinísticos em `internal/sensor`.

# Implementação

O pacote `internal/sensor/semantic` foi projetado para atuar como o ponto focal de consultas semânticas determinísticas do pipeline.

1. **Tipos e Contratos (`types.go`):**
   - Estrutura `NoulDecision`: encapsula a decisão booleana (`Decision`), score de calibração (`Confidence`), justificativa concisa (`Reason`) e latência em milissegundos (`LatencyMs`).
   - Estrutura `ChoiceDecision` e `ScoreDecision`: tipagem para classificação múltipla e scores contínuos.
   - Estrutura `EvalRequest`: modela o estado avaliado (`State`), a pergunta/critério (`Question`) e o limiar mínimo de confiança aceitável (`Threshold` entre `0.0` e `1.0`).
   - Estrutura `ClientConfig`: parâmetros de conexão contendo `Endpoint`, `APIKey`, `Timeout` e `MaxBodyBytes`.

2. **Cliente Semântico (`client.go`):**
   - `NewClient(cfg ClientConfig) *Client`: instancia o cliente HTTP nativo configurando timeout seguro.
   - `IsConfigured() bool`: valida presença de credenciais e endpoint não vazios.
   - `CheckAvailable(ctx context.Context) bool`: verifica conectividade e prontidão do endpoint via requisição GET autenticada com Bearer token.
   - `EvalNoul(ctx context.Context, req EvalRequest) (*NoulDecision, error)`: executa a avaliação semântica completa:
     - Validação estrita de parâmetros de entrada (retorna `ErrInvalidRequest` em caso de dados inválidos).
     - Serialização JSON do payload e envio via POST autenticado.
     - Proteção contra DoS e zip-bombs através de `io.LimitReader(resp.Body, maxLimit+1)` retornando `ErrResponseTooLarge` quando o tamanho exceder `MaxBodyBytes`.
     - Tratamento específico de erros HTTP (`401` -> `ErrUnauthorized`, `5xx` -> `ErrServerFailure`).
     - Cálculo de latência da requisição quando não fornecido pelo servidor.
     - Validação fail-closed do score de confiança contra o threshold exigido (`ErrInsufficientConfidence`).
   - `ParseNoulResponse(data []byte) (*NoulDecision, error)`: parser isolado e seguro para desserialização da resposta JSON.
   - `sanitizeErr(err error, apiKey string) error`: mecanismo de redação que intercepta qualquer erro e substitui a chave de API por `[REDACTED]`.

**Arquivos alterados:**
- `internal/sensor/semantic/types.go` — Definições de tipos, contratos e erros canônicos
- `internal/sensor/semantic/client.go` — Cliente HTTP, sanitização de erros e parser Noul
- `internal/sensor/semantic/semantic_test.go` — Testes unitários funcionais e de segurança
- `internal/sensor/semantic/fuzz_test.go` — Fuzz testing para desserialização de payloads
- `docs-ai/DECISOES.md` — Formalização da Decisão D-33 na seção 12
- `docs-ai/deliveries/INDEX.md` — Registro da entrega DEL-025 na tabela de deliveries
- `docs-ai/deliveries/DEL-025/report.md` — Relatório formal de entrega

# Testes

**Gates:**

```
$ gofmt -l .
(nenhum arquivo fora do padrão)

$ go build ./...
(compilação com sucesso, exit code 0)

$ go vet ./...
(análise estática sem avisos, exit code 0)

$ go test -count=1 ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	11.405s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.141s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	6.089s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.569s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard	0.075s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.117s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	0.016s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	7.738s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics	0.343s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin	1.377s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.062s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	17.456s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	3.962s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	10.999s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	0.861s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.827s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	2.774s
```

**Fuzz Testing:**

O teste contínuo `FuzzParseNoulResponse` foi executado com mutações randômicas e corpus representativo, superando 270.000 iterações sem nenhum panic ou falha de integridade de memória:

```
$ go test -fuzz=FuzzParseNoulResponse -fuzztime=8s ./internal/sensor/semantic/...
fuzz: elapsed: 0s, gathering baseline coverage: 0/343 completed
fuzz: elapsed: 2s, gathering baseline coverage: 343/343 completed, now fuzzing with 18 workers
fuzz: elapsed: 3s, execs: 59668 (19885/sec), new interesting: 12 (total: 355)
fuzz: elapsed: 6s, execs: 187907 (42745/sec), new interesting: 31 (total: 374)
fuzz: elapsed: 8s, execs: 267614 (38168/sec), new interesting: 43 (total: 386)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic	8.417s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Avaliação booleana (Noul) com sucesso e alta confiança | `TestEvalNoul_HappyPath` em `internal/sensor/semantic/semantic_test.go` |
| Verificação de threshold e corte por baixa confiança | `TestEvalNoul_ConfidenceThreshold` em `internal/sensor/semantic/semantic_test.go` |
| Validação estrita de parâmetros de entrada (`EvalRequest`) | `TestEvalNoul_Validation` em `internal/sensor/semantic/semantic_test.go` |
| Proteção e redação de APIKey em mensagens de erro (SEC-01) | `TestEvalNoul_Security_TokenRedaction` em `internal/sensor/semantic/semantic_test.go` |
| Limite estrito de leitura de payload HTTP (SEC-02) | `TestEvalNoul_Security_MaxBodyBytes` em `internal/sensor/semantic/semantic_test.go` |
| Respeito a context cancelation e timeout (SEC-03) | `TestEvalNoul_Security_ContextTimeout` em `internal/sensor/semantic/semantic_test.go` |
| Mapeamento de erros HTTP 401, 500 e malformação de resposta | `TestEvalNoul_HTTPErrors` em `internal/sensor/semantic/semantic_test.go` |
| Verificação de configuração e disponibilidade de endpoint | `TestClient_Availability` em `internal/sensor/semantic/semantic_test.go` |
| Desserialização de respostas Noul | `TestParseNoulResponse` em `internal/sensor/semantic/semantic_test.go` |
| Resiliência a entradas malformadas por Fuzzing | `FuzzParseNoulResponse` em `internal/sensor/semantic/fuzz_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:**
- Isolamento do parser `ParseNoulResponse` em função pura para permitir Fuzz testing direcionado na biblioteca padrão de testes de Go (`testing.F`).
- Implementação de `sanitizedError` com suporte a `Unwrap()` para preservar compatibilidade com `errors.Is` e `errors.As` enquanto oculta segredos no output de `Error()`.
- Adoção de padrão fail-closed caso o threshold de confiança retornado seja inferior ao solicitado pela esteira.

**Sugestões recusadas:** nenhuma

# Security

**Superfície revisada:**
- Comunicação de rede HTTP com serviços de inferência semântica (`net/http.Client`).
- Manipulação e transmissão de credenciais de autenticação (`Authorization: Bearer <APIKey>`).
- Leitura, processamento e desserialização de payloads JSON não confiáveis (`encoding/json`).

**Cumprimento dos Requisitos de Segurança:**
- **SEC-01 (Proteção de Segredos):** A função `sanitizeErr` garante que a chave `APIKey` seja redactada (`[REDACTED]`) em qualquer mensagem de erro gerada durante o ciclo de vida da requisição HTTP, impedindo vazamentos em logs ou stacktraces.
- **SEC-02 (Mitigação de DoS e Limitação de Payload):** A leitura do corpo de respostas remotas é estritamente delimitada por `io.LimitReader(resp.Body, maxLimit+1)`. Payloads que ultrapassem o limite de `MaxBodyBytes` (default de 10MB) são abortados imediatamente com `ErrResponseTooLarge`, prevenindo exaustão de memória e ataques de bomba descompressiva.
- **SEC-03 (Controle de Concorrência e Evasão de Timeout):** Todas as operações utilizam `context.Context` repassado para `http.NewRequestWithContext`, garantindo que cancelamentos e expirações de deadline interrompam conexões e sockets sem retenção de goroutines zumbis.
- **SEC-04 (Auditoria de Dependências e Vulnerabilidades):** O projeto mantém rigorosamente zero dependências externas em `go.mod` (100% Go 1.24+ standard library). O código foi submetido à análise de conformidade de segurança e não introduz bibliotecas de terceiros ou código binário opaco.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
