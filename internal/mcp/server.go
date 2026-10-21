package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor"
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
	// Aumenta buffer para suportar payloads grandes se necessário
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
	// Notificações não possuem ID e não devem gerar resposta JSON
	isNotification := req.ID == nil

	switch req.Method {
	case "initialize":
		res := map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "pipeline-ai",
				"version": "v0.4.0",
			},
		}
		s.sendResult(out, req.ID, res)

	case "notifications/initialized":
		// Handshake concluído pelo cliente MCP (nenhuma resposta necessária)
		return

	case "ping":
		s.sendResult(out, req.ID, map[string]any{})

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
