# Orquestração Contínua e Princípio da Não-Interrupção

Este documento estabelece o modelo de execução operacional do **pipeline-ai** no ecossistema Antigravity.

---

## 1. Princípio da Não-Interrupção

Com IA, a eficiência do ciclo de desenvolvimento depende de gates determinísticos e verificações automatizadas, sem sobrecarregar o usuário humano com confirmações protocolares a cada transição de etapa.

- **A sessão principal atua como Orquestrador:** Uma vez iniciada uma demanda técnica (via `pm-triage` ou comando inicial do usuário), o orquestrador conduz o fluxo de ponta a ponta chamando os subagents (`test-writer`, `dev-implementation`, etc.), passando o contexto via contrato de handoff e validando cada gate da toolchain.
- **Transição Automática:** Quando uma etapa conclui com `STATUS: ok`, o orquestrador **NÃO DEVE** pausar para pedir autorização ou perguntar se deve prosseguir. Ele deve invocar imediatamente a etapa seguinte indicada em `PROXIMO`.
- **A intervenção humana ocorre por exceção, nunca por rotina.**

---

## 2. Condições Exclusivas de Interrupção

O orquestrador só interrompe a esteira autônoma e devolve o controle ao usuário em três situações estritas:

1. **Ambiguidade de Requisitos ou Decisão de Design:**
   - Durante `pm-triage` ou `architecture-guardrails`, se houver indefinição grave de escopo, regras de negócio conflitantes ou trade-offs de arquitetura que exijam validação humana explícita.
2. **`STATUS: bloqueado` em Qualquer Gate:**
   - Violação de invariante que não pode ser resolvida pelo subagent.
   - Reprovação ou bloqueio no `code-reviewer` ou `security-review`.
   - Falha irrecuperável de compilação ou testes no `qa-validation`.
   - Nesse caso, o orquestrador detalha a evidência do bloqueio e solicita intervenção ou decisão do usuário.
3. **Pull Request Criado (`git-delivery`):**
   - O fluxo técnico completou com sucesso, os artefatos foram validados pelo `pipeline-validate` e o Pull Request foi aberto no GitHub (`gh pr create`). A bola passa ao usuário para review e merge.

---

## 3. Protocolo de Despacho de Subagents

1. A sessão principal não edita arquivos de produção em `protected_paths` diretamente.
2. Para cada etapa técnica de código:
   - Invoca o subagent correspondente (`invoke_subagent`).
   - Aguarda a notificação de término via mensagens reativas da plataforma (sem polling manual).
   - Inspeciona o bloco de handoff e a evidência de `CONFIRMADO`.
   - Se `STATUS: ok`, despacha imediatamente a próxima skill.
