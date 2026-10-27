---
delivery_id: DEL-018
task_code: TASK-018
title: "Auditoria automatizada da matriz de rastreabilidade e validação de handoffs (C12)"
type: feat
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

A governança do `pipeline-ai` fundamenta-se na passagem estruturada de contexto entre agentes/skills através de blocos de handoff (`.agents/rules/handoff-contract.md`). No entanto, até então não existia mecanismo programático e determinístico para auditar se os blocos de handoff emitidos durante a esteira cumpriam os requisitos obrigatórios, tais como evidência real em `CONFIRMADO`, sinais de roteamento válidos e preenchimento de requisitos adversariais quando `SECURITY: sim`.

Esta entrega introduz o motor de validação e auditoria automatizada de handoffs (atendendo à crítica C12 do contraditório técnico):
- Novo pacote `internal/handoff` com parser resiliente e validador de regras de rastreabilidade.
- Subcomando CLI `pipeline-ai handoff validate [-json] <arquivo-ou-string>`.
- Ferramenta MCP `pipeline_validate_handoff` permitindo a auditoria remota de handoffs por agentes inteligentes via JSON-RPC.
- Salvaguarda rigorosa contra evasivas em `CONFIRMADO` (reprovando textos genéricos como "ok", "verifiquei", "tudo certo" desprovidos de comandos e saídas reais).
- Alvo de fuzzing `FuzzParseHandoff` com centenas de milhares de iterações sem panics.

# Escopo

- Criação do pacote `internal/handoff`:
  - Tipos: `Handoff`, `Signals`, `ValidationResult`.
  - Função `Parse(raw string) (*Handoff, error)` com suporte a texto puro ou cercas markdown (`text`, ````).
  - Função `Validate(h *Handoff) []string` verificando campos obrigatórios, valores válidos de `STATUS`, preenchimento substancial de `CONFIRMADO` e salvaguarda condicional de `REQUISITOS_SEGURANCA` quando `SECURITY: sim`.
  - Função `ValidateContent(content string) (*ValidationResult, error)`.
  - Alvo de fuzzing `FuzzParseHandoff`.
- Subcomando CLI `pipeline-ai handoff validate`:
  - Suporte a leitura de arquivos ou passagem de blocos diretos em string.
  - Formato textual e saída estruturada via `-json`.
- Ferramenta MCP `pipeline_validate_handoff`:
  - Registro em `tools/list` e execução em `tools/call`.
- Suíte completa de testes automatizados (unitários, CLI e MCP).

**Fora de escopo:**
- Alterações no formato do bloco de handoff (100% compatível com a regra 7 e padrões existentes).
- Adição de dependências externas (100% Go 1.24+ standard library).

# Implementação

1. **`internal/handoff/handoff.go`:**
   - Implementado o parser linha a linha normalizando chaves e valores.
   - Implementada a validação de campos obrigatórios (`DE`, `PARA`, `DELIVERY_ID`, `STATUS`, `PROXIMO`, `MOTIVO`).
   - Implementada a lista de evasivas em `evasivePhrases`, bloqueando declarações não-comprovadas no campo `CONFIRMADO`.
   - Adicionada a exigência de `REQUISITOS_SEGURANCA` quando o sinal `SECURITY` for `sim`.
2. **`cmd/pipeline-ai/main.go`:**
   - Adicionada a função `runHandoff` com subcomando `validate`.
   - Implementada distinção inteligente entre caminhos de arquivo no disco e strings literais de handoff contendo caminhos em campos internos (`ESCOPO`, `ARQUIVOS`).
3. **`internal/mcp/server.go`:**
   - Adicionada a ferramenta `pipeline_validate_handoff` no catálogo MCP e tratado o retorno estruturado via JSON-RPC.

**Arquivos alterados:**
- `internal/handoff/handoff.go` — Parser e validador de contratos de handoff
- `internal/handoff/handoff_test.go` — Testes unitários e fuzzing do pacote handoff
- `cmd/pipeline-ai/main.go` — Subcomando handoff validate
- `cmd/pipeline-ai/main_test.go` — Testes de validação de handoff via CLI
- `internal/mcp/server.go` — Tool MCP pipeline_validate_handoff
- `internal/mcp/server_test.go` — Testes da tool MCP de validação de handoff
- `docs-ai/deliveries/INDEX.md` — Registro de DEL-018
- `docs-ai/deliveries/DEL-018/report.md` — Relatório de entrega da delivery

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
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	2.924s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.111s
?   	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-init	[no test files]
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.326s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.209s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.079s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	0.006s
?   	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/hook	[no test files]
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	1.889s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.026s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner	4.845s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.782s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	2.756s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.102s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.479s
PASS

$ go test -run=^$ -fuzz=FuzzParseHandoff -fuzztime=5s ./internal/handoff
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff	6.060s
```

**Execução Integrada (`pipeline-ai run`):**

```
=== pipeline-ai runner ===
Delivery: DEL-018

 [PASS] doctor     : Ambiente validado com sucesso
 [PASS] sensor     : sensores aprovados (447 testes executados)
 [PASS] validate   : validação aprovada (0 erros, 0 avisos)

Status geral: PASS
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Parsing de handoff e alvos de fuzzing | `TestParse_Success` e `FuzzParseHandoff` em `internal/handoff/handoff_test.go` |
| Validação de campos obrigatórios | `TestValidate_MandatoryFields` em `internal/handoff/handoff_test.go` |
| Salvaguarda de evidência em CONFIRMADO | `TestValidate_EvidenceRequirement` em `internal/handoff/handoff_test.go` |
| Exigência de REQUISITOS_SEGURANCA condicional | `TestValidate_SecurityRequirementsCondition` em `internal/handoff/handoff_test.go` |
| CLI `pipeline-ai handoff validate` (arquivo e string inline) | `TestCLIHandOffValidate` em `cmd/pipeline-ai/main_test.go` |
| Tool MCP `pipeline_validate_handoff` | `TestMCPToolsValidateHandoff` em `internal/mcp/server_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Detecção de blocos inline com caminhos de arquivo em campos como `ESCOPO` ou `ARQUIVOS` para que a CLI diferencie confiavelmente strings passadas diretamente de arquivos no disco.

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
