---
delivery_id: DEL-014
task_code: TASK-014
title: Checkpoints de invariantes, critérios de reversibilidade/rollback e gestão dinâmica de módulos
status: merged
require_security: false
require_migration: false
require_performance: false
require_rollback: true
---

# Contexto

O contraditório técnico do framework identificou duas deficiências de governança e operabilidade técnica:
1. **Ausência de Critérios de Reversibilidade e Rollback (C16):** Entregas técnicas podiam ser submetidas e aprovadas sem especificar previamente um plano de contingência ou reversibilidade. Quando uma entrega em produção falhava, não havia garantia determinística de como efetuar o rollback seguro.
2. **Gestão Estática de Módulos Opcionais (C14):** A ativação ou desativação de módulos opcionais (`enabled_modules`) exigia edição manual de JSON no arquivo `.pipeline-ai.json`, propensa a erros de sintaxe ou inclusão de módulos inválidos.

Esta entrega introduz:
- Suporte a `require_rollback` na configuração do projeto e no frontmatter dos relatórios, exigindo formalmente a seção `# Rollback Plan` (ou `# Plano de Rollback` / `# Reversibilidade`), com checagens em `architecture-guardrails` e `code-reviewer`.
- Subcomando CLI `pipeline-ai module [list | enable <nome> | disable <nome>]` e ferramenta MCP `pipeline_manage_modules` para manipulação segura e idempotente dos módulos opcionais reconhecidos.
- Checkpoints de integridade e invariantes no diagnóstico (`pipeline-ai doctor`).

# Escopo

- Em `internal/config`:
  - Campo `RequireRollback *bool` e método `IsRollbackRequired() bool`.
  - Métodos `EnableModule(mod string) error`, `DisableModule(mod string) error` e `Save(targetPath ...string) error`.
  - Validação estrita contra `AllowedModules`.
- Em `internal/report`:
  - Validação da seção de rollback (`Rollback Plan`, `Plano de Rollback`, `Reversibilidade`) quando `require_rollback` estiver ativo (no frontmatter ou por política do projeto em `.pipeline-ai.json`).
  - Rejeição mandatória caso a seção esteja ausente, vazia ou contenha marcadores de evasão (`N/A`, `não se aplica`).
  - Alvo de fuzzing `FuzzValidate_RequireRollback`.
- Em `cmd/pipeline-ai`:
  - Subcomando `module` com `list`, `enable` e `disable`, com flag `-dir`.
- Em `internal/mcp`:
  - Tool MCP `pipeline_manage_modules` com ações `list`, `enable` e `disable`.
- Em `internal/doctor`:
  - Checkpoint de integridade reportando módulos ativos e política mandatória de rollback.
- Documentação e templates:
  - Atualização de `.agents/skills/architecture-guardrails/SKILL.md` e `.agents/skills/code-reviewer/SKILL.md`.
  - Atualização do template `docs-ai/deliveries/_template/report.md`.

**Fora de escopo:**
- Dependências externas de terceiros (100% Go stdlib).
- Alterações em módulos de persistência ou schema de banco.

# Implementação

1. **`internal/config/config.go`:**
   - Adicionado campo `RequireRollback *bool` na struct `Config`.
   - Implementado `IsRollbackRequired()` com fallback seguro para false.
   - Implementados `EnableModule`, `DisableModule` e `Save` com persistência formatada e validação atômica.
2. **`internal/report/report.go`:**
   - Implementado `checkRollback()` integrado no `Validate()` e no `ValidateWithConfig(cfg)`.
   - Detecção de seções normalizadas de rollback e bloqueio de marcadores `isNA()`.
3. **`internal/doctor/doctor.go`:**
   - Atualizado `CheckConfig` para registrar os módulos habilitados e destacar a exigência da política de rollback mandatória.
4. **`cmd/pipeline-ai/main.go`:**
   - Registrado o subcomando `module` com suporte a `list`, `enable` e `disable`.
5. **`internal/mcp/server.go`:**
   - Adicionada ferramenta `pipeline_manage_modules` ao catálogo MCP e seu manipulador JSON-RPC.
6. **Skills e Templates:**
   - Atualizados `SKILL.md` de guardrails e code-reviewer, bem como os templates de scaffolding.

**Arquivos alterados:**
- `internal/config/config.go` — Métodos de gestão de módulos e campo de rollback
- `internal/config/config_test.go` — Testes unitários de gestão de módulos e salvamento
- `internal/report/report.go` — Validação de rollback no relatório
- `internal/report/report_test.go` — Testes e fuzzing de validação de rollback
- `internal/doctor/doctor.go` — Checkpoints de integridade de módulos e invariantes
- `internal/doctor/doctor_test.go` — Testes de diagnóstico de integridade
- `cmd/pipeline-ai/main.go` — Subcomando module
- `cmd/pipeline-ai/main_test.go` — Testes do subcomando module
- `internal/mcp/server.go` — Tool MCP pipeline_manage_modules
- `internal/mcp/server_test.go` — Testes da tool MCP
- `.agents/skills/architecture-guardrails/SKILL.md` — Requisitos de reversibilidade
- `.agents/skills/code-reviewer/SKILL.md` — Verificação de reversibilidade
- `docs-ai/deliveries/_template/report.md` — Seção de rollback no template
- `internal/scaffold/templates/...` — Templates embutidos sincronizados
- `docs-ai/deliveries/INDEX.md` — Registro de DEL-014
- `docs-ai/deliveries/DEL-014/report.md` — Relatório de entrega da delivery

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
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	1.229s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.120s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.356s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.210s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.102s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	1.166s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.040s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.819s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	2.589s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.103s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.509s
PASS

$ go test -run=^$ -fuzz=FuzzValidate_RequireRollback -fuzztime=5s ./internal/report
PASS
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	6.063s
```

**Execução de Sensores (`pipeline-ai sensor`):**

```
Status geral: PASS
Testes executados: 388
- [PASS] fmt: Todos os arquivos estão formatados segundo o padrão gofmt.
- [PASS] vet: go vet executado com sucesso sem avisos ou erros.
- [PASS] test: Todos os testes passaram (388 testes executados, 0 pulados).
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Gestão dinâmica de módulos e persistência | `TestConfig_RequireRollback`, `TestConfig_ModuleManagement` em `internal/config/config_test.go` |
| Validação de reversibilidade/rollback e fuzzing | `TestValidate_RequireRollback` e `FuzzValidate_RequireRollback` em `internal/report/report_test.go` |
| CLI `pipeline-ai module` (list, enable, disable) | `TestCLIModuleCommands` em `cmd/pipeline-ai/main_test.go` |
| Tool MCP `pipeline_manage_modules` | `TestMCPToolsManageModules` em `internal/mcp/server_test.go` |
| Checkpoint de integridade no Doctor | `TestDoctorModuleAndInvariantIntegrity` em `internal/doctor/doctor_test.go` |

# Rollback Plan

Esta entrega adiciona funcionalidades puramente aditivas e retrocompatíveis sem dependências externas nem alterações em formatos pré-existentes. Em caso de necessidade de reversão:

1. **Reversão via Git:**
   ```bash
   git revert -m 1 HEAD
   ```
2. **Impacto:** Nenhuma alteração irreversível de esquema, filesystem ou estado de execução é persistida. Configurações que não utilizem `require_rollback` ou comandos de módulo não são impactadas.
3. **Verificação pós-reversão:**
   ```bash
   go test ./...
   go run ./cmd/pipeline-validate docs-ai/deliveries/DEL-013/report.md
   ```

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Sincronização automática dos templates embutidos em `internal/scaffold/templates/` para garantir consistência em novos projetos criados via `pipeline-ai init`.

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
