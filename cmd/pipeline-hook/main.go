// Command pipeline-hook é o hook PreToolUse que impede a sessão principal de
// editar código diretamente, forçando a delegação aos agents do pipeline.
//
// É a camada 2 da decisão D-11. A camada 1, que de fato barra, são as deny
// rules de permissão em .claude/settings.json. Este hook existe para dar ao
// modelo uma razão legível, que o Claude Code repassa ao agent principal a
// partir do stderr quando o exit code é 2.
//
// Entrada: JSON no stdin, no formato do PreToolUse.
// Saída:
//
//	exit 0 → permitido
//	exit 2 → bloqueado (a razão vai no stderr e é mostrada ao modelo)
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

// hookInput é o subconjunto do payload do PreToolUse que interessa aqui.
//
// Atenção ao formato: o caminho do arquivo vem em tool_input.file_path, e não
// em file_path na raiz. O script bash original lia a raiz, por isso nunca
// bloqueou nada. Ver D-11.
type hookInput struct {
	ToolName  string `json:"tool_name"`
	AgentType string `json:"agent_type"`
	CWD       string `json:"cwd"`
	ToolInput struct {
		FilePath string `json:"file_path"`
	} `json:"tool_input"`
}

// editTools são as ferramentas que alteram arquivos.
var editTools = map[string]bool{
	"Edit":         true,
	"Write":        true,
	"MultiEdit":    true,
	"NotebookEdit": true,
}

func main() {
	os.Exit(run(os.Stdin, os.Stderr))
}

func run(stdin io.Reader, stderr io.Writer) int {
	raw, err := io.ReadAll(stdin)
	if err != nil {
		// Sem entrada legível não há o que avaliar. A camada de permissões
		// continua ativa, então liberar aqui não abre brecha real.
		fmt.Fprintf(stderr, "pipeline-hook: falha ao ler stdin: %v\n", err)
		return exitAllow
	}

	var in hookInput
	if err := json.Unmarshal(raw, &in); err != nil {
		fmt.Fprintf(stderr, "pipeline-hook: payload inválido: %v\n", err)
		return exitAllow
	}

	if !editTools[in.ToolName] {
		return exitAllow
	}

	path := strings.TrimSpace(in.ToolInput.FilePath)
	if path == "" {
		return exitAllow
	}

	start := in.CWD
	if start == "" {
		if wd, err := os.Getwd(); err == nil {
			start = wd
		} else {
			start = "."
		}
	}

	cfg, err := config.Load(start)
	if err != nil {
		if err == config.ErrNotFound {
			// Projeto não adota o pipeline. Nada a fazer.
			return exitAllow
		}
		// Config presente mas quebrada: bloqueia, porque seguir em frente
		// daria a falsa impressão de que a proteção está ativa.
		fmt.Fprintf(stderr, "pipeline-hook: %v\n", err)
		fmt.Fprintln(stderr, "Corrija o "+config.FileName+" antes de editar arquivos.")
		return exitBlock
	}

	if !cfg.IsProtected(path) {
		return exitAllow
	}
	if cfg.IsEditorAgent(in.AgentType) {
		return exitAllow
	}

	fmt.Fprint(stderr, blockMessage(path, in.AgentType, cfg))
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
