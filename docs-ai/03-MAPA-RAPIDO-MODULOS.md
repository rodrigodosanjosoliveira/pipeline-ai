# Mapa Rápido dos Módulos da Aplicação

> Este documento serve de guia para o `pm-triage` e os agentes técnicos identificarem módulos impactados pelo caminho de arquivos, sem necessidade de ler a base de código inteira.

---

## 1. Formato Desktop (Wails v3 + Go + React)

A arquitetura de um projeto desktop concentra backend Go e UI React empacotados num único executável binário.

```text
├── main.go               # Ponto de entrada Go e inicialização da runtime Wails v3
├── Taskfile.yml          # Tarefas determinísticas de build, dev e gates
├── .pipeline-ai.json     # Configuração de governança (caminhos protegidos e módulos)
├── .agents/              # Skills, regras e hooks do Antigravity
├── build/                # Superfície de segurança: manifests Windows, ícones e empacotamento
├── frontend/             # Interface do usuário (UI)
│   └── src/              # Componentes React, hooks, estado (Zustand), Tailwind
├── internal/             # Código de backend protegido
│   ├── app/              # Orquestração da aplicação e bindings expostos ao Wails/JS
│   ├── domain/           # Regras de negócio puras, entidades e interfaces (sem I/O)
│   ├── storage/          # Superfície de segurança: persistência e drivers de banco
│   └── config/           # Parsing e validação de variáveis de ambiente
└── docs-ai/              # Relatórios de entregas e documentação de arquitetura
```

### Guia de Roteamento por Caminho (Desktop):
- Alterações em `internal/domain/`: lógica de negócio → `architecture-guardrails` → `test-writer` → `dev-implementation`.
- Alterações em `internal/app/`: novos métodos expostos ao frontend → requer sincronização dos bindings do Wails.
- Alterações em `internal/storage/` ou `internal/auth/`: superfície de segurança → ativa `SECURITY: sim`.
- Alterações em `frontend/src/`: apenas UI → validação de build frontend (`npm run build`).
- Alterações em `build/`: empacotamento ou privilégios de sistema Windows → ativa `release-ops` e `security-review`.

---

## 2. Formato API (Go API + UI React Separada)

Projetos no formato API estruturam um servidor backend independente com pontos de entrada em `cmd/`.

```text
├── cmd/
│   └── api/
│       └── main.go       # Ponto de entrada do servidor HTTP / daemon
├── Taskfile.yml          # Automação de tarefas e gates
├── .pipeline-ai.json     # Configuração de governança
├── .agents/              # Skills, regras e hooks do Antigravity
├── migrations/           # Scripts DDL de migração de banco (superfície de segurança)
├── frontend/             # Aplicação SPA / Web consumidor
│   └── src/              # React, TypeScript, Tailwind
├── internal/             # Código de backend protegido
│   ├── http/             # Rotas, controllers HTTP, serialização e middlewares
│   ├── domain/           # Entidades, casos de uso e regras de negócio
│   ├── storage/          # Persistência e repositórios de dados
│   └── config/           # Configuração de ambiente e flags
└── docs-ai/              # Relatórios de entregas e documentação de arquitetura
```

### Guia de Roteamento por Caminho (API):
- Alterações em `internal/http/middleware/` ou `internal/auth/`: ativa compulsoriamente `SECURITY: sim`.
- Alterações em `migrations/`: ativa obrigatoriamente `MIGRATION: sim` e a skill `migration-reviewer`.
- Alterações em `internal/storage/`: avaliar se índices ou queries exigem `PERFORMANCE: sim`.
- Alterações em `cmd/`: ciclo de vida do servidor, sinais de SO e graceful shutdown.

---

## 3. Diretórios Globais do Padrão (Ambos os Formatos)

| Diretório / Arquivo | Função | Nível de Acesso |
|---|---|---|
| `.agents/` | Configurações, hooks e skills de automação do Antigravity | Protegido |
| `.github/workflows/` | Automação de CI/CD e gates remotos | Superfície de Segurança |
| `go.mod` / `go.sum` | Grafo de dependências e ferramentas do Go | Superfície de Segurança |
| `docs-ai/deliveries/` | Relatórios de entregas (`report.md`) e índice (`INDEX.md`) | Aberto para escrita de relatórios |
| `docs-ai/` | Decisões e invariantes de arquitetura | Aberto para leitura |
