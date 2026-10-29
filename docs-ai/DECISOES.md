# Decisões do Padrão de Novos Projetos

> Registro das decisões que definem este padrão, com o motivo de cada uma.
> Toda decisão aqui foi tomada explicitamente. Nada foi inferido.
>
> **Status:** Fase 1 (construção do conteúdo em pastas)
> **Última atualização:** 2026-09-16

---

## 1. Contexto

Este padrão nasce do kit `pipeline-agents`, que era a instância de um projeto
específico (RIS Hardening) e não um template neutro. O objetivo é extrair dele um
padrão reutilizável para iniciar novos projetos, com Go como linguagem padrão.

O padrão incorpora princípios do artigo *Why Go is an ideal language for
AI-assisted software engineering* (Google Developers Blog). A tese central do
artigo é que, com IA, o gargalo do ciclo de desenvolvimento se desloca da
**geração** para a **verificação**. Por isso este padrão investe em gates de
verificação determinísticos, não em ajudar o agent a escrever mais rápido.

O artigo também observa que refatorações iterativas de um agent **sem validação
externa degradam rapidamente**, com erros se acumulando a cada passada. Essa é a
justificativa direta para a existência dos gates descritos na seção 5.

---

## 2. Decisões de stack e ambiente

### D-01 — Go é a linguagem padrão do backend
Todo novo projeto usa Go no backend.

**Motivo:** formato único imposto pelo `gofmt`, design de linguagem que limita
abstrações complexas de propósito, e uma stdlib ampla que reduz a superfície de
dependências externas.

---

### D-02 — Frontend em React + TypeScript
Convenções mantidas: componentes funcionais com hooks, Zustand para estado
compartilhado, Tailwind sem estilos inline.

---

### D-03 — Dois formatos de projeto suportados
- **Desktop:** Go + Wails + React num binário único.
- **Web/API:** API Go separada + UI React consumindo essa API.

**Consequência:** o mapa de módulos e a estrutura de diretórios precisam de duas
variantes. O padrão não assume um formato único.

---

### D-04 — .NET sai completamente do padrão
Sem seções de exemplo, sem menções residuais.

---

### D-05 — Wails v3
**Ressalva registrada:** o v3 ainda **não é GA**. O anúncio de 02/08/2026 afirma
que é um beta, que a API de desktop está estável e que times já usam v3 em
produção, mas que é preciso testar antes de fazer deploy. O v2 segue como release
estável e continua recebendo correções.

**Motivo da escolha mesmo assim:**
1. Cross-compilação a partir do WSL2 é resolvida no v3 — ele suporta build para
   Windows, macOS e Linux a partir de qualquer SO host, e o alvo Windows não
   exige CGO por padrão.
2. O build system baseado em Taskfile é mais fácil de transformar em gate.
3. Adotar v2 agora criaria uma migração garantida no meio do caminho.

**Mitigação obrigatória:** fixar a versão exata do `wails3` em cada projeto, em
vez de depender do que estiver instalado na máquina. Betas noturnos mudam com
frequência e um build não pode variar entre projetos por causa disso.

---

### D-06 — Evitar CGO em projetos desktop
**Motivo:** é a condição que mantém a cross-compilação do WSL2 para Windows
funcionando sem Docker. Se o app exigir CGO, o build a partir de Linux passa a
precisar de Docker.

**Consequência:** o agent `architecture-guardrails` deve bloquear dependências
que forcem CGO em projetos desktop.

---

### D-07 — Ambiente: WSL2, repositórios em `~/workspace`
Repositórios ficam no filesystem do WSL2, não em `/mnt/c/...`.

**Motivo:** evita o problema de bit de execução (`chmod +x`) em drives montados do
Windows, que quebraria os hooks.

**Consequência:** build e ferramentas rodam no WSL2. A execução do binário `.exe`
gerado continua sendo no lado Windows.

---

### D-08 — GitHub + GitHub Actions
Substitui GitLab. `glab` → `gh`, MR → PR, `.gitlab-ci.yml` → workflow do Actions.

---

### D-09 — Banco varia por projeto: PostgreSQL, MongoDB ou DynamoDB
Um único banco por projeto.

**Consequência:** os agents ligados a persistência não podem ser escritos para um
banco específico. Viram módulos opcionais com conteúdo por banco (ver D-15).

---

### D-10 — Escopo de uso: pessoal e da PJ
Aprovador único. Os campos de aprovação e escalonamento permanecem apontando para
o próprio Rodrigo.

---

## 3. Decisões de governança da pipeline

### D-11 — Bloqueio de edição direta usa três camadas

**Revisado.** A versão original desta decisão colocava as deny rules como camada
de enforcement e o hook como camada pedagógica. Isso estava errado, e a leitura
da referência de permissões mostrou por quê: **regras de permissão não distinguem
sessão principal de subagent**. Um `Edit(/internal/**)` no deny bloquearia também
o `dev-implementation`. As regras `Agent(...)` controlam quais subagents podem ser
invocados, não o que cada um edita.

Além disso, a referência mostra que o hook é enforcement de verdade: um
PreToolUse que sai com código 2 interrompe a chamada **antes** de as regras de
permissão serem avaliadas, e o bloqueio vale mesmo quando uma allow rule
permitiria a operação. A própria documentação recomenda esse arranjo para
bloqueio seletivo: liberar a ferramenta na allow list e registrar um PreToolUse
que rejeita os casos específicos.

**Desenho adotado:**

| Camada | Mecanismo | Função |
|---|---|---|
| 1 | Hook `PreToolUse`, exit 2 | Bloqueio consciente de agent. Camada principal |
| 2 | `tools:` no frontmatter de cada agent | Agent sem `Edit` na lista não edita, independente do hook |
| 3 | Deny rules | Só para o que ninguém pode tocar: `.env`, segredos, credenciais |

A camada 2 é per-agent e nativa: `pm-triage` sem `Edit` em `tools:` não consegue
editar nada, sem depender de config nem de hook.

**Armadilha registrada:** a referência confirma que regras de caminho só são
consultadas para `Edit(...)` e `Read(...)`. Escrever `Write(...)`,
`NotebookEdit(...)` ou `MultiEdit(...)` é aceito mas nunca consultado, e gera
aviso na inicialização. Use sempre `Edit(...)`.

**Nota de robustez:** há relatos abertos de deny rules de `Edit` não sendo
aplicadas em algumas versões. Mais um motivo para não apoiar a proteção em uma
camada só.

**Correções obrigatórias em relação ao kit atual:**
- Ler `tool_input.file_path` do JSON de stdin — **não** `file_path` na raiz.
  O kit atual lê o campo errado, e por isso **não bloqueia nada hoje**.
- Identificar o agent pelo campo `agent_type` do JSON, presente quando a sessão
  usa `--agent` ou quando o hook dispara dentro de um subagent.
  **Não** usar `CLAUDE_FILE_PATH` nem `CLAUDE_AGENT` — essas variáveis não
  existem na referência e são usadas pela versão do guia.

**Agents autorizados a editar código:**
`dev-implementation`, `test-writer`, `code-reviewer`, `query-performance`,
`docs-update`.

---

### D-21 — Hooks moram no `settings.json` na fase 1

Um arquivo `hooks/hooks.json` separado só existe em plugin. Na fase 1, os hooks
vão na chave `hooks` do próprio `.claude/settings.json`. O `hooks.json` aparece
na fase 2 (D-20).

---

### D-22 — Estrutura de diretórios dos dois formatos

**`desktop`** (Wails v3 gera `build/`, `frontend/`, `main.go`, `Taskfile.yml`,
`go.mod`, `go.sum`; saída em `bin/`):

```
.claude/  .pipeline-ai.json  docs-ai/  bin/  build/
frontend/src/
internal/{app,domain,storage,config}/
main.go  Taskfile.yml  go.mod  go.sum
```

**`api`:**

```
.claude/  .pipeline-ai.json  docs-ai/
cmd/api/main.go
internal/{http,domain,storage,config}/
frontend/src/
migrations/        (só em banco relacional)
Taskfile.yml  go.mod  go.sum
```

**`frontend/` nos dois formatos**, e não `web/` ou `ui/` no segundo: o Wails fixa
`frontend/` no desktop, então alinhar faz `protected_paths`, contratos de agent e
documentação valerem para ambos sem bifurcação.

**Escolhas deliberadas:**
- `.github/workflows/` é superfície de segurança: alterar a CI pode desligar os
  próprios gates.
- `Taskfile.yml` é caminho protegido: é onde os gates do D-18 moram.
- `build/` é superfície de segurança no desktop: contém os manifests do Windows,
  que definem privilégios de execução.
- `docs-ai/` fica **fora** de `protected_paths`: o fluxo de delivery escreve
  relatórios o tempo todo.

---

### D-12 — Padrão de git (Opção A)
- **Branch:** `task/{code}`
- **Commit:** `{type}({scope}): {summary} [{task_code}]`

A versão divergente que estava na skill `git-delivery` é descartada.

---

### D-13 — Segurança bidirecional: Requisitos no PRE e Auditoria no POST (Security Shift-Left)

A segurança opera em um modelo bidirecional, dividida em dois momentos complementares:

1. **No PRE (`architecture-guardrails` e `test-writer`):**
   Quando a demanda toca superfície sensível (`SECURITY: sim`), o `architecture-guardrails` executa modelagem de ameaças e formaliza a seção `REQUISITOS_SEGURANCA` no contrato de handoff. Isso orienta o `test-writer` a criar testes adversariais e negativos antes de qualquer código ser escrito. A modelagem de segurança antecipada no PRE reduz drasticamente o retrabalho de dev, evitando que vulnerabilidades arquiteturais sejam codificadas para depois serem descartadas.

2. **No POST (`security-review`):**
   O `security-review` roda após o `code-reviewer` e antes do `qa-validation`. Ele realiza a auditoria cruzada do diff contra os `REQUISITOS_SEGURANCA` definidos no PRE e executa ferramentas de análise estática e de vulnerabilidades (`govulncheck ./...`). Preservar a auditoria do diff no POST garante que desvios de implementação ou vulnerabilidades acidentais não passem para o QA nem para a entrega final.

- **Security entra antes do QA:** um achado de segurança muda a implementação, e revalidar QA depois sai mais caro.
- **Release/Ops roteia para Docs** (e não o inverso).

Ordem consolidada:

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

Colchetes indicam módulos opcionais.

---

### D-14 — `require_security` passa a `false` por padrão
O triage liga a flag quando houver superfície de segurança.

**Salvaguarda obrigatória:** a flag **não pode depender só da memória do triage**.
O validador deve ligá-la automaticamente quando o diff tocar arquivos de
superfície sensível (auth, cripto, segredos, `go.mod`). Esquecer no triage não
pode deixar a delivery passar sem revisão.

---

### D-15 — Núcleo obrigatório + módulos opcionais

| Núcleo (sempre) | Módulos opcionais (o projeto ativa) |
|---|---|
| pm-triage | data-model |
| architecture-guardrails | migration-reviewer |
| test-writer | query-performance |
| dev-implementation | release-ops |
| code-reviewer | agent-auditor |
| qa-validation | context-loader |
| security-review | |
| supply-chain-review (novo) | |
| docs-update | |
| git-delivery | |

**Motivo de security e supply-chain estarem no núcleo:** em Go, ambos têm comando
determinístico associado (`govulncheck`), então o custo de mantê-los sempre ativos
é baixo e o retorno é alto.

**Motivo dos módulos de persistência serem opcionais:** dependem do banco, que
varia por projeto (D-09).

---

### D-16 — Novo agent: `supply-chain-review`
Acionado quando `go.mod` ou `go.sum` mudarem.

**Motivo (direto do artigo):** LLMs tendem a sugerir dependências desatualizadas,
sem manutenção ou até maliciosas. A stdlib do Go direciona os modelos para pacotes
mantidos oficialmente. O `govulncheck` aponta código que **efetivamente invoca**
símbolos vulneráveis, e não apenas dependências listadas.

---

## 4. Decisões de teste e verificação

### D-17 — `testing` da stdlib com table-driven tests
Testify é **opt-in por projeto**, não padrão.

**Motivo:** cada dependência que entra no padrão entra em todos os projetos
futuros. A stdlib direciona os modelos para pacotes mantidos oficialmente.

**Contra-argumento registrado:** testify é onipresente e os modelos geram
`assert.Equal` com muita fluência; com stdlib pura as asserções ficam mais
verbosas. Avaliado e aceito, porque o padrão table-driven já é o que os modelos
produzem bem em Go.

---

### D-18 — Gates da toolchain Go

| Ferramenta | Onde entra | Motivo |
|---|---|---|
| `gofmt -l` | hook `PostToolUse`, a cada edição | Barato e determinístico; corrige formatação sem gastar turno do agent |
| `go build ./...` | dev | Primeiro gate real de verificação |
| `go vet ./...` | dev + CI | |
| `go test ./...` | dev + QA + CI | |
| `govulncheck ./...` | supply-chain-review + CI | Aponta código que invoca símbolos vulneráveis |
| `go test -fuzz` | test-writer, sob gatilho | Só para parsing, validação e desserialização |
| `go fix` (modernizers) | delivery de melhoria técnica | Atualização determinística; não é gate de toda entrega |
| PGO | release-ops (opcional) | Só quando houver perfil de produção |

**Ressalva 1 — orçamento de fuzz.** Um alvo de fuzz roda indefinidamente até ser
interrompido. O gate **precisa** fixar um limite (ex.: `-fuzztime=30s`), senão o
agent trava.

**Ressalva 2 — regressão de graça.** Os seeds do corpus já rodam como testes
normais no `go test ./...` comum, mesmo sem o modo fuzz ativo.

**Ressalva 3 — linter externo.** `golangci-lint` **não** entra no padrão:
`gofmt` + `go vet` cobrem a maior parte sem adicionar dependência. Fica como
módulo opcional.

---

### D-19 — Scripts de automação reescritos em Go
O hook de bloqueio e o validador de delivery deixam de ser bash.

**Motivo decisivo:** o hook precisa ler JSON do stdin. Em bash isso exige `jq` ou
`python3` instalados — exatamente a fragilidade que quebrou a versão atual. Em Go
vira binário estático sem dependência de sistema.

**Efeito colateral desejado:** o padrão passa a ser escrito na linguagem que ele
mesmo padroniza, sendo o primeiro cliente das próprias regras.

**Custo aceito:** binário precisa ser compilado. Resolvido com um `Taskfile` de
setup, que conversa com o build system do Wails v3.

**Consequência:** `install-claude-hooks.sh` deixa de fazer sentido e sai.

---

## 5. Decisão de distribuição

### D-20 — Caminho faseado (C)

**Fase 1 — pastas.** Construir a versão corrigida e neutra no formato atual
(`.claude/`, `docs-ai/`, binários Go). Validar em um projeto real.

**Fase 2 — plugin.** Empacotar como plugin do Claude Code.

**Motivo:** validar o conteúdo antes de investir na embalagem. O trabalho da fase
1 não é descartado — os arquivos de agents, commands e skills são os mesmos nos
dois formatos; muda só o empacotamento.

**O que a fase 2 resolve:** hoje, por cópia de pastas, melhorar um agent exige
replicar a mudança manualmente em todos os projetos já criados. Como plugin, a
atualização é centralizada.

**Divisão prevista para a fase 2:**

| Vai para o plugin | Fica no projeto |
|---|---|
| Agents (núcleo + opcionais) | `CLAUDE.md` |
| Commands | `docs-ai/01-INVARIANTES-GLOBAIS.md` |
| Skills | `docs-ai/03-MAPA-RAPIDO-MODULOS.md` |
| `hooks/hooks.json` + binários | `docs-ai/deliveries/` |
| Contratos genéricos | Deny rules de permissão |

**Notas técnicas para a fase 2:**
- Apenas `plugin.json` fica em `.claude-plugin/`. Componentes na raiz do plugin.
  Se forem para dentro de `.claude-plugin/`, o plugin carrega mas eles somem.
- Hooks de plugin usam `${CLAUDE_PLUGIN_ROOT}` para apontar aos scripts.
- Comandos mudam de `/triage` para `/plugin:triage`.
- Marketplace privado via `.claude-plugin/marketplace.json` em repositório próprio.

---

## 6. Backlog de correções herdadas

Divergências encontradas no kit original que a fase 1 precisa resolver.

| # | Problema | Status |
|---|---|---|
| 1 | Hook não bloqueia nada (lê campo errado do JSON) | Resolvido por D-11 e D-19 |
| 2 | Duas versões divergentes de `validate-delivery.sh` | Resolver na reescrita (D-19) |
| 3 | `require_security: true` no template | Resolvido por D-14 |
| 4 | Contrato de handoff diverge entre doc, skill e command | Resolvido por D-24 e D-25 (.agents/rules/handoff-contract.md) |
| 5 | Padrões de git conflitantes | Resolvido por D-12 |
| 6 | Ordem do fluxo conflitante | Resolvido por D-13 |
| 7 | Faltam contratos para 6 agents | Resolvido (todas as 16 skills possuem contratos estruturados) |
| 8 | `agent-auditor` duplica skill `agent-audit`; `delivery-check` duplica `validate-delivery` | Resolvido (unificado na skill nativa `agent-auditor`) |
| 9 | `INDEX.md` registra delivery inexistente; caminhos errados no prompt | Resolvido (`INDEX.md` limpo) |
| 10 | Allowlist mais permissiva que as regras dos agents (`git add *`, `make *`) | Resolvido por D-11 e pre-push hook |
| 11 | Seção Go do guia não cobre a toolchain | Resolvido por D-18 |

---

## 7. Pendências resolvidas da Fase 1

1. **Fallback de roteamento quando `release-ops` está desativado:**
   Resolvido: `qa-validation` assume o roteamento e direciona diretamente para `docs-update` quando `release-ops` não estiver habilitado em `.pipeline-ai.json`.

2. **Salvaguarda do validador (D-14 estrito):**
   Resolvido: falha ao calcular o `git diff` reprova fatalmente com `ERRO`. Para ignorar intencionalmente, é obrigatório passar a flag `-no-git`.

3. **Permissão de escrita do `code-reviewer`:**
   Resolvido: permissão de edição removida de `editor_agents` em `.pipeline-ai.json` e proibição explícita na skill `code-reviewer` (D-04 / D-11).

---

## 8. Migração para Antigravity / Gemini (Opção 1B) e Versão Estável v0.3.0

### D-20 — Migração total para o ecossistema Antigravity

- **Estrutura nativa:** `.claude` e `CLAUDE.md` substituídos por `.agents/` (`hooks.json`, `rules/`, `skills/`) e diretrizes raiz `AGENTS.md` e `GEMINI.md`.
- **Skills nativas:** 16 skills do Antigravity implementadas (`SKILL.md` com YAML frontmatter `name` e `description`):
  - *Núcleo (10):* `pm-triage`, `architecture-guardrails`, `test-writer`, `dev-implementation`, `code-reviewer`, `security-review`, `supply-chain-review`, `qa-validation`, `docs-update`, `git-delivery`.
  - *Módulos opcionais (6):* `release-ops`, `data-model` (Postgres, Mongo, DynamoDB), `migration-reviewer`, `query-performance`, `context-loader`, `agent-auditor`.
- **Hook PreToolUse em Go:** `cmd/pipeline-hook` adaptado para o protocolo de ciclo de vida do Antigravity, processando `toolCall` e emitindo resposta JSON no `stdout` (`{"decision": "allow" | "deny", "reason": "..."}`), com suporte a retrocompatibilidade.
- **Módulos opcionais em Go (D-15):** Schema de `.pipeline-ai.json` atualizado com `enabled_modules ([]string)`, método `c.IsModuleEnabled(name)` e catálogo `AllowedModules`.
- **Scaffolding de Projetos (`cmd/pipeline-init`):** Criação de novos projetos em 1 comando com templates embutidos via `//go:embed all:templates`.
- **Automação e CI:** Padronização com `Taskfile.yml` e workflow no GitHub Actions (`.github/workflows/ci.yml`).
- **Superfície protegida:** `go.mod` e `go.sum` incluídos em `protected_paths` em todos os templates de configuração.
- **Proteção de branch local:** Script `.githooks/pre-push` configurado para impedir pushes diretos na `main` em repositórios privados, garantindo entrega 100% via Pull Request.

---

## 9. Consolidação Arquitetural da v0.6.0 (Contraditório Técnico)

### D-21 — Modelo Híbrido de Autorização e Delivery Ativa (C3 / DEL-010)

A governança de edição no workspace adota um modelo híbrido e resiliente:
1. **Autorização Híbrida:** A escrita em caminhos protegidos é autorizada se o agente for um `editor_agent` nominal OU se houver uma delivery ativa válida (`state.IsActive`), eliminando fragilidades de perda de identidade de agentes em runtimes efêmeros (como Antigravity e sessões de pair programming).
2. **Ciclo de Vida com TTL:** Toda entrega possui ciclo temporal explícito (padrão 7 dias) persistido em `.pipeline-state/active-delivery.json`. Entregas expiradas são bloqueadas deterministicamente (`ErrExpired`).
3. **Caminhos Imutáveis (`immutable_paths`):** Arquivos essenciais de governança, CI/CD ou integridade arquitetural podem ser marcados como incondicionalmente imutáveis, bloqueando qualquer edição automatizada independentemente do agente ou status de delivery.

---

### D-22 — Security Shift-Left Bidirecional com Modelagem Prévia e Pós-Auditoria (C8 / DEL-011)

A segurança é tratada como um processo contínuo e bidirecional em dois momentos obrigatórios do pipeline:
1. **No PRE (`architecture-guardrails` e `test-writer`):** Quando `SECURITY: sim`, o guardrails executa modelagem formal de ameaças e emite `REQUISITOS_SEGURANCA` no contrato de handoff, orientando o `test-writer` a elaborar testes adversariais e de borda antes de qualquer código de produção ser escrito.
2. **No POST (`security-review`):** Auditoria cruzada do diff pós-implementação contra os requisitos contratuais do PRE e checagem de análise estática e vulnerabilidades de símbolos com `govulncheck ./...`.

Isso elimina retrabalhos onde vulnerabilidades eram descobertas apenas no final do ciclo.

---

### D-23 — Governança Flexível de Entregas, Regexes e Relatórios Customizados (C10/C11 / DEL-012)

O pipeline suporta personalização corporativa sem perder o rigor de validação:
1. **Padrões Customizáveis de Identificadores:** Suporte declarativo a regexes para `delivery_id_pattern` e `task_code_pattern` no `.pipeline-ai.json`, permitindo identificadores como Jira, Wrike ou padrões proprietários (`RIS-NNN`, `PLAT-NNN`).
2. **Seções Obrigatórias Customizadas (`custom_sections`):** Projetos podem declarar seções adicionais no `report.md` (como `Tenant Isolation`, `Frontend Handoff`, `Checkpoints`) que se tornam gates mandatórios de validação no `pipeline-validate`.
3. **Índice Central Opcional (`require_index`):** Suporte a desativar a obrigatoriedade do `INDEX.md` (`require_index: false`) para repositórios com fluxos git descentralizados.

---

### D-24 — Sensores Estruturados em JSON e Autoteste Negativo contra Falsos Positivos (C13/C15 / DEL-013)

A automação de qualidade e testes adota salvaguardas estritas contra aprovações indevidas:
1. **Sensores Estruturados:** Implementação de `pipeline-ai sensor` e flags `-json` no `pipeline-validate`, expondo resultados tipados e estruturados (`fmt`, `vet`, `test`) para agentes de IA, IDEs e CI.
2. **Salvaguarda contra Falsos Positivos (Autoteste Negativo):** Execuções com zero testes (`tests_run == 0` ou pacotes desprovidos de testes) são tratadas como falha fatal determinística (`Overall: FAIL`), prevenindo aprovação acidental de comandos sem cobertura.
3. **Evidência Real de Testes:** O validador rejeita relatórios cuja seção `# Testes` contenha saídas vazias, declarações genéricas ou apenas `[no test files]`.

---

### D-25 — Invariantes, Critérios de Reversibilidade/Rollback e Gestão Dinâmica de Módulos (C14/C16 / DEL-014)

A prontidão operacional e a integridade de módulos são formalizadas:
1. **Obrigatoriedade de Reversibilidade/Rollback (`require_rollback`):** Entregas críticas exigem formalmente a seção `# Rollback Plan` / `# Reversibilidade`, detalhando procedimentos de contingência e reversão antes do merge. Tentativas de evasão (`N/A`, `não se aplica`) são rejeitadas deterministicamente.
2. **Gestão Segura de Módulos Opcionais:** A CLI (`pipeline-ai module list|enable|disable`) e a MCP tool `pipeline_manage_modules` gerenciam o catálogo de módulos com validação de esquema, impedindo corrupção manual do arquivo `.pipeline-ai.json`.
3. **Checkpoints de Integridade no Diagnóstico:** O utilitário `doctor` valida e audita o estado dos módulos ativos e conformidade com as diretivas do projeto.

---

## 10. Consolidação Arquitetural da v0.7.0 (Orquestração Autônoma, Runner e Observabilidade)

### D-26 — Princípio da Não-Interrupção e Orquestração Autônoma Contínua (DEL-016)

Eliminação da fadiga de confirmação em esteiras de agentes através da **Regra 7 de Orquestração**:
1. **Fluxo Contínuo:** Uma vez iniciada a demanda pelo `pm-triage`, as transições entre skills ocorrem sem interrupções manuais enquanto o status dos gates for `ok`.
2. **Intervenção Humana Estrita:** Reservada unicamente para:
   - Ambiguidade crítica de negócio na triagem.
   - Bloqueio técnico irrecuperável (`STATUS: bloqueado`).
   - Revisão final do Pull Request gerado.
3. **Isolamento de Responsabilidade:** O orquestrador na sessão principal coordena transições e não edita código de negócio diretamente, garantindo que todo código passe pelos gates de TDD e Code Review.

---

### D-27 — Pipeline Runner Determinístico em Cascata (DEL-017)

Consolidação da verificação multi-estágio em um único comando de alta performance:
1. **Tríade Determinística:** O comando `pipeline-ai run` executa a sequência `Doctor` → `Sensor` → `Validate` de forma atômica.
2. **Auto-Descoberta de Delivery:** O runner identifica a delivery ativa no repositório (`state.Get` ou inspeção do `report.md`), eliminando a necessidade de passar parâmetros manuais repetitivos.
3. **Integração MCP Nativa:** Disponibilização da ferramenta `pipeline_run_pipeline` para que agentes invoquem o pipeline com saída estruturada JSON-RPC.

---

### D-28 — Parser e Validação de Contratos de Handoff com Fuzz Testing (DEL-018)

Formalização do protocolo de comunicação entre agentes para mitigar alucinações e perda de contexto:
1. **Contrato Estruturado:** Cabeçalho padrão (`DE`, `PARA`, `TAREFA`, `STATUS`, `ESCOPO`, `EVIDÊNCIAS`) obrigatório em todas as transições de skills.
2. **Validador Determinístico:** Utilitário `pipeline-ai handoff validate` que verifica completude, valores canônicos e integridade do bloco.
3. **Robustez por Fuzzing:** O parser de handoffs é submetido a testes contínuos de Fuzzing na biblioteca padrão de Go, garantindo imunidade a panics diante de saídas malformadas de LLMs.

---

### D-29 — Detecção e Injeção Transparente de Cross-Compilação Desktop Wails v3 (DEL-019)

Suporte nativo a ambientes heterogêneos de desenvolvimento:
1. **Invariante Desktop sem CGO:** Projetos com `project_type: "desktop"` (Wails v3) exigem compilação cruzada Windows a partir de hosts Linux/WSL2 sem dependências CGO (`pkg-config` / GTK).
2. **Injeção Transparente pelo Sensor:** O motor de sensores em `internal/sensor/sensor.go` detecta automaticamente projetos desktop e injeta `CGO_ENABLED=0 GOOS=windows GOARCH=amd64` nas invocações de `go build` e `go test`, garantindo que agentes executando em containers Linux operem 100% no verde.

---

### D-30 — Motor de Telemetria e Métricas DORA para Agentes de IA (DEL-020)

Gestão quantitativa de desempenho e confiabilidade da esteira autônoma:
1. **As Quatro Métricas DORA em Velocidade de Máquina:**
   - *Deployment Frequency:* volume diário de micro-entregas validadas.
   - *Lead Time for Changes:* latência da issue ao PR em minutos.
   - *Change Failure Rate:* taxa real de falhas/bloqueios nos gates determinísticos.
   - *Time to Restore Service (MTTR):* tempo de autorrecuperação da esteira.
2. **Zero Dependências Externas:** Implementação em Go 1.24 puro (stdlib), com persistência atômica fail-safe em `.pipeline-state/metrics.json`.
3. **Fuzz Testing & Protocolo MCP:** Parser blindado com mais de 1.1 milhão de iterações de Fuzzing sem falhas e ferramenta MCP `pipeline_get_metrics` para autoavaliação contínua da IA.

---

## 11. Consolidação Arquitetural da v0.8.0 (Generative UI e Plugin Nativo Antigravity)

### D-31 — Dashboard Generative UI Autocontido e Responsivo (DEL-022)

Observabilidade visual unificada sem dependências de servidores ou frameworks externos:
1. **Conformidade Estrita com Generative UI:** O gerador HTML (`internal/dashboard`) adota estritamente o Tailwind CSS oficial da runtime Antigravity (`https://www.gstatic.com/antigravity/web/dev/tailwindcss.min.js`), tipografia com piso mínimo em 12px (`text-xs`) e tokens semânticos de CSS do host.
2. **Autonomia e Portabilidade:** O dashboard é compilado em um documento HTML5 autocontido (com Vanilla JS embutido), eliminando necessidade de processos daemon em background e permitindo tanto embedding inline com `<agent-embed>` quanto visualização local no navegador (`pipeline-ai dashboard -output`).
3. **Visão Holística em 4 Abas:** Integração simultânea das métricas DORA, da esteira visual das 16 skills em pipeline stepper, do histórico tabular de entregas do `INDEX.md` e dos pilares de governança.

---

### D-32 — Empacotamento Oficial e Distribuição como Plugin do Antigravity (DEL-023)

Transformação do toolkit em uma extensão nativa de primeira classe no ecossistema Antigravity:
1. **Bundle Unificado Declarativo:** Distribuição oficial em `plugin/` contendo `plugin.json` (schema Antigravity), servidor MCP integrado (`mcp_config.json`), hooks de ciclo de vida (`hooks.json`), regras consolidadas (`rules/AGENTS.md`) e catálogo completo das 16 skills especializadas.
2. **Instalação Determinística em 1 Comando:** Implementação do subcomando CLI `pipeline-ai plugin install` para instalação automatizada e idempotente no diretório de plugins do usuário (`~/.gemini/config/plugins/pipeline-ai`).
3. **Validação Estrutural e Fuzzing:** Utilitário `pipeline-ai plugin verify` e analisador sintático de manifestos blindado por Fuzz Testing contínuo.

---

## 12. Sensoriamento Semântico e Decisões Calibradas com Jev

### D-33 — Sensoriamento Semântico e Decisões Calibradas com Jev (DEL-025)

Integração de sensoriamento semântico determinístico no pipeline através do pacote `internal/sensor/semantic`:
1. **Modelos "System One" e Decisões Calibradas Tipadas:** Uso de modelos "System One" (Jev) para classificação calibrada e tipada (`NoulDecision`, `ChoiceDecision`, `ScoreDecision`), substituindo a geração estocástica e não-determinística de texto livre por inferências rápidas com scores de confiança calibrados e thresholds configuráveis.
2. **Preservação Estrita das Invariantes Arquiteturais:**
   - Implementação em Go 1.24+ puro utilizando exclusivamente a standard library (`net/http`, `encoding/json`, `context`, etc.).
   - Zero dependências externas adicionadas em `go.mod`.
   - Política fail-closed graciosa: clientes não configurados, indisponíveis ou requisições com confiança abaixo do threshold falham de forma segura (`ErrInsufficientConfidence`, `ErrUnauthorized`, `ErrServerFailure`).
3. **Hardening de Segurança e Resiliência:**
   - Proteção de credenciais (SEC-01): sanitização e redação automática de APIKey em mensagens de erro (`[REDACTED]`).
   - Mitigação de DoS e exaustão de memória (SEC-02): limitação estrita de leitura de payload HTTP via `io.LimitReader` com `MaxBodyBytes` (padrão 10MB) retornando `ErrResponseTooLarge`.
   - Fuzz testing contínuo (`FuzzParseNoulResponse`): garantia de imunidade contra panics, payloads malformados ou entradas randômicas no unmarshaling do JSON.





