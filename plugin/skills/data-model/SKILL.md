---
name: data-model
description: >-
  Projeta e evolui o modelo de dados e entidades antes da implementação, suportando PostgreSQL, MongoDB e DynamoDB.
  Use após architecture-guardrails quando houver persistência ou alteração de esquema.
---

# Data Model

## Missão

Definir o design das entidades de domínio e o esquema de persistência com base no banco de dados adotado pelo projeto (PostgreSQL, MongoDB ou DynamoDB), garantindo integridade, consistência e alinhamento com os padrões de acesso antes da escrita de testes e código.

## Entrada

Handoff do `architecture-guardrails` com os limites técnicos da demanda e decisões arquiteturais.

## Procedimento

1. **Identificar o banco de dados configurado no projeto:**
   - Consultar `AGENTS.md` e `internal/storage/` para identificar o motor de persistência:
     - `PostgreSQL` (Relacional / SQL)
     - `MongoDB` (Documento / BSON)
     - `DynamoDB` (NoSQL Key-Value / Single-Table)

2. **Modelagem de acordo com o paradigma:**
   - **PostgreSQL (Relacional):**
     - Mapear tabelas, colunas, tipos SQL idiomáticos (`uuid`, `timestamptz`, `text`, `numeric`).
     - Definir chaves primárias e estrangeiras com ações referenciais explícitas (`ON DELETE RESTRICT`).
     - Declarar constraints de integridade (`NOT NULL`, `UNIQUE`, `CHECK`).
   - **MongoDB (Documento):**
     - Estruturar schemas de documentos BSON e tags das structs Go (`bson:"campo"`).
     - Decidir explicitamente entre *embedding* (dados que são lidos juntos e possuem ciclo de vida acoplado) ou *referencing* (relacionamentos 1:N com alta cardinalidade).
     - Definir regras de validação de schema para coleções.
   - **DynamoDB (NoSQL / Chave-Valor):**
     - Mapear todos os padrões de acesso (*access patterns*) antes de definir as chaves.
     - Definir chave composta: Partition Key (`PK`) e Sort Key (`SK`) com prefixos semânticos (ex: `USER#123`, `ORDER#456`).
     - Projetar Global Secondary Indexes (GSIs) apenas para consultas que o par PK/SK não atenda.

3. **Mapeamento para o Domínio:**
   - Especificar as structs de domínio Go puras em `internal/domain/`, garantindo separação entre modelo de domínio e modelo de persistência de banco.

## O que NÃO fazer

- Não escrever código de drivers nem repositórios (isso é do `dev-implementation`).
- Não criar scripts de migração definitivos (a migração é codificada em `dev-implementation` e revisada por `migration-reviewer`).
- Não misturar paradigmas ou sugerir múltiplos bancos para o mesmo projeto (invariante D-09).
- Não violar o invariante sem CGO em projetos desktop (drivers SQLite com CGO são proibidos no desktop).

## Roteamento

Sempre para `test-writer`.

## Saída obrigatória

O bloco do contrato de handoff, contendo:
- `ENTIDADES_DOMINIO`: structs e campos de negócio especificados.
- `ESQUEMA_BANCO`: especificação de tabelas/coleções/chaves para o banco do projeto.
- `PADROES_DE_ACESSO`: lista de consultas/leituras previstas.

