# Invariantes do Pipeline

Diretrizes e restrições obrigatórias para qualquer desenvolvimento neste projeto:

1. **Go 1.24+**: A diretiva `tool` no `go.mod` não é aceita por versões anteriores.
2. **Testes com stdlib `testing`**: Table-driven tests por padrão. Testify só se o projeto optar explicitamente.
3. **Sem CGO em desktop**: Essencial para cross-compilação WSL2 -> Windows funcionar sem containers Docker.
4. **Dependências externas exigem justificativa**: A stdlib cobre a maior parte das necessidades. Toda alteração em `go.mod` ou `go.sum` exige passagem por `supply-chain-review`.
5. **Nenhum segredo em código**: Chaves, tokens e credenciais devem vir exclusivamente de variáveis de ambiente.
6. **Proteção de código ativo**: Edições diretas em caminhos protegidos fora de agentes autorizados são barradas em tempo de execução via hook `PreToolUse`.
7. **Formatação automática determinística**: Todo arquivo editado passa pelo `gofmt -w` no hook `PostToolUse`.
