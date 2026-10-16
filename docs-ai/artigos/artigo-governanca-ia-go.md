# O gargalo da IA não é gerar código, é governar: Como construí um pipeline determinístico com Go e Google Antigravity

*Por Rodrigo Oliveira — Engenheiro de Software & Pós-Graduando em Engenharia de IA Aplicada*

---

Quem já colocou agentes autônomos de IA (como Claude, Devin ou Antigravity) para codificar em projetos reais conhece a sensação: em um minuto o agente implementa uma função brilhante; no minuto seguinte, ele reescreveu o `go.mod`, editou um arquivo fora do escopo, alucinou testes inexistentes e tentou fazer commit direto na branch `main`.

A verdade que a indústria está começando a encarar é simples:

> **O gargalo do desenvolvimento com IA não é a velocidade de geração de código. É a capacidade de verificar, restringir e governar o que foi gerado.**

Na minha jornada na pós-graduação em **Engenharia de IA Aplicada**, decidi transformar essa dor em um framework prático de engenharia de software: o **Pipeline-AI**. 

Neste artigo, compartilho como estruturei uma esteira de agentes autônomos baseada em **Zero-Trust com hooks em Go**, TDD rigoroso e separação estrita de responsabilidades integrada nativamente ao ecossistema do **Google Antigravity**.

---

## O Problema: Por que "Prompt Engineering" não é governança

A abordagem ingênua para controlar agentes de IA é adicionar instruções textuais no prompt:
> *"Por favor, não edite arquivos fora de `internal/user` e não commite na `main`."*

Isso falha por um motivo matemático fundamental: **LLMs são probabilísticos, mas a engenharia de software exige garantias determinísticas**.

Se houver 1% de chance de o modelo ignorar a instrução sob um contexto extenso, esse 1% vai acontecer na sua branch de release. A governança não pode viver no prompt do agente; ela precisa viver **no ambiente de execução** (no sistema operacional).

---

## 1. A Camada de Defesa: Hooks de Ciclo de Vida (`PreToolUse`)

A primeira decisão de arquitetura foi implementar o princípio do menor privilégio (*Least Privilege*): a sessão principal do chat **não tem permissão para editar código diretamente**.

O Google Antigravity oferece hooks de ciclo de vida, em especial o evento **`PreToolUse`**. Toda vez que o modelo invoca uma ferramenta como `write_to_file` ou `replace_file_content`, o hook intercepta a chamada **antes** que qualquer byte seja tocado no disco.

Construí o binário **`pipeline-hook`** em Go 1.24 puro (sem nenhuma dependência externa além da biblioteca padrão). Ele recebe o payload JSON no `stdin`, identifica qual subagente está operando e qual arquivo está sendo tocado:

```go
// Trecho simplificado da lógica de interceptação do pipeline-hook
func run(stdin io.Reader, stdout, stderr io.Writer) int {
    var in hookInput
    if err := json.Unmarshal(raw, &in); err != nil {
        return replyAllow(stdout)
    }

    // Se a ferramenta tentar modificar o disco
    if editTools[in.ToolName] {
        target := in.FilePath()
        
        // Verifica se o caminho é protegido (.pipeline-ai.json)
        if cfg.IsProtected(target) && !cfg.IsEditorAgent(in.AgentType) {
            msg := fmt.Sprintf(
                "BLOCKED: O agente '%s' não tem permissão para editar '%s'. "+
                "Apenas os agentes autorizados [%s] podem realizar alterações neste diretório.",
                in.AgentType, target, strings.Join(cfg.EditorAgents, ", "),
            )
            return replyDeny(msg, stdout)
        }
    }
    return replyAllow(stdout)
}
```

Se o agente `pm-triage` ou a sessão principal tentar mexer em `internal/auth/token.go`, o hook retorna `deny` no nível do processo. Não há negociação: **a ferramenta simplesmente não executa**.

---

## 2. A Esteira de 16 Skills: Especialização vs. Agente Monolítico

Um dos erros mais comuns com IA é usar um "agente sabe-tudo". Quando um único prompt tenta triar o negócio, planejar a arquitetura, escrever testes, implementar o código e fazer o review, a dispersão de contexto é imensa.

No **Pipeline-AI**, decompusemos o ciclo de desenvolvimento em um pipeline modular de **16 skills especializadas**:

```text
1. pm-triage              → Delimita o escopo da issue e atribui DEL-ID
2. architecture-guardrails → Define limites técnicos e bloqueia violações de invariantes
3. [data-model]           → Modela entidades e esquemas (PostgreSQL, Mongo, DynamoDB)
4. test-writer            → Escreve cenários de TDD e alvos de Fuzzing (Red)
5. dev-implementation     → Codifica estritamente até os testes passarem (Green)
6. code-reviewer          → Analisa o diff identificando bloqueios vs sugestões
7. [migration-reviewer]   → Revisa DDLs, locks e planos de rollback
8. [security-review]      → Acionado automaticamente se tocar auth, crypto ou storage
9. [supply-chain-review]  → Audita alterações em go.mod e invoca govulncheck
10. qa-validation         → Executa a suíte de testes de ponta a ponta e checa invariantes
11. [release-ops]         → Avalia SemVer, tags e prontidão de manifesto
12. docs-update           → Monta o relatório de delivery auditável
13. git-delivery          → Cria a branch, commit convencional e Pull Request
```

*(Colchetes indicam módulos opcionais, ativados via configuração de projeto).*

Cada agente sabe exatamente o seu limite: o `code-reviewer` não edita código (D-04); o `dev-implementation` só entra em ação depois que o `test-writer` já criou os testes que falharam.

---

## 3. TDD Estrito como Antídoto à Alucinação

Na prática, a melhor especificação para um agente de IA é um teste unitário que falha:

1. O `test-writer` recebe os invariantes da arquitetura e escreve os testes na biblioteca padrão `testing`, usando *table-driven tests*.
2. O agente roda `go test ./...` e **comprova a falha** (fase *Red* do TDD).
3. O `dev-implementation` assume. Seu objetivo agora não é "adivinhar o que o desenvolvedor queria", mas sim **fazer os testes passarem**.
4. Quando `go test` fica verde (*Green*) e `gofmt` passa, a implementação termina. Sem código morto, sem especulações desnecessárias.

---

## 4. O Portão de Entrega: `pipeline-validate`

Antes de qualquer Pull Request ser aberto, uma ferramenta em Go faz o double-check de conformidade:

```bash
go tool pipeline-validate docs-ai/deliveries/DEL-011/report.md
```

O validador reprova a entrega se:
- O checklist técnico contiver caixas desmarcadas.
- O texto do relatório tiver marcadores como `TODO`, `FIXME`, `TBD` ou expressões vazias de preenchimento.
- O `git diff` real do repositório tocar arquivos da superfície de segurança (`security_surface_paths`) sem que a seção de segurança tenha sido formalmente preenchida e aprovada.

Se o validador falhar, o script bloqueia o avanço. O código nem sequer sobe para o GitHub.

---

## 5. Validação em um Projeto Real

Testamos essa arquitetura em um projeto desktop real construído em Go e Wails:
- **Zero CGO:** O pipeline garantiu que nenhuma biblioteca externa quebrasse a compilação cruzada WSL2 (Linux) → Windows.
- **Segurança Proativa:** Quando a entrega tocou o módulo `internal/storage/sqlite/`, o pipeline detectou automaticamente a superfície de persistência, exigindo auditoria anti-SQL Injection e análise via `govulncheck`.
- **Governança de Git:** Pushes diretos na `main` foram bloqueados localmente (`.githooks/pre-push`) e remotamente via GitHub Rulesets, forçando a abertura do PR e a execução limpa dos workflows de CI.

---

## Lições para Engenheiros de IA

1. **Trate agentes de IA como estagiários hiper-rápidos com acesso ao terminal:** Dê ferramentas poderosas, mas cerque-os com testes automatizados e hooks de bloqueio no sistema operacional.
2. **Determinismo supera probabilidade:** Sempre que possível, substitua instruções em linguagem natural por executáveis compilados (`go tool`) e linters determinísticos.
3. **A separação de papéis reduz o custo de contexto:** Agentes menores com papéis cirúrgicos gastam menos tokens, cometem menos erros e são infinitamente mais fáceis de auditar.

---

O toolkit completo, incluindo a CLI de scaffolding `pipeline-init`, os hooks e o catálogo de 16 skills, é open-source (MIT):
👉 **[github.com/rodrigodosanjosoliveira/pipeline-ai](https://github.com/rodrigodosanjosoliveira/pipeline-ai)**

Como você tem lidado com governança e validação de código gerado por IA no seu dia a dia? Vamos trocar experiências nos comentários! 💬👇
