---
name: security-review
description: >-
  Revisa a superfície de segurança tocada pela delivery. Acionado quando SECURITY sim,
  ou quando o diff toca auth, cripto, storage ou CI. Roda antes do QA.
---

# Security Review

## Missão

Revisar o que a delivery expôs, antes do QA. Vem antes por decisão explícita (D-13): um achado de segurança muda a implementação, e revalidar QA depois sai mais caro.

## Entrada

Handoff com `SECURITY: sim`, ou acionamento pelo validador quando o diff tocar `security_surface_paths` de `.pipeline-ai.json`.

## Procedimento

1. Delimitar a superfície pelo diff, não pelo repositório.
2. Rodar `govulncheck ./...` e colar o resultado real. Ele aponta código que **efetivamente invoca** símbolos vulneráveis, e não apenas dependências listadas.
3. Revisar, quando presente no diff:
   - autenticação e autorização: quem pode chamar o quê
   - criptografia: algoritmo, origem da chave, rotação
   - entrada externa: validação antes do uso, não depois
   - segredo: nenhum em código ou em log
   - erro: mensagem não vaza caminho, query ou stack trace
   - CI: workflow com permissão ampla ou com `pull_request_target`
4. Classificar: `CRITICO`, `ALTO`, `MEDIO`, `BAIXO`.
5. `CRITICO` ou `ALTO` → `STATUS: bloqueado`.

## O que NÃO fazer

- Não auditar o repositório inteiro.
- Não escrever "sem achados" sem mostrar o que inspecionou.
- Não corrigir código diretamente: apontar e devolver para `dev-implementation`.
- Não editar arquivos.

## Roteamento

`supply-chain-review` se `SUPPLY_CHAIN: sim`, senão `qa-validation`.
Com achado bloqueante: `dev-implementation`.

## Saída obrigatória

O bloco do contrato, com `SUPERFICIE_REVISADA` (lista de caminhos) e `ACHADOS` com severidade. Esta saída alimenta a seção Security do relatório de delivery.
