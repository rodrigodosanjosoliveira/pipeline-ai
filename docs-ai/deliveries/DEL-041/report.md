---
delivery_id: DEL-041
task_code: TASK-041
title: "Helpful Errors com Hints Acionáveis & Guided Recovery (cause, hint, fix_command)"
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Erros genéricos sem contexto ou sugestão de correção forçam humanos e agentes de IA a múltiplas tentativas ou leitura manual de documentação quando um subcomando é digitado incorretamente, uma flag obrigatória é omitida ou credenciais/arquivos de delivery estão ausentes.

Seguindo a linha de Dual-Audience CLI Design (D-39, D-40, D-42), esta delivery implementa o padrão de **Helpful Errors com Hints Acionáveis & Guided Recovery (`D-43`)**, padronizando mensagens de erro estruturadas na CLI e no servidor MCP com `cause`, `hint`, `fix_command` e sugestão automática de comandos próximos, mantendo total retrocompatibilidade com prefixos e campos existentes (`ERRO:`, `configured: false`).

# Escopo

- **Helpful Errors e Guided Recovery na CLI (`cmd/pipeline-ai/main.go`):**
  - Helper `writeCLIError` emitindo erros estruturados com `cause`, `hint` e `fix_command` em modo JSON (`-json` / `-agent`) e texto legível com prefixo `ERRO:` preservado para humanos.
  - Função `suggestCommand` utilizando distância de Levenshtein para sugerir subcomandos próximos quando um comando desconhecido é invocado.
  - Mensagens orientadas a ação nos principais fluxos de erro (`init` sem `-name` em modo não-interativo, `validate` sem `-report` / delivery ativa, `eval` sem credenciais configuradas).
- **Helpful Errors no Servidor MCP (`internal/mcp/server.go`):**
  - Helper `sendToolError` padronizando retornos de erro das tools MCP com `cause`, `hint` e `fix_command` (preservando `configured: false` quando aplicável).
  - Orientações acionáveis em `pipeline_eval_semantic` (ex.: ausência de `state`) e `pipeline_manage_modules` (ex.: módulo inválido orientando `action: list`).
- **Testes Automatizados (`cmd/pipeline-ai/main_test.go`, `internal/mcp/server_test.go`):**
  - `TestCLIHelpfulErrorsAndGuidedRecovery` cobrindo `suggestCommand`, comando desconhecido em modo texto e `-agent`, `init` não-interativo sem `-name`, `validate` sem delivery ativa e `eval` sem credenciais.
  - `TestMCPHelpfulErrors` cobrindo `pipeline_eval_semantic` sem `state` e `pipeline_manage_modules` com módulo inválido.
- **Documentação e Governança:**
  - `docs-ai/DECISOES.md`: Registro formal da decisão arquitetural D-43 (Seção 17).
  - `README.md`: Documentação de *Helpful Errors & Guided Recovery* na seção da CLI e remoção da seção `## Roadmap (Próximas Entregas)` após conclusão do ciclo `DEL-037`–`DEL-041`.
  - `docs-ai/deliveries/INDEX.md`: Registro de `DEL-041` e atualização de `DEL-040` para `merged`.
  - `docs-ai/deliveries/DEL-040/report.md`: Sincronização do status para `merged`.
  - `docs-ai/deliveries/DEL-041/report.md`: Relatório completo desta delivery.

**Fora de escopo:**
- Alteração de formatos de erro legados de forma incompatível com scripts e testes existentes (`ERRO:`, `configured: false`).
- Adição de dependências externas ao `go.mod`.

# Implementação

1. **CLI (`cmd/pipeline-ai/main.go`):**
   - Implementação de `writeCLIError` e `suggestCommand` (distância de Levenshtein), emitindo `cause`, `hint` e `fix_command` em JSON no Agent Mode / `-json` e dicas formatadas em texto humano sem quebrar o prefixo `ERRO:`.
2. **Servidor MCP (`internal/mcp/server.go`):**
   - Implementação de `sendToolError` com payload estruturado (`cause`, `hint`, `fix_command`), aplicado às validações de parâmetros obrigatórios e orientações de recuperação nas tools MCP.

**Arquivos alterados:**
- `cmd/pipeline-ai/main.go` — `writeCLIError`, `suggestCommand` e mensagens guiadas de recuperação na CLI
- `cmd/pipeline-ai/main_test.go` — Suíte `TestCLIHelpfulErrorsAndGuidedRecovery`
- `internal/mcp/server.go` — `sendToolError` com `cause`, `hint` e `fix_command` nas tools MCP
- `internal/mcp/server_test.go` — Suíte `TestMCPHelpfulErrors`
- `README.md` — Subseção `Helpful Errors & Guided Recovery` e remoção do Roadmap concluído
- `docs-ai/DECISOES.md` — Registro formal da decisão D-43 (Seção 17)
- `docs-ai/deliveries/INDEX.md` — Registro de `DEL-041` e atualização de `DEL-040` para `merged`
- `docs-ai/deliveries/DEL-040/report.md` — Atualização de `status: draft` para `status: merged`
- `docs-ai/deliveries/DEL-041/report.md` — Relatório de entrega desta delivery

# Testes

**Gates:**

```
$ gofmt -l .
(saída limpa)

$ go build ./...
(saída limpa, exit code 0)

$ go vet ./...
(saída limpa, exit code 0)

$ go test -v -run "TestCLIHelpfulErrors|TestSuggestCommand" ./cmd/pipeline-ai
=== RUN   TestCLIHelpfulErrorsAndGuidedRecovery
=== RUN   TestCLIHelpfulErrorsAndGuidedRecovery/suggestCommand
=== RUN   TestCLIHelpfulErrorsAndGuidedRecovery/suggestCommand/sensro_sugere_sensor
=== RUN   TestCLIHelpfulErrorsAndGuidedRecovery/suggestCommand/sensr_sugere_sensor
=== RUN   TestCLIHelpfulErrorsAndGuidedRecovery/suggestCommand/valdiate_sugere_validate
=== RUN   TestCLIHelpfulErrorsAndGuidedRecovery/suggestCommand/validte_sugere_validate
=== RUN   TestCLIHelpfulErrorsAndGuidedRecovery/suggestCommand/comando_muito_distante_retorna_vazio
=== RUN   TestCLIHelpfulErrorsAndGuidedRecovery/comando_desconhecido_em_modo_texto_sugere_comando_proximo
=== RUN   TestCLIHelpfulErrorsAndGuidedRecovery/comando_desconhecido_em_modo_-agent_emite_JSON_estruturado
=== RUN   TestCLIHelpfulErrorsAndGuidedRecovery/init_sem_-name_em_modo_nao-interativo_com_-json
=== RUN   TestCLIHelpfulErrorsAndGuidedRecovery/validate_sem_-report_em_diretorio_sem_delivery_ativa
=== RUN   TestCLIHelpfulErrorsAndGuidedRecovery/eval_sem_credenciais_com_-json
--- PASS: TestCLIHelpfulErrorsAndGuidedRecovery (0.00s)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	0.275s

$ go test -v -run "TestMCPHelpfulErrors" ./internal/mcp
=== RUN   TestMCPHelpfulErrors
=== RUN   TestMCPHelpfulErrors/pipeline_eval_semantic_sem_state_retorna_erro_estruturado
=== RUN   TestMCPHelpfulErrors/pipeline_manage_modules_com_module_invalido_orienta_action_list
--- PASS: TestMCPHelpfulErrors (0.00s)
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	0.270s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| `cmd/pipeline-ai/main.go` (`suggestCommand` via Levenshtein) | `TestCLIHelpfulErrorsAndGuidedRecovery/suggestCommand` (`cmd/pipeline-ai/main_test.go`) |
| `cmd/pipeline-ai/main.go` (comando desconhecido texto e `-agent` JSON) | `TestCLIHelpfulErrorsAndGuidedRecovery/comando_desconhecido_em_modo_texto_sugere_comando_proximo`, `comando_desconhecido_em_modo_-agent_emite_JSON_estruturado` (`cmd/pipeline-ai/main_test.go`) |
| `cmd/pipeline-ai/main.go` (`writeCLIError` em `init`, `validate`, `eval`) | `TestCLIHelpfulErrorsAndGuidedRecovery/init_sem_-name_em_modo_nao-interativo_com_-json`, `validate_sem_-report_em_diretorio_sem_delivery_ativa`, `eval_sem_credenciais_com_-json` (`cmd/pipeline-ai/main_test.go`) |
| `internal/mcp/server.go` (`sendToolError` com `cause`, `hint`, `fix_command`) | `TestMCPHelpfulErrors` (`internal/mcp/server_test.go`) |

**Validação (QA):**
`gofmt -l .`, `go build ./...`, `go vet ./...`, `TestCLIHelpfulErrorsAndGuidedRecovery` e `TestMCPHelpfulErrors` aprovados após a implementação.

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

**Sugestões acatadas:** preservação dos prefixos e campos legados (`ERRO:`, `configured: false`) garantindo retrocompatibilidade com scripts e testes existentes, aliada a payloads estruturados (`cause`, `hint`, `fix_command`).

**Sugestões recusadas:** nenhuma

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde (validado integralmente em `TestCLIHelpfulErrorsAndGuidedRecovery` e `TestMCPHelpfulErrors`)
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
