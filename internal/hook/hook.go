// Package hook implementa a lógica do PreToolUse hook para o Antigravity e compatibilidade legada.
package hook

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state"
)

const (
	ExitAllow = 0
	ExitBlock = 2
)

type toolCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type hookInput struct {
	ToolCall       *toolCall `json:"toolCall"`
	WorkspacePaths []string  `json:"workspacePaths"`
	SubagentID     string    `json:"subagentId"`

	ToolName  string `json:"tool_name"`
	AgentType string `json:"agent_type"`
	CWD       string `json:"cwd"`
	ToolInput *struct {
		FilePath     string `json:"file_path"`
		NotebookPath string `json:"notebook_path"`
	} `json:"tool_input"`
}

type antigravityOutput struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

var editTools = map[string]bool{
	"write_to_file":              true,
	"replace_file_content":       true,
	"multi_replace_file_content": true,
	"Edit":                       true,
	"Write":                      true,
	"MultiEdit":                  true,
	"NotebookEdit":               true,
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
		for _, k := range []string{"TargetFile", "target_file", "file_path", "FilePath", "path", "Path", "NotebookPath", "notebook_path"} {
			if val, ok := in.ToolCall.Args[k]; ok {
				if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
	}
	if in.ToolInput != nil {
		if s := strings.TrimSpace(in.ToolInput.FilePath); s != "" {
			return s
		}
		if s := strings.TrimSpace(in.ToolInput.NotebookPath); s != "" {
			return s
		}
	}
	return ""
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

// Run executa a validação de escrita do PreToolUse hook.
func Run(stdin io.Reader, stdout, stderr io.Writer) int {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		msg := fmt.Sprintf("BLOQUEADO pelo pipeline-ai: falha ao ler stdin do hook: %v\nExecute 'pipeline-ai doctor' para verificar o ambiente.\n", err)
		fmt.Fprint(stderr, msg)
		return replyDeny(false, msg, stdout)
	}

	var in hookInput
	if err := json.Unmarshal(raw, &in); err != nil {
		msg := fmt.Sprintf("BLOQUEADO pelo pipeline-ai: payload de hook ilegível ou JSON inválido: %v\nExecute 'pipeline-ai doctor' para verificar a integridade da IDE ou do ambiente.\n", err)
		fmt.Fprint(stderr, msg)
		isAnti := strings.Contains(string(raw), "toolCall")
		return replyDeny(isAnti, msg, stdout)
	}

	tool := in.toolName()
	if !editTools[tool] {
		return replyAllow(in.isAntigravity(), stdout)
	}

	if !in.isAntigravity() && in.ToolInput == nil {
		return replyAllow(false, stdout)
	}

	path := in.filePath()
	if path == "" {
		msg := fmt.Sprintf("BLOQUEADO pelo pipeline-ai: ferramenta de edição %q invocada com caminho de arquivo ausente ou vazio.\n", tool)
		fmt.Fprint(stderr, msg)
		return replyDeny(in.isAntigravity(), msg, stdout)
	}

	start := in.startDir()
	cfg, err := config.Load(start)
	if err != nil {
		if err == config.ErrNotFound {
			return replyAllow(in.isAntigravity(), stdout)
		}
		msg := fmt.Sprintf("pipeline-hook: %v\nCorrija o %s antes de editar arquivos.\n", err, config.FileName)
		fmt.Fprint(stderr, msg)
		return replyDeny(in.isAntigravity(), msg, stdout)
	}

	if cfg.IsImmutable(path) {
		msg := fmt.Sprintf("BLOQUEADO pelo pipeline-ai: caminho %q é estritamente imutável (nenhum agente tem permissão de escrita direta).\n", path)
		fmt.Fprint(stderr, msg)
		return replyDeny(in.isAntigravity(), msg, stdout)
	}

	if !cfg.IsProtected(path) {
		return replyAllow(in.isAntigravity(), stdout)
	}
	if cfg.IsEditorAgent(in.agentType()) {
		return replyAllow(in.isAntigravity(), stdout)
	}

	active, activeErr := state.IsActive(cfg.Root())
	if active {
		return replyAllow(in.isAntigravity(), stdout)
	}
	if errors.Is(activeErr, state.ErrExpired) {
		activeDel, _ := state.Get(cfg.Root())
		var delID, expStr string
		if activeDel != nil {
			delID = activeDel.DeliveryID
			expStr = activeDel.ExpiresAt.Format(time.RFC3339)
		}
		msg := fmt.Sprintf("BLOQUEADO pelo pipeline-ai: a delivery ativa %s expirou em %s. Renove com 'pipeline-ai delivery open %s' ou finalize com 'pipeline-ai delivery close'.\n", delID, expStr, delID)
		fmt.Fprint(stderr, msg)
		return replyDeny(in.isAntigravity(), msg, stdout)
	}

	msg := blockMessage(path, in.agentType(), cfg)
	fmt.Fprint(stderr, msg)
	return replyDeny(in.isAntigravity(), msg, stdout)
}

func replyAllow(isAntigravity bool, stdout io.Writer) int {
	if isAntigravity {
		_ = json.NewEncoder(stdout).Encode(antigravityOutput{Decision: "allow"})
	}
	return ExitAllow
}

func replyDeny(isAntigravity bool, reason string, stdout io.Writer) int {
	if isAntigravity {
		_ = json.NewEncoder(stdout).Encode(antigravityOutput{
			Decision: "deny",
			Reason:   reason,
		})
		return ExitAllow
	}
	return ExitBlock
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

	b.WriteString("\nCódigo só é alterado por agents do pipeline ou com uma delivery ativa.\n")
	b.WriteString("Para liberar edições na sessão principal, abra uma delivery com 'pipeline-ai delivery open <DEL-ID>'.\n")
	b.WriteString("Agents autorizados:\n")
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
