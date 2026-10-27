---
delivery_id: DEL-016
task_code: TASK-016
title: "Regra formal de orquestração autônoma e princípio da não-interrupção"
type: feat
status: draft
require_security: false
require_migration: false
require_performance: false
require_rollback: false
---

# Contexto

Durante o ciclo de desenvolvimento das entregas anteriores, o orquestrador da sessão principal pausava a execução a cada transição de etapa para solicitar permissão ao usuário ("Deseja prosseguir com...?"), gerando um ciclo repetitivo de confirmações manuais por gate.

Essa fricção burocrática contrariava o princípio de gates determinísticos do `pipeline-ai` e dos pipelines de referência (`Guard-IA` e `ris-backend`), nos quais a intervenção humana deve ocorrer por exceção e não por rotina.

Esta entrega formaliza o **Princípio da Não-Interrupção**:
- O orquestrador conduz a esteira de ponta a ponta sem pedir confirmação intermediária quando o status for `ok`.
- As interrupções ficam restritas a três eventos legítimos: ambiguidade crítica de requisitos, falha bloqueante (`STATUS: bloqueado`) ou conclusão com abertura do Pull Request (`git-delivery`).

# Escopo

- Criação da regra de orquestração:
  - `.agents/rules/orchestration.md`
- Atualização do contrato de handoff:
  - `.agents/rules/handoff-contract.md`: adição da Regra 7 formalizando a transição contínua automática quando `STATUS: ok`.
- Atualização das diretrizes globais:
  - `AGENTS.md`: formalização do Princípio da Não-Interrupção na seção "Como o trabalho acontece aqui".
- Sincronização dos templates embutidos:
  - `internal/scaffold/templates/agents/rules/orchestration.md`
  - `internal/scaffold/templates/agents/rules/handoff-contract.md`
- Testes automatizados:
  - `internal/scaffold/sync_test.go` e `internal/scaffold/scaffold_test.go` verificando que a nova regra é propagada em novos projetos e no comando `pipeline-ai sync`.

**Fora de escopo:**
- Criação de novos binários ou comandos CLI nesta entrega (escopo da DEL-017 / pipeline runner).

# Implementação

1. **`.agents/rules/orchestration.md`:**
   - Definidos o papel do orquestrador, as regras de avanço autônomo e as 3 condições exclusivas de interrupção (ambiguidade de negócio, bloqueio técnico e entrega concluída).
2. **`.agents/rules/handoff-contract.md`:**
   - Adicionada a regra 7 estabelecendo o avanço compulsório ao próximo agente sem solicitar confirmações manuais intermediárias.
3. **`AGENTS.md`:**
   - Atualizada a introdução operacional do framework documentando a esteira não-interrompida.
4. **`internal/scaffold/templates/`:**
   - Templates sincronizados e cobertos pela suíte de testes do scaffold e sync.

**Arquivos alterados:**
- `.agents/rules/orchestration.md` — Nova regra de orquestração autônoma
- `.agents/rules/handoff-contract.md` — Regra 7 de transição contínua
- `AGENTS.md` — Formalização do princípio da não-interrupção
- `internal/scaffold/templates/agents/rules/orchestration.md` — Template embutido
- `internal/scaffold/templates/agents/rules/handoff-contract.md` — Template embutido
- `internal/scaffold/sync_test.go` — Teste de sincronização da nova regra
- `internal/scaffold/scaffold_test.go` — Teste de scaffold com a nova regra
- `docs-ai/deliveries/INDEX.md` — Registro de DEL-016
- `docs-ai/deliveries/DEL-016/report.md` — Relatório de entrega da delivery

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
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-ai	1.146s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-hook	0.082s
?   	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-init	[no test files]
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-validate	1.266s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config	0.170s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor	0.065s
?   	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/hook	[no test files]
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp	1.088s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/report	0.029s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold	0.781s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor	2.221s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state	0.087s
ok  	github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate	0.460s
PASS
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
| Sincronização de `orchestration.md` via `pipeline-ai sync` | `TestSyncAgents` em `internal/scaffold/sync_test.go` |
| Presença de `orchestration.md` no scaffold de projetos novos | `TestScaffoldDesktop` em `internal/scaffold/scaffold_test.go` |

# Code Review

**Bloqueios:** nenhum

**Sugestões acatadas:** Inclusão de asserções em `scaffold_test.go` e `sync_test.go` garantindo que tanto novos projetos quanto projetos legados que rodam `pipeline-ai sync` recebam a regra.

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
