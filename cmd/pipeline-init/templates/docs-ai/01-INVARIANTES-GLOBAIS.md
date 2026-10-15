# Invariantes Globais do Projeto

> Este documento estabelece as regras invioláveis de arquitetura, código, segurança e fluxo de trabalho.
> Nenhuma decisão técnica ou implementação pode violar estes invariantes sem aprovação explícita do arquiteto.

---

## 1. Stack e Ferramental

1. **Go 1.24+ obrigatório:**
   - O projeto utiliza a diretiva `tool` no `go.mod` para gerenciar ferramentas de desenvolvimento determinísticas (substituindo o antigo padrão `tools.go`). Versões anteriores do Go não são suportadas.
2. **Testes com a stdlib `testing`:**
   - Padrão **table-driven tests**.
   - Asserções feitas com construções idiomáticas da biblioteca padrão (`t.Errorf`, `t.Fatalf`).
   - `testify` ou outros frameworks de asserção são estritamente *opt-in* e precisam de justificativa formal.
3. **Sem CGO em projetos Desktop:**
   - O binário desktop (Go + Wails v3 + React) deve compilar sem CGO (`CGO_ENABLED=0`).
   - Isso garante a cross-compilação direta a partir do ambiente WSL2 (Linux) gerando o binário Windows `.exe` sem necessidade de Docker ou toolchains cruzadas externas.
   - O agente `architecture-guardrails` deve barrar qualquer dependência externa que exija CGO.
4. **Wails v3 com versão fixada:**
   - Em projetos desktop, a versão exata do `wails3` deve ser fixada no `go.mod` e no `Taskfile.yml`. Por ser uma versão em desenvolvimento, *builds* não podem variar entre ambientes.
5. **Frontend em React + TypeScript:**
   - Componentes funcionais com Hooks.
   - Gerenciamento de estado compartilhado via **Zustand**.
   - Estilização com **Tailwind CSS** (sem estilos inline ou CSS solto).

---

## 2. Dependências e Supply Chain

1. **Dependência externa exige justificativa:**
   - A biblioteca padrão do Go cobre a maior parte das necessidades. Qualquer inclusão de pacote externo em `go.mod` deve ser justificada no handoff de arquitetura.
2. **Gatilho de Supply Chain:**
   - Toda alteração em `go.mod` ou `go.sum` aciona compulsoriamente a skill `supply-chain-review`.
   - É obrigatória a execução do `govulncheck ./...` e `go mod verify`.

---

## 3. Segurança e Segredos

1. **Zero credenciais em código:**
   - Nenhum segredo, chave privada, token ou senha pode ser versionado no Git.
   - Todas as configurações sensíveis são injetadas exclusivamente via variáveis de ambiente (`.env`).
   - Arquivos `.env*`, `.pem`, `.key` e diretórios de credenciais são barrados por `.gitignore` e pelo validador de delivery.
2. **Salvaguarda de Segurança (D-14):**
   - Alterações que toquem a superfície de segurança (`security_surface_paths` em `.pipeline-ai.json`, incluindo `internal/auth/`, `internal/crypto/`, `go.mod`, `.github/workflows/`) exigem a flag `require_security: true` e a revisão formal pela skill `security-review`.

---

## 4. Governança e Git

1. **Sessão principal não edita código protegido:**
   - O hook nativo `PreToolUse` do Antigravity bloqueia tentativas de edição direta da sessão principal em caminhos protegidos (`protected_paths`).
   - Código de produção só é alterado por skills autorizadas (`dev-implementation`, `test-writer`, `docs-update`).
2. **Padrão de Branch e Commit:**
   - **Branch:** `task/{code}` ou `feat/{nome}`
   - **Commit:** `{type}({scope}): {summary} [{task_code}]`
3. **Pushes diretos na `main` são bloqueados:**
   - O repositório utiliza `.githooks/pre-push` local. Pushes diretos para `main` falham fatalmente.
   - Toda entrega entra via Pull Request (`gh pr create`).
4. **Validação de Entrega:**
   - Toda tarefa deve gerar um relatório em `docs-ai/deliveries/{DELIVERY_ID}/report.md` registrado no `INDEX.md`.
   - O comando `go tool pipeline-validate` deve retornar código `0` antes da abertura do PR.

