package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// FlagSchema descreve uma flag de linha de comando para introspecção por agentes.
type FlagSchema struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Default     any      `json:"default"`
	Required    bool     `json:"required"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// EnvVarSchema descreve uma variável de ambiente reconhecida pela CLI.
type EnvVarSchema struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Default     string `json:"default"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// CommandSchema descreve um subcomando da CLI, suas flags e seu contrato de saída.
type CommandSchema struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Usage        string         `json:"usage"`
	SupportsJSON bool           `json:"supports_json"`
	Flags        []FlagSchema   `json:"flags"`
	OutputSchema map[string]any `json:"output_schema"`
}

// CLISchema descreve o catálogo completo da CLI para descoberta determinística por agentes.
type CLISchema struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	GlobalFlags []FlagSchema    `json:"global_flags"`
	EnvVars     []EnvVarSchema  `json:"env_vars"`
	Commands    []CommandSchema `json:"commands"`
}

func buildCLISchema() CLISchema {
	return CLISchema{
		Name:        "pipeline-ai",
		Version:     version,
		Description: "Toolkit de governança determinística, orquestração autônoma, runner e telemetria DORA para múltiplos agentes de IA.",
		GlobalFlags: []FlagSchema{
			{
				Name:        "-agent",
				Type:        "bool",
				Default:     false,
				Required:    false,
				Description: "Ativa modo agente (saída estruturada em JSON, desativa prompts interativos)",
			},
			{
				Name:        "-no-agent",
				Type:        "bool",
				Default:     false,
				Required:    false,
				Description: "Força modo humano interativo (ignora variáveis de ambiente)",
			},
			{
				Name:        "-schema",
				Type:        "bool",
				Default:     false,
				Required:    false,
				Description: "Emite o schema JSON da CLI ou do comando especificado sem executar efeitos colaterais",
			},
		},
		EnvVars: []EnvVarSchema{
			{
				Name:        "PIPELINE_AGENT_MODE",
				Type:        "bool",
				Default:     "false",
				Required:    false,
				Description: "Ativa ou desativa o modo agente via ambiente (1/true/0/false)",
			},
			{
				Name:        "AI_AGENT",
				Type:        "bool",
				Default:     "false",
				Required:    false,
				Description: "Variável de ambiente alternativa para detectar execução por agente de IA",
			},
			{
				Name:        "GEMINI_API_KEY",
				Type:        "string",
				Default:     "",
				Required:    false,
				Description: "Chave de API para avaliação semântica com provider Gemini",
			},
			{
				Name:        "PIPELINE_SEMANTIC_PROVIDER",
				Type:        "string",
				Default:     "",
				Required:    false,
				Description: "Provider padrão para avaliação semântica ('jev' ou 'gemini')",
			},
			{
				Name:        "PIPELINE_SEMANTIC_ENDPOINT",
				Type:        "string",
				Default:     "",
				Required:    false,
				Description: "Endpoint HTTP padrão para o sensor semântico",
			},
			{
				Name:        "PIPELINE_SEMANTIC_API_KEY",
				Type:        "string",
				Default:     "",
				Required:    false,
				Description: "Chave de API padrão para o sensor semântico",
			},
			{
				Name:        "JEV_ENDPOINT",
				Type:        "string",
				Default:     "",
				Required:    false,
				Description: "Endpoint HTTP legado para o servidor semântico Jev",
			},
			{
				Name:        "JEV_API_KEY",
				Type:        "string",
				Default:     "",
				Required:    false,
				Description: "Chave de API legada para o servidor semântico Jev",
			},
		},
		Commands: []CommandSchema{
			{
				Name:         "version",
				Description:  "Exibe a versão instalada do pipeline-ai",
				Usage:        "pipeline-ai version [-json]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "json", Type: "bool", Default: false, Description: "Emite a versão em formato JSON"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name":    map[string]any{"type": "string"},
						"version": map[string]any{"type": "string"},
					},
				},
			},
			{
				Name:         "init",
				Description:  "Inicializa um novo projeto no padrão pipeline-ai",
				Usage:        "pipeline-ai init [flags]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "name", Type: "string", Default: "", Required: true, Description: "Nome do projeto (obrigatório)"},
					{Name: "type", Type: "string", Default: "", Description: "Formato do projeto", Enum: []string{"desktop", "api", "cli"}},
					{Name: "db", Type: "string", Default: "postgres", Description: "Banco de dados", Enum: []string{"postgres", "mongo", "dynamo"}},
					{Name: "module", Type: "string", Default: "", Description: "Caminho do módulo Go (ex: github.com/user/repo)"},
					{Name: "dir", Type: "string", Default: ".", Description: "Diretório de destino"},
					{Name: "summary", Type: "string", Default: "", Description: "Uma frase descritiva sobre o projeto"},
					{Name: "no-git", Type: "bool", Default: false, Description: "Não inicializa repositório Git"},
					{Name: "non-interactive", Type: "bool", Default: false, Description: "Desativa prompts interativos"},
					{Name: "json", Type: "bool", Default: false, Description: "Exibe o resultado em formato JSON"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"dir":           map[string]any{"type": "string"},
						"created_files": map[string]any{"type": "array"},
					},
				},
			},
			{
				Name:         "doctor",
				Description:  "Diagnostica o ambiente local e configurações do projeto",
				Usage:        "pipeline-ai doctor [-dir <caminho>] [-json]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "dir", Type: "string", Default: ".", Description: "Diretório raiz do projeto a ser verificado"},
					{Name: "json", Type: "bool", Default: false, Description: "Emite o diagnóstico estruturado em formato JSON"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"ok":     map[string]any{"type": "boolean"},
						"checks": map[string]any{"type": "array"},
					},
				},
			},
			{
				Name:         "validate",
				Description:  "Valida o relatório de delivery e salvaguardas de segurança antes do PR",
				Usage:        "pipeline-ai validate [-base <ref>] [-no-git] [-json] <caminho-do-report.md>",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "base", Type: "string", Default: "", Description: "Referência git para o diff"},
					{Name: "no-git", Type: "bool", Default: false, Description: "Pula as checagens que dependem do git"},
					{Name: "json", Type: "bool", Default: false, Description: "Emite o resultado estruturado em JSON"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"ok":     map[string]any{"type": "boolean"},
						"errors": map[string]any{"type": "array"},
					},
				},
			},
			{
				Name:         "handoff",
				Description:  "Valida integridade e contratos de blocos de handoff",
				Usage:        "pipeline-ai handoff validate [-json] <arquivo-ou-string>",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "json", Type: "bool", Default: false, Description: "Emite resultado estruturado em JSON"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"ok":     map[string]any{"type": "boolean"},
						"errors": map[string]any{"type": "array"},
					},
				},
			},
			{
				Name:         "run",
				Description:  "Executa a suíte integrada de gates de governança (doctor, sensor, validate)",
				Usage:        "pipeline-ai run [flags]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "dir", Type: "string", Default: ".", Description: "Diretório raiz do projeto"},
					{Name: "delivery", Type: "string", Default: "", Description: "ID da delivery a validar (ex: DEL-001)"},
					{Name: "report", Type: "string", Default: "", Description: "Caminho do relatório de delivery"},
					{Name: "no-git", Type: "bool", Default: false, Description: "Pula verificações dependentes do git"},
					{Name: "base", Type: "string", Default: "", Description: "Referência git para o diff"},
					{Name: "max-output-bytes", Type: "int", Default: 65536, Description: "Limite máximo de bytes na saída dos sensores"},
					{Name: "vuln", Type: "bool", Default: false, Description: "Inclui auditoria govulncheck no estágio de sensores"},
					{Name: "semantic-gate", Type: "bool", Default: false, Description: "Executa gate semântico opcional (YAGNI) após validação"},
					{Name: "json", Type: "bool", Default: false, Description: "Emite os resultados consolidados em JSON"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"ok":          map[string]any{"type": "boolean"},
						"delivery_id": map[string]any{"type": "string"},
						"overall":     map[string]any{"type": "string"},
						"steps":       map[string]any{"type": "array"},
					},
				},
			},
			{
				Name:         "sync",
				Description:  "Sincroniza skills, regras e hooks em .agents/ a partir dos templates",
				Usage:        "pipeline-ai sync [-dir <caminho>] [-json]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "dir", Type: "string", Default: ".", Description: "Diretório raiz do projeto a ser sincronizado"},
					{Name: "json", Type: "bool", Default: false, Description: "Exibe o resultado em formato JSON"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"ok":            map[string]any{"type": "boolean"},
						"updated_files": map[string]any{"type": "array"},
					},
				},
			},
			{
				Name:         "delivery",
				Description:  "Gerencia o ciclo de vida da delivery ativa (open, close, status)",
				Usage:        "pipeline-ai delivery <open|close|status> [flags]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "dir", Type: "string", Default: ".", Description: "Diretório raiz do projeto"},
					{Name: "task", Type: "string", Default: "", Description: "Código da tarefa (subcomando open)"},
					{Name: "ttl", Type: "string", Default: "168h", Description: "Tempo de validade TTL (subcomando open)"},
					{Name: "json", Type: "bool", Default: false, Description: "Exibe o status em formato JSON (subcomando status)"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"delivery_id": map[string]any{"type": "string"},
						"task":        map[string]any{"type": "string"},
						"status":      map[string]any{"type": "string"},
					},
				},
			},
			{
				Name:         "module",
				Description:  "Gerencia módulos opcionais do pipeline (list, enable, disable)",
				Usage:        "pipeline-ai module <list|enable|disable> [nome] [-dir <caminho>] [-json]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "dir", Type: "string", Default: ".", Description: "Diretório raiz do projeto"},
					{Name: "json", Type: "bool", Default: false, Description: "Emite a lista de módulos em formato JSON"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"modules": map[string]any{"type": "array"},
					},
				},
			},
			{
				Name:         "sensor",
				Description:  "Executa os sensores de qualidade (fmt, vet, test) com saída estruturada",
				Usage:        "pipeline-ai sensor [flags]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "dir", Type: "string", Default: ".", Description: "Diretório raiz do projeto a ser analisado"},
					{Name: "max-output-bytes", Type: "int", Default: 65536, Description: "Limite máximo de bytes na saída de cada sensor"},
					{Name: "verbose-output", Type: "bool", Default: false, Description: "Preserva logs verbosos de testes que passaram até o limite de bytes"},
					{Name: "vuln", Type: "bool", Default: false, Description: "Executa auditoria de vulnerabilidades (govulncheck ./...)"},
					{Name: "json", Type: "bool", Default: false, Description: "Emite os resultados estruturados em JSON"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"ok":      map[string]any{"type": "boolean"},
						"sensors": map[string]any{"type": "array"},
					},
				},
			},
			{
				Name:         "metrics",
				Description:  "Exibe métricas de engenharia e governança DORA (summary, list)",
				Usage:        "pipeline-ai metrics <summary|list> [-dir <caminho>] [-window <dias>] [-json]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "dir", Type: "string", Default: ".", Description: "Diretório raiz do projeto"},
					{Name: "window", Type: "float64", Default: 0.0, Description: "Janela de cálculo em dias (opcional)"},
					{Name: "json", Type: "bool", Default: false, Description: "Emite os resultados estruturados em JSON"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"summary": map[string]any{"type": "object"},
						"events":  map[string]any{"type": "array"},
					},
				},
			},
			{
				Name:         "dashboard",
				Description:  "Gera e renderiza dashboard HTML interativo de engenharia e métricas",
				Usage:        "pipeline-ai dashboard [-dir <caminho>] [-output <arquivo.html>] [-json]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "dir", Type: "string", Default: ".", Description: "Diretório raiz do projeto"},
					{Name: "output", Type: "string", Default: "", Description: "Caminho para gravar o arquivo HTML"},
					{Name: "json", Type: "bool", Default: false, Description: "Emite os dados do dashboard em JSON"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"generated_at": map[string]any{"type": "string"},
						"metrics":      map[string]any{"type": "object"},
					},
				},
			},
			{
				Name:         "mcp",
				Description:  "Executa o servidor MCP sobre stdio (JSON-RPC 2.0)",
				Usage:        "pipeline-ai mcp [-dir <caminho>]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "dir", Type: "string", Default: ".", Description: "Diretório raiz do projeto"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"jsonrpc": map[string]any{"type": "string"},
						"result":  map[string]any{"type": "object"},
					},
				},
			},
			{
				Name:         "plugin",
				Description:  "Gerencia o bundle e instalação do plugin Antigravity (verify, install)",
				Usage:        "pipeline-ai plugin <verify|install> [flags]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "dir", Type: "string", Default: "plugin", Description: "Diretório do bundle do plugin"},
					{Name: "src", Type: "string", Default: "plugin", Description: "Diretório de origem do bundle (subcomando install)"},
					{Name: "target", Type: "string", Default: "", Description: "Diretório de destino da instalação (subcomando install)"},
					{Name: "json", Type: "bool", Default: false, Description: "Exibe o resultado em formato JSON"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"ok":    map[string]any{"type": "boolean"},
						"files": map[string]any{"type": "array"},
					},
				},
			},
			{
				Name:         "eval",
				Description:  "Submete uma avaliação semântica calibrada (System One / Jev / Gemini)",
				Usage:        "pipeline-ai eval [flags]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "state", Type: "string", Default: "", Description: "Caminho de arquivo ou string de estado para avaliação"},
					{Name: "question", Type: "string", Default: "", Description: "Pergunta ou critério semântico a ser avaliado"},
					{Name: "preset", Type: "string", Default: "", Description: "Preset semântico calibrado (ex: yagni, minimalism, ponytail)"},
					{Name: "threshold", Type: "float64", Default: 0.8, Description: "Score mínimo de confiança requerido (0.0 a 1.0)"},
					{Name: "mode", Type: "string", Default: "noul", Description: "Modo de avaliação", Enum: []string{"noul", "choice", "score"}},
					{Name: "choices", Type: "string", Default: "", Description: "Opções separadas por vírgula para -mode choice"},
					{Name: "max-score", Type: "float64", Default: 10.0, Description: "Pontuação máxima para -mode score"},
					{Name: "dir", Type: "string", Default: ".", Description: "Diretório raiz para registro de telemetria"},
					{Name: "json", Type: "bool", Default: false, Description: "Saída em formato JSON estruturado"},
					{Name: "endpoint", Type: "string", Default: "", Description: "Endpoint HTTP do sensor semântico (Jev)"},
					{Name: "api-key", Type: "string", Default: "", Description: "Chave de API do sensor semântico"},
					{Name: "timeout", Type: "duration", Default: "10s", Description: "Tempo limite para a requisição de avaliação"},
					{Name: "provider", Type: "string", Default: "", Description: "Provider semântico", Enum: []string{"jev", "gemini"}},
					{Name: "model", Type: "string", Default: "", Description: "Modelo para avaliação semântica"},
					{Name: "thinking-level", Type: "string", Default: "", Description: "Nível de thinking para gemini", Enum: []string{"low", "medium", "high"}},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"ok":         map[string]any{"type": "boolean"},
						"answer":     map[string]any{"type": "string"},
						"confidence": map[string]any{"type": "number"},
						"provider":   map[string]any{"type": "string"},
					},
				},
			},
			{
				Name:         "debt",
				Description:  "Lista atalhos deliberados (// shortcut:) com limites e gatilhos de evolução",
				Usage:        "pipeline-ai debt [-dir <caminho>] [-json]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "dir", Type: "string", Default: ".", Description: "Diretório raiz do projeto a ser analisado"},
					{Name: "json", Type: "bool", Default: false, Description: "Emite o inventário de atalhos em formato JSON"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"count":     map[string]any{"type": "integer"},
						"shortcuts": map[string]any{"type": "array"},
					},
				},
			},
			{
				Name:         "schema",
				Description:  "Emite o schema JSON de introspecção da CLI ou de um comando específico",
				Usage:        "pipeline-ai schema [comando]",
				SupportsJSON: true,
				Flags: []FlagSchema{
					{Name: "json", Type: "bool", Default: false, Description: "Emite o schema em formato JSON (comportamento padrão)"},
				},
				OutputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name":         map[string]any{"type": "string"},
						"version":      map[string]any{"type": "string"},
						"global_flags": map[string]any{"type": "array"},
						"env_vars":     map[string]any{"type": "array"},
						"commands":     map[string]any{"type": "array"},
					},
				},
			},
		},
	}
}

func runSchema(args []string, stdout, stderr io.Writer) int {
	var target string
	for _, a := range args {
		if a == "-json" || a == "--json" {
			continue
		}
		if target == "" {
			target = strings.ToLower(strings.TrimSpace(a))
		}
	}

	schema := buildCLISchema()

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")

	if target == "" {
		if err := enc.Encode(schema); err != nil {
			fmt.Fprintf(stderr, "pipeline-ai schema: erro ao serializar JSON: %v\n", err)
			return 1
		}
		return 0
	}

	for _, cmd := range schema.Commands {
		if cmd.Name == target {
			if err := enc.Encode(cmd); err != nil {
				fmt.Fprintf(stderr, "pipeline-ai schema: erro ao serializar JSON: %v\n", err)
				return 1
			}
			return 0
		}
	}

	errMsg := fmt.Sprintf("comando desconhecido para schema: %q", target)
	hint := "Execute 'pipeline-ai schema' para listar todos os comandos disponíveis."
	_ = enc.Encode(map[string]any{
		"ok":    false,
		"error": errMsg,
		"hint":  hint,
	})
	fmt.Fprintf(stderr, "pipeline-ai schema: %s\n%s\n", errMsg, hint)
	return 1
}
