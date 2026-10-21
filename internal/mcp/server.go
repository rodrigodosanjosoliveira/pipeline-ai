package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate"
)

// Server representa o servidor MCP do pipeline-ai sobre transporte stdio JSON-RPC 2.0.
type Server struct {
	RootDir string
}

// NewServer inicializa um novo servidor MCP.
func NewServer(rootDir string) *Server {
	if rootDir == "" {
		rootDir = "."
	}
	return &Server{RootDir: rootDir}
}

// ServeStream processa requisições JSON-RPC linha a linha até encontrar EOF.
func (s *Server) ServeStream(in io.Reader, out, errOut io.Writer) error {
	scanner := bufio.NewScanner(in)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(out, nil, CodeParseError, "erro ao decodificar JSON")
			continue
		}

		s.handleRequest(&req, out, errOut)
	}

	return scanner.Err()
}

func (s *Server) handleRequest(req *Request, out, errOut io.Writer) {
	isNotification := req.ID == nil

	switch req.Method {
	case "initialize":
		res := map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "pipeline-ai",
				"version": "v0.5.0",
			},
		}
		s.sendResult(out, req.ID, res)

	case "notifications/initialized":
		return

	case "ping":
		s.sendResult(out, req.ID, map[string]any{})

	case "resources/list":
		s.handleResourcesList(req, out)

	case "resources/read":
		s.handleResourcesRead(req, out)

	case "tools/list":
		s.handleToolsList(req, out)

	case "tools/call":
		s.handleToolsCall(req, out, errOut)

	default:
		if !isNotification {
			s.sendError(out, req.ID, CodeMethodNotFound, fmt.Sprintf("método não encontrado: %s", req.Method))
		}
	}
}

func (s *Server) handleResourcesList(req *Request, out io.Writer) {
	resources := []Resource{
		{
			URI:         "pipeline://invariants",
			Name:        "Invariantes Globais do Projeto",
			Description: "Regras inegociáveis de arquitetura e governança do projeto",
			MimeType:    "text/markdown",
		},
		{
			URI:         "pipeline://active-delivery",
			Name:        "Delivery Ativa",
			Description: "Status da delivery mais recente registrada no INDEX.md",
			MimeType:    "text/markdown",
		},
		{
			URI:         "pipeline://catalog",
			Name:        "Catálogo de Skills",
			Description: "Lista de skills nativas em .agents/skills",
			MimeType:    "text/markdown",
		},
	}
	s.sendResult(out, req.ID, map[string]any{"resources": resources})
}

func (s *Server) handleResourcesRead(req *Request, out io.Writer) {
	var params struct {
		URI string `json:"uri"`
	}
	_ = json.Unmarshal(req.Params, &params)

	switch params.URI {
	case "pipeline://invariants":
		var text string
		candidates := []string{
			filepath.Join(s.RootDir, "docs-ai", "01-INVARIANTES-GLOBAIS.md"),
			filepath.Join(s.RootDir, ".agents", "rules", "invariants.md"),
			filepath.Join(s.RootDir, "AGENTS.md"),
		}
		for _, p := range candidates {
			if data, err := os.ReadFile(p); err == nil {
				text = string(data)
				break
			}
		}
		s.sendResult(out, req.ID, ReadResourceResult{
			Contents: []ResourceContent{
				{
					URI:      params.URI,
					MimeType: "text/markdown",
					Text:     text,
				},
			},
		})

	case "pipeline://active-delivery":
		var text strings.Builder
		indexFile := filepath.Join(s.RootDir, "docs-ai", "deliveries", "INDEX.md")
		if data, err := os.ReadFile(indexFile); err == nil {
			text.WriteString("# Índice de Deliveries\n\n")
			text.WriteString(string(data))
		}
		s.sendResult(out, req.ID, ReadResourceResult{
			Contents: []ResourceContent{
				{
					URI:      params.URI,
					MimeType: "text/markdown",
					Text:     text.String(),
				},
			},
		})

	case "pipeline://catalog":
		var text strings.Builder
		text.WriteString("# Catálogo de Skills Nativas (.agents/skills)\n\n")
		skillsDir := filepath.Join(s.RootDir, ".agents", "skills")
		if entries, err := os.ReadDir(skillsDir); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					text.WriteString(fmt.Sprintf("- **%s**\n", e.Name()))
				}
			}
		}
		s.sendResult(out, req.ID, ReadResourceResult{
			Contents: []ResourceContent{
				{
					URI:      params.URI,
					MimeType: "text/markdown",
					Text:     text.String(),
				},
			},
		})

	default:
		s.sendError(out, req.ID, CodeInvalidParams, fmt.Sprintf("recurso não encontrado: %s", params.URI))
	}
}

func (s *Server) handleToolsList(req *Request, out io.Writer) {
	tools := []Tool{
		{
			Name:        "pipeline_doctor",
			Description: "Executa o diagnóstico completo do ambiente local e integridade do projeto (.pipeline-ai.json, Go 1.24+, git, gh, hooks, skills).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"dir": map[string]any{
						"type":        "string",
						"description": "Diretório raiz do projeto a ser verificado (padrão: raiz atual)",
					},
				},
			},
		},
		{
			Name:        "pipeline_validate",
			Description: "Valida o relatório de entrega (report.md) e as salvaguardas de git e segurança antes do commit ou PR.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"report_path": map[string]any{
						"type":        "string",
						"description": "Caminho do arquivo report.md da delivery a ser validado",
					},
					"no_git": map[string]any{
						"type":        "boolean",
						"description": "Se verdadeiro, desativa a validação do git diff contra a base_ref",
					},
					"base_ref": map[string]any{
						"type":        "string",
						"description": "Referência base do git para comparação (ex: origin/main)",
					},
				},
				"required": []string{"report_path"},
			},
		},
		{
			Name:        "pipeline_scaffold_delivery",
			Description: "Inicializa o esqueleto de uma nova entrega: cria docs-ai/deliveries/DEL-NNN/report.md com seções obrigatórias e registra no INDEX.md.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{
						"type":        "string",
						"description": "Título descritivo da nova delivery",
					},
					"type": map[string]any{
						"type":        "string",
						"description": "Tipo da entrega (feat, bugfix, refactor, chore, docs)",
						"enum":        []string{"feat", "bugfix", "refactor", "chore", "docs"},
					},
					"task_code": map[string]any{
						"type":        "string",
						"description": "Código da tarefa (opcional, padrão: TASK-NNN)",
					},
				},
				"required": []string{"title", "type"},
			},
		},
		{
			Name:        "pipeline_sync_skills",
			Description: "Sincroniza skills, regras e hooks em .agents/ a partir dos templates embutidos da CLI.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"dir": map[string]any{
						"type":        "string",
						"description": "Diretório raiz do projeto a ser sincronizado (padrão: raiz atual)",
					},
				},
			},
		},
	}

	s.sendResult(out, req.ID, map[string]any{
		"tools": tools,
	})
}

func (s *Server) handleToolsCall(req *Request, out, errOut io.Writer) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(out, req.ID, CodeInvalidParams, "parâmetros inválidos para tools/call")
		return
	}

	switch params.Name {
	case "pipeline_doctor":
		targetDir := s.RootDir
		if rawDir, ok := params.Arguments["dir"].(string); ok && strings.TrimSpace(rawDir) != "" {
			if filepath.IsAbs(rawDir) {
				targetDir = rawDir
			} else {
				targetDir = filepath.Join(s.RootDir, rawDir)
			}
		}

		var buf bytes.Buffer
		code := doctor.Run(targetDir, &buf, errOut)
		isError := code != 0

		s.sendResult(out, req.ID, ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: buf.String(),
				},
			},
			IsError: isError,
		})

	case "pipeline_validate":
		reportPath, _ := params.Arguments["report_path"].(string)
		if strings.TrimSpace(reportPath) == "" {
			s.sendResult(out, req.ID, ToolResult{
				Content: []ToolContent{
					{
						Type: "text",
						Text: "ERRO: parâmetro 'report_path' é obrigatório",
					},
				},
				IsError: true,
			})
			return
		}

		if !filepath.IsAbs(reportPath) {
			reportPath = filepath.Join(s.RootDir, reportPath)
		}

		noGit, _ := params.Arguments["no_git"].(bool)
		baseRef, _ := params.Arguments["base_ref"].(string)

		var outBuf, errBuf bytes.Buffer
		code := validate.Run(reportPath, baseRef, noGit, &outBuf, &errBuf)

		output := outBuf.String()
		if errBuf.Len() > 0 {
			if output != "" {
				output += "\n"
			}
			output += errBuf.String()
		}

		s.sendResult(out, req.ID, ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: output,
				},
			},
			IsError: code != 0,
		})

	case "pipeline_scaffold_delivery":
		title, _ := params.Arguments["title"].(string)
		delivType, _ := params.Arguments["type"].(string)
		taskCode, _ := params.Arguments["task_code"].(string)

		if strings.TrimSpace(title) == "" || strings.TrimSpace(delivType) == "" {
			s.sendResult(out, req.ID, ToolResult{
				Content: []ToolContent{
					{
						Type: "text",
						Text: "ERRO: parâmetros 'title' e 'type' são obrigatórios",
					},
				},
				IsError: true,
			})
			return
		}

		delivID, reportPath, err := ScaffoldDelivery(s.RootDir, title, delivType, taskCode)
		if err != nil {
			s.sendResult(out, req.ID, ToolResult{
				Content: []ToolContent{
					{
						Type: "text",
						Text: fmt.Sprintf("ERRO ao criar scaffold da delivery: %v", err),
					},
				},
				IsError: true,
			})
			return
		}

		s.sendResult(out, req.ID, ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Delivery %s criada com sucesso!\nRelatório: %s\nRegistrada no INDEX.md com status 'draft'.", delivID, reportPath),
				},
			},
			IsError: false,
		})

	case "pipeline_sync_skills":
		targetDir := s.RootDir
		if rawDir, ok := params.Arguments["dir"].(string); ok && strings.TrimSpace(rawDir) != "" {
			if filepath.IsAbs(rawDir) {
				targetDir = rawDir
			} else {
				targetDir = filepath.Join(s.RootDir, rawDir)
			}
		}
		var outBuf, errBuf bytes.Buffer
		err := scaffold.Sync(targetDir, &outBuf, &errBuf)
		output := outBuf.String()
		if err != nil {
			if output != "" {
				output += "\n"
			}
			output += fmt.Sprintf("ERRO: %v", err)
		}
		s.sendResult(out, req.ID, ToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: output,
				},
			},
			IsError: err != nil,
		})

	default:
		s.sendError(out, req.ID, CodeInvalidParams, fmt.Sprintf("ferramenta desconhecida: %s", params.Name))
	}
}

func (s *Server) sendResult(out io.Writer, id any, result any) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, _ := json.Marshal(resp)
	fmt.Fprintf(out, "%s\n", string(data))
}

func (s *Server) sendError(out io.Writer, id any, code int, msg string) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: msg,
		},
	}
	data, _ := json.Marshal(resp)
	fmt.Fprintf(out, "%s\n", string(data))
}
