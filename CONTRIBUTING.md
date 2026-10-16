# Guia de Contribuição — pipeline-ai

Obrigado pelo interesse em contribuir com o **pipeline-ai**! 🎉

Este repositório fornece um toolkit e padrão de engenharia para desenvolvimento assistido por agentes de IA com governança determinística. Toda contribuição que fortaleça a confiabilidade, simplifique o uso ou expanda os recursos dentro dos invariantes do projeto é muito bem-vinda.

---

## Princípios e Invariantes do Projeto

Antes de propor alterações no código ou nas ferramentas, é fundamental observar os invariantes que garantem a integridade da arquitetura:

1. **Go 1.24+**: Usamos a diretiva nativa `tool` no `go.mod`.
2. **Zero Dependências Externas**: Os binários do toolkit (`pipeline-hook`, `pipeline-validate`, `pipeline-init`) utilizam estritamente a biblioteca padrão do Go (`stdlib`). Nenhuma dependência externa deve ser adicionada sem discussão prévia e justificativa incontornável.
3. **TDD com a Biblioteca Padrão (`testing`)**: Todo código novo deve ser acompanhado de testes de unidade table-driven.
4. **Sem CGO**: O build deve ser 100% puro (`CGO_ENABLED=0`) para preservar a portabilidade e cross-compilação entre Linux/WSL2 e Windows.
5. **Formatação Determinística**: Todo código deve ser formatado com `gofmt`.

---

## Fluxo de Trabalho (Branch → PR)

Diretrizes para o ciclo de desenvolvimento:

1. **Abra ou encontre uma Issue**: Para alterações não triviais, abra uma issue descrevendo o problema ou proposta antes de iniciar o código.
2. **Crie uma Branch**:
   - `feat/{nome-da-feature}` para novas funcionalidades.
   - `fix/{descricao-do-bug}` para correções.
   - `docs/{tema}` para melhorias de documentação.
   - Pushes diretos na branch `main` são bloqueados pelo hook local (`.githooks/pre-push`) e pela proteção de branch do repositório.
3. **Commits Convencionais**: Siga a convenção de commits:
   - `feat(escopo): descrição concisa`
   - `fix(escopo): descrição concisa`
   - `docs(escopo): descrição concisa`
   - `test(escopo): descrição concisa`
   - `refactor(escopo): descrição concisa`
4. **Validação Local**: Antes de abrir o Pull Request, execute os gates obrigatórios:
   ```bash
   task check
   # ou manualmente:
   gofmt -l .
   go vet ./...
   go test -race -v ./...
   ```
5. **Abra o Pull Request**: Preencha o template de PR detalhando a motivação, o escopo da alteração e as evidências de teste.

---

## Comandos Úteis com Task

Se você tiver a ferramenta [Task](https://taskfile.dev/) instalada, pode usar os atalhos:

| Comando | Descrição |
|---|---|
| `task check` | Executa formatação, vet e testes com detector de race |
| `task test` | Executa todos os testes unitários |
| `task test:race` | Executa testes com detecção de concorrência (`-race`) |
| `task fmt` | Formata o código com `gofmt` |
| `task vet` | Executa o `go vet` em todos os pacotes |
| `task build` | Compila todos os binários em `cmd/` |
| `task vuln` | Executa o `govulncheck` para auditoria de segurança |

---

## Código de Conduta

Esperamos que todas as interações no repositório (issues, PRs, discussões) sejam pautadas pelo respeito mútuo, cordialidade, escuta ativa e foco em soluções construtivas.

