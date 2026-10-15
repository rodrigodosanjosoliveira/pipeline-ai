---
name: supply-chain-review
description: >-
  Audita mudanças em go.mod e go.sum. Acionado sempre que dependências mudarem.
  Avalia se a dependência é necessária, mantida e segura.
---

# Supply Chain Review

## Missão

Avaliar toda dependência que entra. Modelos de linguagem tendem a sugerir pacotes desatualizados, sem manutenção ou desnecessários, enquanto a stdlib do Go direciona para pacotes mantidos oficialmente.

## Entrada

Handoff com `SUPPLY_CHAIN: sim`, ou diff tocando `go.mod` / `go.sum`.

## Procedimento

1. `git diff origin/main...HEAD -- go.mod go.sum` para ver exatamente o que entrou.
2. Para **cada** dependência nova, responder às quatro perguntas:
   - A stdlib resolve isso? Se sim, a dependência é injustificada.
   - Está mantida? Último release, issues abertas, atividade recente.
   - Quantas dependências transitivas ela traz?
   - A licença é compatível?
3. `go mod verify` e colar o resultado.
4. `govulncheck ./...` e colar o resultado.
5. Conferir se a versão é explícita (sem `latest`, sem pseudo-versão de commit sem justificativa).
6. Verificar se alguma dependência força CGO. Em projeto desktop, isso é **BLOQUEIO**: quebra a cross-compilação do WSL2 para Windows sem Docker.

## Critério de bloqueio

- Vulnerabilidade que o `govulncheck` aponta como invocada
- Dependência sem release há mais de 2 anos, sem justificativa forte
- Dependência que a stdlib cobre satisfatoriamente
- CGO em projeto desktop
- Licença incompatível

## O que NÃO fazer

- Não aprovar por popularidade superficial.
- Não rodar `go get` para corrigir: apontar e devolver para `dev-implementation`.
- Não editar arquivos de código ou dependências diretamente.

## Roteamento

`qa-validation`. Com bloqueio: `dev-implementation`.

## Saída obrigatória

O bloco do contrato, com `DEPENDENCIAS_NOVAS` (nome, versão, justificativa, veredicto) e a saída real de `go mod verify` e `govulncheck`.
