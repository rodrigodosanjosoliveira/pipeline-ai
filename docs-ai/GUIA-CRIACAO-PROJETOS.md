# Guia de Criação de Novos Projetos (pipeline-ai)

> Roteiro prático e determinístico para inicializar novos projetos com o padrão **pipeline-ai** integrado nativamente ao ecossistema **Antigravity**.

---

## 1. Pré-requisitos no WSL2 / Linux

- **Go 1.24+**: Obrigatório para suportar a diretiva `tool` no `go.mod`.
- **Git** e **GitHub CLI (`gh`)**: Autenticado para submissão de Pull Requests.
- **Node.js / npm**: Para projetos com frontend React / Wails.

---

## 2. Como Criar um Novo Projeto (1 Comando)

Crie o diretório do novo projeto e execute o `pipeline-init` diretamente via `go run`:

```bash
mkdir ~/workspace/meu-novo-app
cd ~/workspace/meu-novo-app

go run github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-init@latest
```

O utilitário iniciará o **modo interativo**, perguntando:
1. **Nome do projeto** (ex: `meu-novo-app`)
2. **Formato**:
   - `1) desktop` (Go + Wails v3 + React em binário único)
   - `2) api` (Servidor Go API + UI React)
3. **Banco de dados**:
   - `1) PostgreSQL`
   - `2) MongoDB`
   - `3) DynamoDB`
4. **Caminho do módulo Go** (ex: `github.com/rodrigodosanjosoliveira/meu-novo-app`)

---

## 3. Criação Não Interativa (Via Flags)

Para scripts de automação ou inicialização rápida sem perguntas:

### Exemplo Desktop (Wails v3 + PostgreSQL):
```bash
go run github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-init@latest \
  -name=gestor-desktop \
  -type=desktop \
  -db=postgres \
  -module=github.com/rodrigodosanjosoliveira/gestor-desktop
```

### Exemplo API (Go API + MongoDB):
```bash
go run github.com/rodrigodosanjosoliveira/pipeline-ai/cmd/pipeline-init@latest \
  -name=catalogo-api \
  -type=api \
  -db=mongo \
  -module=github.com/rodrigodosanjosoliveira/catalogo-api
```

---

## 4. O que é Gerado Automaticamente

Ao executar o comando, o toolkit entrega o projeto 100% pronto para uso:

| Componente | O que é configurado |
|---|---|
| **Git & Hooks** | `git init` executado, hook `.githooks/pre-push` instalado (bloqueando pushes diretos na `main`) e `.gitignore` configurado para Go e credenciais. |
| **Go & Toolchain** | `go.mod` inicializado com diretivas `tool` apontando para `pipeline-hook` e `pipeline-validate`. |
| **Antigravity Hooks** | `.agents/hooks.json` configurado com `PreToolUse` (bloqueio ativo de caminhos protegidos) e `PostToolUse` (`gofmt -w .`). |
| **Skills Nativas** | Todas as skills do pipeline instaladas em `.agents/skills/` (`pm-triage`, `architecture-guardrails`, `test-writer`, `dev-implementation`, `code-reviewer`, `security-review`, `supply-chain-review`, `qa-validation`, `release-ops`, `docs-update`, `git-delivery`, `agent-auditor`). |
| **Regras de Governança** | `AGENTS.md` e `GEMINI.md` com placeholders preenchidos; regras contextuais em `.agents/rules/`. |
| **Governança Go** | `.pipeline-ai.json` configurado de acordo com o formato escolhido (`desktop` ou `api`). |
| **Documentação Raiz** | `docs-ai/01-INVARIANTES-GLOBAIS.md`, `docs-ai/03-MAPA-RAPIDO-MODULOS.md`, `docs-ai/DECISOES.md` e estrutura de deliveries com `INDEX.md`. |
| **Taskfile & Código Inicial** | `Taskfile.yml` com tarefas determinísticas (`build`, `test`, `vet`, `validate`) e ponto de entrada Go inicial (`main.go`). |

---

## 5. Como Iniciar o Desenvolvimento no Antigravity

1. **Abra o projeto no Antigravity IDE ou Antigravity:**
   ```bash
   cd ~/workspace/meu-novo-app
   ```
   No Antigravity IDE: **File → Open Folder** → selecione a pasta do WSL2.

2. **Inicie sua primeira demanda:**
   - No chat do Antigravity, acione a skill `pm-triage`:
   ```text
   Olá! Gostaria de abrir uma nova demanda técnica para implementar a funcionalidade X.
   ```
   - O `pm-triage` lerá o `AGENTS.md`, atribuirá o `DEL-001` no `docs-ai/deliveries/INDEX.md` e conduzirá as etapas sequenciais com gates determinísticos comprovados!
