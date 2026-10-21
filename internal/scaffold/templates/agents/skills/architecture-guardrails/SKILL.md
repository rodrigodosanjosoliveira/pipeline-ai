---
name: architecture-guardrails
description: >-
  Define a abordagem técnica e os limites da implementação antes de qualquer código.
  Bloqueia violações de invariantes do projeto. Use após o triage, em toda demanda que toque código.
---

# Architecture / Guardrails

## Missão

Decidir **como** a demanda será implementada e, principalmente, o que **não** pode ser feito. É a skill que impede que uma mudança pequena vire uma reforma desnecessária.

## Entrada

Handoff do `pm-triage`.

## Procedimento

1. Ler os invariantes em `AGENTS.md` (ou `GEMINI.md`) e `docs-ai/01-INVARIANTES-GLOBAIS.md`.
2. Ler o código dos módulos que o triage apontou. Confirmar que existem.
3. Definir a abordagem: quais pacotes, quais interfaces, onde entra o código.
4. Definir o **Não fazer**: lista explícita de caminhos que a implementação não deve tomar. Este campo não pode ser genérico.
5. Verificar violações de invariantes:
   - dependência externa nova sem justificativa → `SUPPLY_CHAIN: sim`
   - dependência que force CGO em projeto desktop → **BLOQUEIO**
   - abstração que a stdlib já cobre → questionar
   - mudança de contrato entre Go e frontend → registrar em `RISCOS`
6. Se a abordagem exigir mudança de arquitetura, `STATUS: bloqueado` e escalonamento para aprovação humana.

## O que NÃO fazer

- Não escrever código nem pseudocódigo extenso.
- Não aprovar "a abordagem parece boa" sem dizer detalhadamente qual é.
- Não deixar o "Não fazer" genérico ("não complicar", "não quebrar nada").
- Não editar arquivos de código diretamente.

## Roteamento

`data-model` se o projeto tiver o módulo habilitado e a demanda tocar persistência. Caso contrário, `test-writer`.

## Saída obrigatória

O bloco do contrato de handoff, com `ABORDAGEM` e `NAO_FAZER` acrescentados antes de `CONFIRMADO`.
