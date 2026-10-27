---
delivery_id: DEL-011
task_code: TASK-011
title: Formalização do Security Shift-Left com segurança bidirecional no PRE e POST
status: merged
require_security: true
require_migration: false
require_performance: false
---

# Contexto

No contraditório técnico do framework, a análise comparativa entre as pipelines em produção (notadamente no `ris-backend`) evidenciou uma oportunidade arquitetural de melhoria no fluxo de segurança:
1. No modelo anterior do pipeline-ai (D-13), a revisão de segurança entrava exclusivamente após a implementação e revisão de código (`code-reviewer → security-review`).
2. Esse posicionamento tardio gerava retrabalho quando uma demanda tocava superfície sensível (como auth, cripto ou storage): uma vulnerabilidade de arquitetura ou requisito de segurança não previsto descoberto somente no pós-dev forçava o descarte de código e testes já aprovados.
3. Descobrir e modelar os requisitos de segurança antes da escrita de qualquer código (no PRE) custa ordens de magnitude menos do que remediar depois do código implementado, além de fornecer os vetores exatos para o `test-writer` elaborar testes adversariais e negativos.

Esta entrega formaliza o **Security Shift-Left (C8)** no pipeline-ai: um modelo de segurança bidirecional que combina modelagem de ameaças e definição de `REQUISITOS_SEGURANCA` no PRE (`architecture-guardrails`) com a auditoria rigorosa de conformidade do diff e análise estática no POST (`security-review`).

# Escopo

- Formalização do fluxo de segurança bidirecional em dois momentos (PRE e POST).
- Inclusão do campo condicional `REQUISITOS_SEGURANCA` no contrato de handoff (`.agents/rules/handoff-contract.md` e templates).
- Atualização da skill `architecture-guardrails` para executar modelagem de ameaças e gerar os requisitos mandatórios de segurança quando `SECURITY: sim`.
- Atualização da skill `test-writer` para cobrir os `REQUISITOS_SEGURANCA` com testes adversariais e negativos.
- Atualização da skill `security-review` para realizar a auditoria cruzada do diff contra os requisitos estabelecidos no PRE.
- Evolução formal da Decisão D-13 em `docs-ai/DECISOES.md`.
- Atualização dos diagramas de fluxo em `AGENTS.md` e `docs-ai/03-MAPA-RAPIDO-MODULOS.md`.
- Sincronização de todos os templates embutidos em `internal/scaffold/templates/`.
- Testes automatizados garantindo a integridade dos templates e da propagação via `Sync()`.

**Fora de escopo:**
- Modificações nas rotinas de compilação ou ferramentas da CLI.
- Adicionar dependências externas ao projeto (100% Go stdlib pura).

# Implementação

1. **`.agents/rules/handoff-contract.md` e templates:**
   - Adicionado o campo `REQUISITOS_SEGURANCA: {quando SECURITY: sim, lista numerada de salvaguardas e critérios de teste adversariais}`.
   - Definida a regra mandando que demandas com `SECURITY: sim` obriguem a formalização desse bloco pelo guardrails.
2. **`.agents/skills/architecture-guardrails/SKILL.md` e templates:**
   - Adicionada etapa formal de modelagem de ameaças no PRE: identificar superfícies expostas, premissas violadas e redigir os critérios de segurança.
3. **`.agents/skills/test-writer/SKILL.md` e templates:**
   - Orientação explícita para transformar cada requisito de segurança em testes automatizados adversariais antes da implementação.
4. **`.agents/skills/security-review/SKILL.md` e templates:**
   - Instrução para realizar a auditoria do diff verificando se os critérios do PRE foram cumpridos, além de executar `govulncheck ./...`.
5. **`docs-ai/DECISOES.md` e templates:**
   - D-13 atualizada para "Segurança bidirecional: Requisitos no PRE e Auditoria no POST (Security Shift-Left)", documentando a justificativa de redução de retrabalho e preservação da auditoria perimetral.
6. **`AGENTS.md` e `docs-ai/03-MAPA-RAPIDO-MODULOS.md`:**
   - Diagrama e documentação atualizados destacando `architecture-guardrails [com REQUISITOS_SEGURANCA quando SECURITY: sim]` e `security-review [auditoria do diff vs requisitos do PRE]`.
7. **Testes automatizados (`internal/scaffold`):**
   - Implementados `TestSecurityShiftLeftTemplates` e `TestSyncSecurityShiftLeft` assegurando a consistência dos templates e da sincronização.

**Arquivos alterados:**
- `.agents/rules/handoff-contract.md` — Campo REQUISITOS_SEGURANCA no handoff
- `.agents/skills/architecture-guardrails/SKILL.md` — Modelagem de ameaças no PRE
- `.agents/skills/test-writer/SKILL.md` — Testes adversariais cobrindo requisitos
- `.agents/skills/security-review/SKILL.md` — Auditoria cruzada do diff vs PRE
- `AGENTS.md` — Fluxo oficial com segurança bidirecional
- `docs-ai/03-MAPA-RAPIDO-MODULOS.md` — Descrição dos módulos no novo fluxo
- `docs-ai/DECISOES.md` — D-13 evoluída para segurança bidirecional
- `internal/scaffold/scaffold_test.go` — Teste de validação dos templates embutidos
- `internal/scaffold/sync_test.go` — Teste de sincronização dos contratos via Sync
- `internal/scaffold/templates/AGENTS.md.tmpl` — Template de AGENTS.md atualizado
- `internal/scaffold/templates/agents/rules/handoff-contract.md` — Template de handoff
- `internal/scaffold/templates/agents/skills/architecture-guardrails/SKILL.md` — Template guardrails
- `internal/scaffold/templates/agents/skills/security-review/SKILL.md` — Template security-review
- `internal/scaffold/templates/agents/skills/test-writer/SKILL.md` — Template test-writer
- `internal/scaffold/templates/docs-ai/03-MAPA-RAPIDO-MODULOS.md` — Template mapa de módulos
- `internal/scaffold/templates/docs-ai/DECISOES.md` — Template de decisões arquiteturais
- `docs-ai/deliveries/INDEX.md` — Registro de DEL-011
- `docs-ai/deliveries/DEL-011/report.md` — Relatório de entrega da delivery

# Testes

**Gates:**

```
$ gofmt -l .
(nenhum arquivo fora do padrão)

$ go build ./...
(compilação com sucesso)

$ go vet ./...
(análise estática sem avisos)

$ go test ./...
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	0.298s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.171s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.350s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.192s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.105s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	0.211s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.027s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.884s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.144s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.588s
```

**Cobertura do escopo:**

| Item do escopo | Teste que cobre |
|---|---|
| Templates embutidos com segurança bidirecional | `TestSecurityShiftLeftTemplates` em `internal/scaffold/scaffold_test.go` |
| Propagação dos contratos via `Sync()` | `TestSyncSecurityShiftLeft` em `internal/scaffold/sync_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Simetria total entre os arquivos locais em `.agents/` e os templates embutidos em `internal/scaffold/templates/`.

**Sugestões recusadas:** nenhuma

# Security

**Superfície revisada:**
- Governança perimetral e arquitetura do pipeline (C8 — Security Shift-Left / Segurança Bidirecional)
- Contratos de handoff e especificações de skills de segurança

**Achados:**
- Nenhum achado de vulnerabilidade residual (0 CRITICO, 0 ALTO, 0 MEDIO, 0 BAIXO).
- Auditoria cruzada confirmou que o gate do pós-dev foi mantido sem afrouxamento e que as novas diretrizes do PRE orientam a criação prévia de testes adversariais.
- Zero dependências externas adicionadas.
- Zero credenciais, chaves ou segredos em código.

# Final Checklist

- [x] `gofmt -l .` sem saída
- [x] `go build ./...` limpo
- [x] `go vet ./...` limpo
- [x] `go test ./...` verde
- [x] Escopo coberto por teste
- [x] Nada do fora-de-escopo foi implementado
- [x] Documentação dos módulos tocados atualizada
- [x] Delivery registrada no INDEX.md
