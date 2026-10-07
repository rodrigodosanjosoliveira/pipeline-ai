// Command pipeline-hook é o hook PreToolUse que impede a sessão principal de
// editar código diretamente, forçando a delegação aos agents do pipeline.
//
// É a camada 2 da decisão D-11. No Antigravity, o hook é configurado em
// .agents/hooks.json (PreToolUse) e emite JSON com decision ("allow" ou "deny")
// e reason no stdout. No formato legado, emite erro no stderr com exit code 2.
//
// Entrada: JSON no stdin, no formato PreToolUse (Antigravity ou legado).
// Saída:
//
//	Antigravity: JSON no stdout ({"decision": "allow"|"deny", "reason": "..."}), exit 0
//	Legado:      exit 0 (permitido) ou exit 2 (bloqueado com motivo no stderr)
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config"
)

const (
	exitAllow = 0
	exitBlock = 2
)

// toolCall representa uma chamada de ferramenta no protocolo Antigravity.
type toolCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// hookInput suporta tanto o payload do Antigravity (toolCall, workspacePaths)
// quanto o formato legado do Claude Code (tool_name, tool_input.file_path, cwd).
type hookInput struct {
	// Protocolo Antigravity
	ToolCall       *toolCall `json:"toolCall"`
	WorkspacePaths []string  `json:"workspacePaths"`
	SubagentID     string    `json:"subagentId"`

	// Protocolo legado
	ToolName  string `json:"tool_name"`
	AgentType string `json:"agent_type"`
	CWD       string `json:"cwd"`
	ToolInput struct {
		FilePath string `json:"file_path"`
	} `json:"tool_input"`
}

// antigravityOutput é a resposta no stdout esperada pelo Antigravity PreToolUse.
type antigravityOutput struct {
	Decision string `json:"decision"` // "allow" ou "deny"
	Reason   string `json:"reason,omitempty"`
}

// editTools são as ferramentas que alteram arquivos (Antigravity e legado).
var editTools = map[string]bool{
	// Antigravity tools
	"write_to_file":              true,
	"replace_file_content":       true,
	"multi_replace_file_content": true,

	// Claude Code tools
	"Edit":         true,
	"Write":        true,
	"MultiEdit":    true,
	"NotebookEdit": true,
}

func (in *hookInput) isAntigravity() bool {
	return in.ToolCall != nil
}

func (in *hookInput) toolName() string {
	if in.ToolCall != nil {
		return in.ToolCall.Name
	}
	return in.ToolName
}

func (in *hookInput) filePath() string {
	if in.ToolCall != nil && in.ToolCall.Args != nil {
		for _, k := range []string{"TargetFile", "target_file", "file_path", "FilePath", "path", "Path"} {
			if val, ok := in.ToolCall.Args[k]; ok {
				if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
	}
	return strings.TrimSpace(in.ToolInput.FilePath)
}

func (in *hookInput) agentType() string {
	if in.AgentType != "" {
		return strings.TrimSpace(in.AgentType)
	}
	if in.SubagentID != "" {
		return strings.TrimSpace(in.SubagentID)
	}
	return ""
}

func (in *hookInput) startDir() string {
	if len(in.WorkspacePaths) > 0 && strings.TrimSpace(in.WorkspacePaths[0]) != "" {
		return in.WorkspacePaths[0]
	}
	if in.CWD != "" {
		return in.CWD
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

func main() {
	os.Exit(run(os.Stdin, os.Stdout, os.Stderr))
}

func run(stdin io.Reader, stdout, stderr io.Writer) int {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		// Sem entrada legível não há o que avaliar.
		fmt.Fprintf(stderr, "pipeline-hook: falha ao ler stdin: %v\n", err)
		return replyAllow(false, stdout)
	}

	var in hookInput
	if err := json.Unmarshal(raw, &in); err != nil {
		fmt.Fprintf(stderr, "pipeline-hook: payload inválido: %v\n", err)
		return replyAllow(false, stdout)
	}

	tool := in.toolName()
	if !editTools[tool] {
		return replyAllow(in.isAntigravity(), stdout)
	}

	path := in.filePath()
	if path == "" {
		return replyAllow(in.isAntigravity(), stdout)
	}

	start := in.startDir()
	cfg, err := config.Load(start)
	if err != nil {
		if err == config.ErrNotFound {
			// Projeto não adota o pipeline. Nada a fazer.
			return replyAllow(in.isAntigravity(), stdout)
		}
		// Config presente mas quebrada: bloqueia, porque seguir em frente
		// daria a falsa impressão de que a proteção está ativa.
		msg := fmt.Sprintf("pipeline-hook: %v\nCorrija o %s antes de editar arquivos.\n", err, config.FileName)
		fmt.Fprint(stderr, msg)
		return replyDeny(in.isAntigravity(), msg, stdout)
	}

	if !cfg.IsProtected(path) {
		return replyAllow(in.isAntigravity(), stdout)
	}
	if cfg.IsEditorAgent(in.agentType()) {
		return replyAllow(in.isAntigravity(), stdout)
	}

	msg := blockMessage(path, in.agentType(), cfg)
	fmt.Fprint(stderr, msg)
	return replyDeny(in.isAntigravity(), msg, stdout)
}

func replyAllow(isAntigravity bool, stdout io.Writer) int {
	if isAntigravity {
		_ = json.NewEncoder(stdout).Encode(antigravityOutput{Decision: "allow"})
	}
	return exitAllow
}

func replyDeny(isAntigravity bool, reason string, stdout io.Writer) int {
	if isAntigravity {
		_ = json.NewEncoder(stdout).Encode(antigravityOutput{
			Decision: "deny",
			Reason:   reason,
		})
		return exitAllow
	}
	return exitBlock
}

func blockMessage(path, agentType string, cfg *config.Config) string {
	var b strings.Builder

	b.WriteString("BLOQUEADO pelo pipeline-ai: edição direta em caminho protegido.\n\n")
	fmt.Fprintf(&b, "  arquivo: %s\n", path)
	if agentType == "" {
		b.WriteString("  origem:  sessão principal\n")
	} else {
		fmt.Fprintf(&b, "  origem:  agent %q (sem permissão de escrita)\n", agentType)
	}

	b.WriteString("\nCódigo só é alterado por agents do pipeline. Autorizados:\n")
	if len(cfg.EditorAgents) == 0 {
		b.WriteString("  (nenhum configurado em editor_agents)\n")
	}
	for _, a := range cfg.EditorAgents {
		fmt.Fprintf(&b, "  - %s\n", a)
	}

	b.WriteString("\nPróximo passo: abra a demanda com /triage, que roteia para o agent correto.\n")
	b.WriteString("Para alterar o escopo protegido, edite protected_paths em " + config.FileName + ".\n")

	return b.String()
}
