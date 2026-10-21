---
name: docs-update
description: >-
  Atualiza a documentação afetada e monta o relatório de delivery para validação.
  Use após qa-validation ou release-ops, antes de git-delivery.
---

# Docs Update

## Missão

Deixar a documentação coerente com o que mudou e montar o relatório de delivery que o `pipeline-validate` vai checar antes da conclusão da tarefa.

## Entrada

Todos os handoffs da delivery.

## Procedimento

1. Atualizar apenas a documentação dos módulos que o diff tocou:
   - `AGENTS.md` (ou `GEMINI.md`): só se a estrutura ou um invariante mudou
   - `docs-ai/03-MAPA-RAPIDO-MODULOS.md`: se um módulo nasceu, sumiu ou mudou de responsabilidade
   - docstrings dos símbolos exportados alterados
2. Montar `docs-ai/deliveries/{DELIVERY_ID}/report.md` a partir do template (`docs-ai/deliveries/_template/report.md`).
3. Preencher o front matter. `require_security` reflete o sinal real da delivery: se `security-review` rodou, é `true`.
4. Preencher todas as seções obrigatórias: Contexto, Escopo, Implementação, Testes, Code Review, Final Checklist.
5. Marcar os itens do Final Checklist **apenas** com a evidência comprovada dos handoffs.
6. Registrar a delivery em `docs-ai/deliveries/INDEX.md`.
7. Rodar o validador e colar a saída:
   ```bash
   go tool pipeline-validate docs-ai/deliveries/{DELIVERY_ID}/report.md
   ```
8. Reprovado → corrigir o relatório e rodar de novo. Não seguir com reprovação.

## O que NÃO fazer

- Não documentar módulo que não foi tocado.
- Não deixar `TODO`, `TBD`, `pendente` ou `N/A` em campo obrigatório: o validador reprova fatalmente.
- Não marcar checkbox sem evidência nos handoffs.
- Não editar arquivos de código de negócio.

## Roteamento

`git-delivery`.

## Saída obrigatória

O bloco do contrato, com a saída real do `pipeline-validate` em `CONFIRMADO`.
