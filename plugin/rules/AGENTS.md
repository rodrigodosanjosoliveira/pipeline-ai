# Pipeline-AI: Regras e Governança do Agente

Toolkit de governança determinística, orquestração autônoma, runner e telemetria DORA para múltiplos agentes de IA.

---

## Como o trabalho acontece aqui

Este projeto adota o padrão **pipeline-ai** integrado nativamente ao ecossistema **Antigravity**. **Você, na sessão principal, atua como o Orquestrador da esteira e não edita código diretamente.** O hook de ciclo de vida `PreToolUse` bloqueia edições em caminhos protegidos quando a chamada não vem de um agent autorizado em `.pipeline-ai.json`.

Com IA, o gargalo do desenvolvimento não é gerar código, mas verificá-lo. Os agents e skills existem para que cada etapa tenha um gate determinístico e evidências comprovadas.

**Princípio da Não-Interrupção:** Uma vez iniciada uma demanda, o orquestrador conduz as transições entre skills de ponta a ponta sem pausas protocolares para confirmação quando o status for `ok`. A intervenção humana só é solicitada em caso de ambiguidade crítica de negócio/design, falha bloqueante (`STATUS: bloqueado`) ou ao concluir com o Pull Request aberto.

**Toda demanda técnica começa na skill `pm-triage`.** O triage define o escopo e roteia para as próximas etapas.

## Fluxo de Execução

```text
pm-triage
  → architecture-guardrails [com REQUISITOS_SEGURANCA quando SECURITY: sim]
  → [data-model]
  → test-writer
  → dev-implementation
  → code-reviewer
  → [migration-reviewer] [query-performance]
  → security-review        (auditoria do diff vs requisitos do PRE)
  → supply-chain-review    (quando go.mod/go.sum mudarem)
  → qa-validation
  → [release-ops]
  → docs-update
  → git-delivery
```

Colchetes representam módulos opcionais ativados via configuração ou sob demanda.

## Gates Obrigatórios da Toolchain

| Comando | Quando | Motivo |
|---|---|---|
| `gofmt -l .` | a cada edição (automático via hook `PostToolUse`) | Formatação determinística sem gastar turnos |
| `go build ./...` | dev | Validação de compilação do projeto |
| `go vet ./...` | dev e CI | Análise estática da stdlib |
| `go test ./...` | dev, QA e CI | Execução da suíte completa de testes |
| `govulncheck ./...` | supply-chain-review e CI | Detecção de símbolos vulneráveis invocados |
| `go test -fuzz=. -fuzztime=30s` | parsing, validação e desserialização | Fuzz sempre com `-fuzztime` explícito |

## Invariantes

1. **Go 1.24+**: A diretiva `tool` no `go.mod` não é aceita por versões anteriores.
2. **Testes com a stdlib `testing`**, table-driven. Testify só se o projeto optar explicitamente.
3. **Sem CGO** em projeto desktop para manter cross-compilação WSL2 -> Windows.
4. **Dependência externa precisa de justificativa**. A stdlib cobre a maior parte. Mudanças em `go.mod` disparam `supply-chain-review`.
5. **Nada de segredos em código**. Credenciais sempre via variáveis de ambiente.
6. **Proteção de código ativo**: Edições diretas em caminhos protegidos fora de agentes autorizados são barradas em tempo de execução via hook `PreToolUse`.
7. **Formatação automática determinística**: Todo arquivo editado passa pelo `gofmt -w` no hook `PostToolUse`.

## Contrato de Handoff

Toda transição entre agentes deve ser registrada estritamente no bloco padrão:

```text
=== HANDOFF CONTRACT ===
DE: <agent-origem>
PARA: <agent-destino>
DELIVERY: <DEL-ID>
STATUS: ok | bloqueado
PROXIMO: <proxima-acao>
ESCOPO: <arquivos-ou-pacotes>
NAO_FAZER: <restricoes>
CONFIRMADO: <evidencias-ou-gates>
========================
```
