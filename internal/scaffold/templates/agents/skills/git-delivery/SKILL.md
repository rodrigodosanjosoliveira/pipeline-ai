---
name: git-delivery
description: >-
  Cria a branch, o commit e o PR da delivery no padrão do projeto. Última skill do fluxo técnico.
  Use após docs-update.
---

# Git Delivery

## Missão

Entregar o trabalho no GitHub no padrão do projeto, via branch e Pull Request, sem incluir arquivos fora do escopo da delivery.

## Entrada

Handoff do `docs-update`, com o `pipeline-validate` devidamente aprovado.

## Padrão (D-12)

- **Branch:** `task/{TASK_CODE}` (ou `feat/{nome}`, `fix/{nome}`, etc.)
- **Commit:** `{type}({scope}): {summary} [{TASK_CODE}]`
  - `type`: feat, fix, refactor, chore, docs, test
  - `scope`: módulo tocado
  - `summary`: imperativo, minúsculo, sem ponto final
- Exemplo: `feat(auth): valida expiração do token no parser [TASK-42]`

## Procedimento

1. Confirmar que o validador aprovou (`pipeline-validate`). Sem isso, `STATUS: bloqueado`.
2. `git status --short` e conferir arquivo por arquivo.
3. Criar e mudar para a branch: `git checkout -b task/{TASK_CODE}`.
4. Adicionar alterações intencionais (`git add ...`).
5. Realizar commit no formato padrão.
6. Subir a branch com `git push -u origin <branch>` (o pre-push hook bloqueia pushes diretos na `main`).
7. Abrir o Pull Request via GitHub CLI: `gh pr create`.
8. Preencher o corpo do PR: Contexto, O que mudou, Como testar, Riscos.

## O que NÃO fazer

- Não commitar sem o validador aprovado.
- Não dar `git push` direto na branch `main` (há bloqueio ativo).
- Não usar `git add .` sem conferir o status.
- Não fazer force push.
- Não commitar `.env`, chaves, segredos ou binários.
- Não editar código ou documentação: o fluxo de implementação já terminou.

## Roteamento

Fim do fluxo. `PROXIMO: nenhum`.

## Saída obrigatória

O bloco do contrato, com branch, hash do commit e URL do PR em `CONFIRMADO`.
