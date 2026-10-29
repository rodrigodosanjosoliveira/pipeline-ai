// Command pipeline-ai é a CLI unificada do toolkit de governança de agentes pipeline-ai.
//
// Suporta os subcomandos:
//   - init: inicializa novo projeto no padrão (desktop ou api)
//   - hook: executa o hook PreToolUse que bloqueia edição direta
//   - validate: valida relatório de delivery e salvaguardas de git
//   - sync: sincroniza/atualiza skills e regras (.agents/) em projetos existentes
//   - delivery: gerencia a delivery ativa (open, close, status)
//   - sensor: executa sensores estruturados do projeto (-json suportado)
//   - module: gerencia módulos opcionais (list, enable, disable)
//   - doctor: diagnostica o ambiente local e configurações do projeto
//   - version: exibe a versão instalada
//   - help: exibe mensagem de ajuda
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/dashboard"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/handoff"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/hook"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/mcp"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/plugin"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/runner"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/scaffold"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate"
)

const version = "v0.8.0"

func main() {
	os.Exit(runWithStdin(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWithStdin(args, os.Stdin, stdout, stderr)
}

func runWithStdin(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}

	cmd := strings.ToLower(strings.TrimSpace(args[0]))
	cmdArgs := args[1:]

	switch cmd {
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0

	case "version", "-v", "--version":
		fmt.Fprintf(stdout, "pipeline-ai %s\n", version)
		return 0

	case "hook":
		return hook.Run(stdin, stdout, stderr)

	case "validate":
		return runValidate(cmdArgs, stdout, stderr)

	case "handoff":
		return runHandoff(cmdArgs, stdout, stderr)

	case "run":
		return runRun(cmdArgs, stdout, stderr)

	case "init":
		return runInit(cmdArgs, stdout, stderr)

	case "sync":
		return runSync(cmdArgs, stdout, stderr)

	case "delivery":
		return runDelivery(cmdArgs, stdout, stderr)

	case "doctor":
		return runDoctor(cmdArgs, stdout, stderr)

	case "module":
		return runModule(cmdArgs, stdout, stderr)

	case "sensor":
		return runSensor(cmdArgs, stdout, stderr)

	case "metrics":
		return runMetrics(cmdArgs, stdout, stderr)

	case "dashboard":
		return runDashboard(cmdArgs, stdout, stderr)

	case "mcp":
		return runMCP(cmdArgs, stdin, stdout, stderr)

	case "plugin":
		return runPlugin(cmdArgs, stdout, stderr)

	case "eval":
		return runEval(cmdArgs, stdin, stdout, stderr)

	default:
		fmt.Fprintf(stderr, "pipeline-ai: comando desconhecido: %q\nExecute 'pipeline-ai help' para ver a lista de comandos.\n", cmd)
		return 1
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Uso: pipeline-ai <comando> [argumentos]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Comandos disponíveis:")
	fmt.Fprintln(w, "  init       Inicializa um novo projeto com o padrão pipeline-ai")
	fmt.Fprintln(w, "  hook       Executa o hook PreToolUse que impede edição direta em caminhos protegidos")
	fmt.Fprintln(w, "  run        Executa a suíte integrada de gates de governança (doctor, sensor, validate)")
	fmt.Fprintln(w, "  validate   Valida o relatório de delivery e salvaguardas de segurança antes do PR")
	fmt.Fprintln(w, "  handoff    Valida integridade e contratos de blocos de handoff (validate [-json])")
	fmt.Fprintln(w, "  sync       Sincroniza skills, regras e hooks em .agents/ a partir dos templates")
	fmt.Fprintln(w, "  delivery   Gerencia o ciclo de vida da delivery ativa (open, close, status)")
	fmt.Fprintln(w, "  doctor     Diagnostica o ambiente local (Go 1.24+, git, gh, hooks e .pipeline-ai.json)")
	fmt.Fprintln(w, "  module     Gerencia módulos opcionais do pipeline (list, enable, disable)")
	fmt.Fprintln(w, "  sensor     Executa os sensores de qualidade (fmt, vet, test) com saída estruturada")
	fmt.Fprintln(w, "  metrics    Exibe métricas de engenharia e governança DORA (summary, list)")
	fmt.Fprintln(w, "  dashboard  Gera e renderiza dashboard HTML interativo de engenharia e métricas")
	fmt.Fprintln(w, "  mcp        Executa o servidor MCP sobre stdio (JSON-RPC 2.0)")
	fmt.Fprintln(w, "  eval       Submete uma avaliação semântica calibrada (System One / Jev)")
	fmt.Fprintln(w, "  plugin     Gerencia o bundle e instalação do plugin Antigravity (verify, install)")
	fmt.Fprintln(w, "  version    Exibe a versão do pipeline-ai")
	fmt.Fprintln(w, "  help       Exibe esta mensagem de ajuda")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Execute 'pipeline-ai <comando> --help' para detalhes sobre opções específicas.")
}

func runRun(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", ".", "diretório raiz do projeto")
	delivery := fs.String("delivery", "", "ID da delivery a validar (ex: DEL-001)")
	report := fs.String("report", "", "caminho do relatório de delivery")
	noGit := fs.Bool("no-git", false, "pula verificações dependentes do git")
	baseRef := fs.String("base", "", "referência git para o diff")
	asJSON := fs.Bool("json", false, "emite os resultados consolidados em JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	opts := runner.Options{
		Dir:        *dir,
		DeliveryID: *delivery,
		ReportPath: *report,
		BaseRef:    *baseRef,
		NoGit:      *noGit,
	}

	res, err := runner.Run(opts)
	if err != nil {
		fmt.Fprintf(stderr, "pipeline-ai run: erro: %v\n", err)
		return 1
	}

	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(res); err != nil {
			fmt.Fprintf(stderr, "erro ao serializar JSON: %v\n", err)
			return 1
		}
	} else {
		fmt.Fprintln(stdout, "=== pipeline-ai runner ===")
		if res.DeliveryID != "" {
			fmt.Fprintf(stdout, "Delivery: %s\n", res.DeliveryID)
		}
		fmt.Fprintln(stdout)
		for _, step := range res.Steps {
			fmt.Fprintf(stdout, " [%s] %-10s : %s\n", step.Status, step.Name, step.Message)
		}
		fmt.Fprintln(stdout)
		fmt.Fprintf(stdout, "Status geral: %s\n", res.Overall)
	}

	if !res.OK {
		return 1
	}
	return 0
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)

	baseRef := fs.String("base", "", "referência git para o diff (default: base_ref do config)")
	noGit := fs.Bool("no-git", false, "pula as checagens que dependem do git")
	asJSON := fs.Bool("json", false, "emite o resultado estruturado em JSON")

	if err := fs.Parse(args); err != nil {
		return validate.ExitBadUsage
	}

	if fs.NArg() != 1 {
		fmt.Fprintf(stderr, "uso: pipeline-ai validate [flags] <caminho-do-relatorio.md>\n")
		return validate.ExitBadUsage
	}

	if *asJSON {
		return validate.RunJSON(fs.Arg(0), *baseRef, *noGit, stdout, stderr)
	}
	return validate.Run(fs.Arg(0), *baseRef, *noGit, stdout, stderr)
}

func runHandoff(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "uso: pipeline-ai handoff <subcomando> [flags] [argumento]")
		fmt.Fprintln(stderr, "Subcomandos disponíveis: validate")
		return 1
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "validate":
		fs := flag.NewFlagSet("handoff validate", flag.ContinueOnError)
		fs.SetOutput(stderr)
		asJSON := fs.Bool("json", false, "emite resultado estruturado em JSON")
		if err := fs.Parse(args[1:]); err != nil {
			return 1
		}
		if fs.NArg() == 0 {
			fmt.Fprintln(stderr, "uso: pipeline-ai handoff validate [-json] <arquivo-ou-string>")
			return 1
		}

		target := fs.Arg(0)
		var content string

		fileInfo, statErr := os.Stat(target)
		if statErr == nil && !fileInfo.IsDir() {
			data, err := os.ReadFile(target)
			if err != nil {
				fmt.Fprintf(stderr, "erro ao ler arquivo %s: %v\n", target, err)
				return 1
			}
			content = string(data)
		} else {
			isBlock := strings.Contains(target, "\n") || strings.Contains(target, "DE:")
			if !isBlock && (strings.Contains(target, "/") || strings.Contains(target, `\`) || strings.HasSuffix(target, ".txt") || strings.HasSuffix(target, ".md")) {
				fmt.Fprintf(stderr, "arquivo não encontrado: %s\n", target)
				return 1
			}
			content = target
		}

		res, err := handoff.ValidateContent(content)
		if res == nil {
			errMsg := "erro ao validar handoff"
			if err != nil {
				errMsg = err.Error()
			}
			res = &handoff.ValidationResult{
				Valid:  false,
				Errors: []string{errMsg},
			}
		}

		if *asJSON {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(res); err != nil {
				fmt.Fprintf(stderr, "erro ao serializar JSON: %v\n", err)
				return 1
			}
		} else {
			if res.Valid {
				fmt.Fprintln(stdout, "Handoff Válido: OK")
				if res.Handoff != nil {
					fmt.Fprintf(stdout, "  DE:       %s\n", res.Handoff.De)
					fmt.Fprintf(stdout, "  PARA:     %s\n", res.Handoff.Para)
					fmt.Fprintf(stdout, "  DELIVERY: %s\n", res.Handoff.DeliveryID)
					fmt.Fprintf(stdout, "  STATUS:   %s\n", res.Handoff.Status)
					fmt.Fprintf(stdout, "  PROXIMO:  %s\n", res.Handoff.Proximo)
				}
			} else {
				fmt.Fprintln(stderr, "Handoff Inválido:")
				for _, e := range res.Errors {
					fmt.Fprintf(stderr, "  - [ERRO] %s\n", e)
				}
				for _, w := range res.Warnings {
					fmt.Fprintf(stderr, "  - [AVISO] %s\n", w)
				}
			}
		}

		if !res.Valid {
			return 1
		}
		return 0

	default:
		fmt.Fprintf(stderr, "pipeline-ai handoff: subcomando desconhecido: %q\n", sub)
		return 1
	}
}

func runSensor(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sensor", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", ".", "diretório raiz do projeto a ser analisado")
	asJSON := fs.Bool("json", false, "emite os resultados estruturados em JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	res, err := sensor.Run(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "sensor: erro na execução: %v\n", err)
		return 1
	}

	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(res); err != nil {
			fmt.Fprintf(stderr, "sensor: erro ao serializar JSON: %v\n", err)
			return 1
		}
	} else {
		fmt.Fprintf(stdout, "Status geral: %s\n", res.Overall)
		fmt.Fprintf(stdout, "Testes executados: %d\n", res.TestsRun)
		for _, s := range res.Sensors {
			fmt.Fprintf(stdout, "- [%s] %s: %s\n", s.Status, s.Name, s.Message)
		}
	}

	if !res.OK {
		return 1
	}
	return 0
}

func runSync(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", ".", "diretório raiz do projeto a ser sincronizado")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if err := scaffold.Sync(*dir, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "ERRO: %v\n", err)
		return 1
	}
	return 0
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", ".", "diretório raiz do projeto a ser verificado")
	asJSON := fs.Bool("json", false, "emite o diagnóstico estruturado em formato JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *asJSON {
		return doctor.RunJSON(*dir, stdout)
	}
	return doctor.Run(*dir, stdout, stderr)
}

func runModule(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "uso: pipeline-ai module <list|enable|disable> [nome] [-dir <caminho>]")
		return 1
	}

	dir := "."
	var filtered []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-dir" && i+1 < len(args) {
			dir = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "-dir=") {
			dir = strings.TrimPrefix(arg, "-dir=")
		} else {
			filtered = append(filtered, arg)
		}
	}

	if len(filtered) == 0 {
		fmt.Fprintln(stderr, "uso: pipeline-ai module <list|enable|disable> [nome] [-dir <caminho>]")
		return 1
	}

	subcmd := strings.ToLower(filtered[0])
	switch subcmd {
	case "list":
		cfg, err := config.Load(dir)
		if err != nil {
			fmt.Fprintf(stderr, "module: erro ao carregar configuração: %v\n", err)
			return 1
		}
		var mods []string
		for m := range config.AllowedModules {
			mods = append(mods, m)
		}
		sort.Strings(mods)
		fmt.Fprintln(stdout, "Módulos opcionais do pipeline-ai:")
		for _, m := range mods {
			status := "[desabilitado]"
			if cfg.IsModuleEnabled(m) {
				status = "[habilitado / ativo]"
			}
			fmt.Fprintf(stdout, "  - %-20s %s\n", m, status)
		}
		return 0

	case "enable":
		if len(filtered) < 2 {
			fmt.Fprintln(stderr, "module enable: nome do módulo não especificado")
			return 1
		}
		modName := filtered[1]
		cfg, err := config.Load(dir)
		if err != nil {
			fmt.Fprintf(stderr, "module: erro ao carregar configuração: %v\n", err)
			return 1
		}
		if err := cfg.EnableModule(modName); err != nil {
			fmt.Fprintf(stderr, "module enable: erro: %v\n", err)
			return 1
		}
		if err := cfg.Save(); err != nil {
			fmt.Fprintf(stderr, "module enable: erro ao salvar: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Módulo %q habilitado com sucesso.\n", modName)
		return 0

	case "disable":
		if len(filtered) < 2 {
			fmt.Fprintln(stderr, "module disable: nome do módulo não especificado")
			return 1
		}
		modName := filtered[1]
		cfg, err := config.Load(dir)
		if err != nil {
			fmt.Fprintf(stderr, "module: erro ao carregar configuração: %v\n", err)
			return 1
		}
		if err := cfg.DisableModule(modName); err != nil {
			fmt.Fprintf(stderr, "module disable: erro: %v\n", err)
			return 1
		}
		if err := cfg.Save(); err != nil {
			fmt.Fprintf(stderr, "module disable: erro ao salvar: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Módulo %q desabilitado com sucesso.\n", modName)
		return 0

	default:
		fmt.Fprintf(stderr, "module: subcomando inválido ou desconhecido: %q\n", subcmd)
		return 1
	}
}

func runMCP(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", ".", "diretório raiz do projeto")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	s := mcp.NewServer(*dir)
	if err := s.ServeStream(stdin, stdout, stderr); err != nil && err != io.EOF {
		fmt.Fprintf(stderr, "mcp: erro no servidor: %v\n", err)
		return 1
	}
	return 0
}

func runDelivery(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printDeliveryUsage(stderr)
		return 1
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	subArgs := args[1:]

	switch sub {
	case "open":
		return runDeliveryOpen(subArgs, stdout, stderr)
	case "close":
		return runDeliveryClose(subArgs, stdout, stderr)
	case "status":
		return runDeliveryStatus(subArgs, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "pipeline-ai delivery: subcomando desconhecido %q (use open, close ou status)\n", sub)
		return 1
	}
}

func printDeliveryUsage(w io.Writer) {
	fmt.Fprintln(w, "Uso: pipeline-ai delivery <open|close|status> [opções]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Subcomandos:")
	fmt.Fprintln(w, "  open <DEL-ID> [-task <TASK_CODE>] [-ttl <duration>] [-dir <dir>]")
	fmt.Fprintln(w, "  close [-dir <dir>]")
	fmt.Fprintln(w, "  status [-dir <dir>]")
}

func runDeliveryOpen(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("delivery open", flag.ContinueOnError)
	fs.SetOutput(stderr)

	task := fs.String("task", "", "código da tarefa (opcional)")
	ttlStr := fs.String("ttl", "168h", "tempo de validade (TTL), ex: 2h, 48h, 7d")
	dir := fs.String("dir", ".", "diretório raiz do projeto")

	// Reorganiza argumentos para permitir flags antes ou depois do delivery_id
	var flagArgs []string
	var posArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		} else {
			posArgs = append(posArgs, arg)
		}
	}

	if err := fs.Parse(flagArgs); err != nil {
		return 1
	}

	if len(posArgs) == 0 || strings.TrimSpace(posArgs[0]) == "" {
		fmt.Fprintln(stderr, "ERRO: delivery_id é obrigatório. Exemplo: pipeline-ai delivery open DEL-001")
		return 1
	}

	delivID := strings.TrimSpace(posArgs[0])

	ttl, err := time.ParseDuration(*ttlStr)
	if err != nil {
		fmt.Fprintf(stderr, "ERRO: ttl %q inválido: %v\n", *ttlStr, err)
		return 1
	}

	if err := state.Open(*dir, delivID, *task, ttl); err != nil {
		fmt.Fprintf(stderr, "ERRO ao abrir delivery: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Delivery %s aberta com sucesso (TTL: %v).\n", delivID, ttl)
	return 0
}

func runDeliveryClose(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("delivery close", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", ".", "diretório raiz do projeto")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if err := state.Close(*dir); err != nil {
		fmt.Fprintf(stderr, "ERRO ao fechar delivery: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "Delivery ativa fechada com sucesso.")
	return 0
}

func runDeliveryStatus(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("delivery status", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", ".", "diretório raiz do projeto")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	d, err := state.Get(*dir)
	if err != nil || d == nil {
		fmt.Fprintln(stdout, "nenhuma delivery ativa")
		return 0
	}

	active, actErr := state.IsActive(*dir)
	if !active {
		if errors.Is(actErr, state.ErrExpired) {
			fmt.Fprintf(stdout, "Delivery ativa: %s (EXPIRADA em %s)\n", d.DeliveryID, d.ExpiresAt.Format(time.RFC3339))
			if d.TaskCode != "" {
				fmt.Fprintf(stdout, "Task:           %s\n", d.TaskCode)
			}
			return 0
		}
		fmt.Fprintln(stdout, "nenhuma delivery ativa")
		return 0
	}

	fmt.Fprintf(stdout, "Delivery ativa: %s\n", d.DeliveryID)
	if d.TaskCode != "" {
		fmt.Fprintf(stdout, "Task:           %s\n", d.TaskCode)
	}
	fmt.Fprintf(stdout, "Aberta em:      %s\n", d.OpenedAt.Format(time.RFC3339))
	fmt.Fprintf(stdout, "Expira em:      %s\n", d.ExpiresAt.Format(time.RFC3339))
	return 0
}

func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var opts scaffold.Options
	fs.StringVar(&opts.Name, "name", "", "Nome do projeto (obrigatório)")
	fs.StringVar(&opts.Type, "type", "", "Formato do projeto: desktop, api ou cli")
	fs.StringVar(&opts.Database, "db", "", "Banco de dados: postgres, mongo ou dynamo (default: postgres)")
	fs.StringVar(&opts.Module, "module", "", "Caminho do módulo Go (ex: github.com/user/repo)")
	fs.StringVar(&opts.TargetDir, "dir", ".", "Diretório de destino (padrão: diretório atual)")
	fs.StringVar(&opts.Summary, "summary", "", "Uma frase descritiva sobre o projeto")
	noGit := fs.Bool("no-git", false, "Não inicializa repositório Git")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	opts.InitGit = !*noGit
	opts.Stdout = stdout
	opts.Stderr = stderr

	// Se executado sem flags interativas e name não foi fornecido
	if opts.Name == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Nome do projeto: ")
		text, _ := reader.ReadString('\n')
		opts.Name = strings.TrimSpace(text)
	}

	if opts.Type == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("\nEscolha o formato do projeto:")
		fmt.Println("  1) desktop (Go + Wails v3 + React num binário único)")
		fmt.Println("  2) api     (Servidor Go API + UI React)")
		fmt.Println("  3) cli     (Aplicação de linha de comando ou terminal em Go)")
		fmt.Print("Opção [1/2/3, default: 1]: ")
		text, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(text)
		switch choice {
		case "2", "api":
			opts.Type = "api"
		case "3", "cli", "tui":
			opts.Type = "cli"
		default:
			opts.Type = "desktop"
		}
	}

	if opts.Database == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("\nEscolha o banco de dados:")
		fmt.Println("  1) PostgreSQL (Relacional)")
		fmt.Println("  2) MongoDB    (Documento)")
		fmt.Println("  3) DynamoDB   (NoSQL Key-Value)")
		fmt.Print("Opção [1/2/3, default: 1]: ")
		text, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(text)
		switch choice {
		case "2", "mongo", "mongodb":
			opts.Database = "MongoDB"
		case "3", "dynamo", "dynamodb":
			opts.Database = "DynamoDB"
		default:
			opts.Database = "PostgreSQL"
		}
	}

	if opts.Module == "" {
		opts.Module = opts.Name
	}

	fmt.Fprintf(stdout, "\nInicializando projeto %q (%s, %s)...\n", opts.Name, opts.Type, opts.Database)
	if err := scaffold.Scaffold(opts); err != nil {
		fmt.Fprintf(stderr, "\nERRO ao inicializar projeto: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "\nSucesso! Projeto configurado com o padrão pipeline-ai.")
	fmt.Fprintf(stdout, "Diretório: %s\n\n", opts.TargetDir)
	return 0
}

func runMetrics(args []string, stdout, stderr io.Writer) int {
	var subcmd string
	var flagArgs []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if (arg == "-dir" || arg == "-window") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else if subcmd == "" {
			subcmd = strings.ToLower(arg)
		} else {
			flagArgs = append(flagArgs, arg)
		}
	}

	if subcmd == "" {
		subcmd = "summary"
	}

	fs := flag.NewFlagSet("metrics "+subcmd, flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", ".", "diretório raiz do projeto")
	asJSON := fs.Bool("json", false, "emite os resultados estruturados em JSON")
	windowDays := fs.Float64("window", 0, "janela de cálculo em dias (opcional)")

	if err := fs.Parse(flagArgs); err != nil {
		return 1
	}

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "metrics: diretório inválido: %v\n", err)
		return 1
	}

	switch subcmd {
	case "summary":
		var opts []metrics.DORAOptions
		if *windowDays > 0 {
			opts = append(opts, metrics.DORAOptions{
				Window: time.Duration(*windowDays * 24 * float64(time.Hour)),
			})
		}

		dora, err := metrics.GetDORA(absDir, opts...)
		if err != nil {
			fmt.Fprintf(stderr, "metrics summary: erro ao calcular métricas DORA: %v\n", err)
			return 1
		}

		if *asJSON {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(dora); err != nil {
				fmt.Fprintf(stderr, "metrics summary: erro ao serializar JSON: %v\n", err)
				return 1
			}
			return 0
		}

		fmt.Fprintln(stdout, "Métricas DORA do Pipeline:")
		fmt.Fprintf(stdout, "  Deployment Frequency:    %.2f deploys/dia (total: %d)\n", dora.DeploymentFrequency, dora.DeploymentsTotal)
		fmt.Fprintf(stdout, "  Lead Time for Changes:   %s\n", dora.LeadTimeForChanges)
		fmt.Fprintf(stdout, "  Change Failure Rate:     %.1f%% (falhas: %d)\n", dora.ChangeFailureRate, dora.FailedDeployments)
		fmt.Fprintf(stdout, "  Time to Restore Service: %s (restaurações: %d)\n", dora.TimeToRestore, dora.RestoresTotal)
		return 0

	case "list":
		data, err := metrics.Load(absDir)
		if err != nil {
			fmt.Fprintf(stderr, "metrics list: erro ao carregar métricas: %v\n", err)
			return 1
		}

		if *asJSON {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(data); err != nil {
				fmt.Fprintf(stderr, "metrics list: erro ao serializar JSON: %v\n", err)
				return 1
			}
			return 0
		}

		fmt.Fprintln(stdout, "Entregas registradas:")
		if len(data.Deliveries) == 0 {
			fmt.Fprintln(stdout, "  (nenhuma entrega registrada)")
		} else {
			for _, d := range data.Deliveries {
				status := "SUCESSO"
				if !d.Success {
					status = "FALHA"
				}
				fmt.Fprintf(stdout, "  - [%s] %s (%s) - Lead time: %s\n", status, d.DeliveryID, d.TaskCode, d.LeadTime)
			}
		}

		fmt.Fprintln(stdout, "\nEventos recentes:")
		if len(data.Events) == 0 {
			fmt.Fprintln(stdout, "  (nenhum evento registrado)")
		} else {
			for _, e := range data.Events {
				status := "SUCESSO"
				if !e.Success {
					status = "FALHA"
				}
				fmt.Fprintf(stdout, "  - [%s] %s %s (%s)\n", status, e.Type, e.DeliveryID, e.Timestamp.Format(time.RFC3339))
			}
		}
		return 0

	default:
		fmt.Fprintf(stderr, "metrics: subcomando desconhecido: %q (use 'summary' ou 'list')\n", subcmd)
		return 1
	}
}

func runDashboard(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dashboard", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", ".", "diretório raiz do projeto")
	output := fs.String("output", "", "caminho para gravar o arquivo HTML")
	asJSON := fs.Bool("json", false, "emite os dados do dashboard em JSON")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "dashboard: diretório inválido: %v\n", err)
		return 1
	}

	data, err := dashboard.CollectDashboardData(absDir)
	if err != nil {
		fmt.Fprintf(stderr, "dashboard: erro ao coletar dados: %v\n", err)
		return 1
	}

	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(data); err != nil {
			fmt.Fprintf(stderr, "dashboard: erro ao serializar JSON: %v\n", err)
			return 1
		}
		return 0
	}

	html, err := dashboard.RenderHTML(data)
	if err != nil {
		fmt.Fprintf(stderr, "dashboard: erro ao renderizar HTML: %v\n", err)
		return 1
	}

	if *output != "" {
		if err := os.WriteFile(*output, []byte(html), 0644); err != nil {
			fmt.Fprintf(stderr, "dashboard: erro ao gravar arquivo %s: %v\n", *output, err)
			return 1
		}
		return 0
	}

	fmt.Fprint(stdout, html)
	return 0
}

func runPlugin(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printPluginUsage(stderr)
		return 1
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	subArgs := args[1:]

	switch sub {
	case "verify":
		return runPluginVerify(subArgs, stdout, stderr)
	case "install":
		return runPluginInstall(subArgs, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "pipeline-ai plugin: subcomando desconhecido %q (use verify ou install)\n", sub)
		return 1
	}
}

func printPluginUsage(w io.Writer) {
	fmt.Fprintln(w, "Uso: pipeline-ai plugin <verify|install> [opções]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Subcomandos:")
	fmt.Fprintln(w, "  verify  [-dir <dir>]")
	fmt.Fprintln(w, "  install [-src <dir>] [-target <dir>]")
}

func runPluginVerify(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("plugin verify", flag.ContinueOnError)
	fs.SetOutput(stderr)

	dir := fs.String("dir", "plugin", "diretório do bundle do plugin a verificar")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if err := plugin.VerifyBundle(*dir); err != nil {
		fmt.Fprintf(stderr, "plugin verify: falha na verificação: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Bundle do plugin em %q é válido [OK].\n", *dir)
	return 0
}

func runPluginInstall(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("plugin install", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var src string
	fs.StringVar(&src, "dir", "plugin", "diretório de origem do bundle do plugin")
	fs.StringVar(&src, "src", "plugin", "diretório de origem do bundle do plugin")
	target := fs.String("target", "", "diretório de destino da instalação")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	targetDir := strings.TrimSpace(*target)
	if targetDir == "" {
		fmt.Fprintln(stderr, "plugin install: a flag -target é obrigatória")
		return 1
	}

	if strings.HasPrefix(targetDir, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			targetDir = filepath.Join(home, targetDir[2:])
		}
	}

	if err := plugin.InstallToDir(src, targetDir); err != nil {
		fmt.Fprintf(stderr, "plugin install: erro ao instalar plugin: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Plugin instalado com sucesso em %q [OK].\n", targetDir)
	return 0
}

func runEval(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	fs.SetOutput(stderr)

	stateFlag := fs.String("state", "", "caminho de arquivo ou string de estado para avaliação")
	questionFlag := fs.String("question", "", "pergunta ou critério semântico a ser avaliado")
	thresholdFlag := fs.Float64("threshold", 0.8, "score mínimo de confiança requerido (0.0 a 1.0)")
	jsonFlag := fs.Bool("json", false, "saída em formato JSON estruturado")
	endpointFlag := fs.String("endpoint", "", "endpoint HTTP do sensor semântico (Jev)")
	apiKeyFlag := fs.String("api-key", "", "chave de API do sensor semântico")
	timeoutFlag := fs.Duration("timeout", 10*time.Second, "tempo limite para a requisição de avaliação")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	const maxStateBytes = 512 * 1024

	var stateContent string
	rawState := *stateFlag

	if rawState != "" {
		if fi, err := os.Stat(rawState); err == nil && !fi.IsDir() {
			if fi.Size() > maxStateBytes {
				fmt.Fprintf(stderr, "eval: tamanho do state excede o limite máximo permitido (512KB)\n")
				return 1
			}
			data, err := os.ReadFile(rawState)
			if err != nil {
				fmt.Fprintf(stderr, "eval: erro ao ler arquivo de state: %v\n", err)
				return 1
			}
			stateContent = string(data)
		} else {
			if len(rawState) > maxStateBytes {
				fmt.Fprintf(stderr, "eval: tamanho do state excede o limite máximo permitido (512KB)\n")
				return 1
			}
			stateContent = rawState
		}
	} else if stdin != nil {
		if f, ok := stdin.(*os.File); ok {
			stat, err := f.Stat()
			if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
				data, err := io.ReadAll(io.LimitReader(f, maxStateBytes+1))
				if err == nil && len(data) > 0 {
					if len(data) > maxStateBytes {
						fmt.Fprintf(stderr, "eval: tamanho do state lido do stdin excede o limite máximo permitido (512KB)\n")
						return 1
					}
					stateContent = string(data)
				}
			}
		} else {
			data, err := io.ReadAll(io.LimitReader(stdin, maxStateBytes+1))
			if err == nil && len(data) > 0 {
				if len(data) > maxStateBytes {
					fmt.Fprintf(stderr, "eval: tamanho do state excede o limite máximo permitido (512KB)\n")
					return 1
				}
				stateContent = string(data)
			}
		}
	}

	question := strings.TrimSpace(*questionFlag)
	if strings.TrimSpace(stateContent) == "" || question == "" {
		fmt.Fprintln(stderr, "eval: os parâmetros -state e -question são obrigatórios. Uso: pipeline-ai eval -state <string|arquivo> -question <pergunta> [opções]")
		return 1
	}

	if *thresholdFlag < 0.0 || *thresholdFlag > 1.0 {
		fmt.Fprintf(stderr, "eval: threshold inválido (deve estar entre 0.0 e 1.0): %f\n", *thresholdFlag)
		return 1
	}

	endpoint := strings.TrimSpace(*endpointFlag)
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("PIPELINE_SEMANTIC_ENDPOINT"))
	}
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("JEV_ENDPOINT"))
	}

	apiKey := strings.TrimSpace(*apiKeyFlag)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("PIPELINE_SEMANTIC_API_KEY"))
	}
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("JEV_API_KEY"))
	}

	if endpoint == "" || apiKey == "" {
		if *jsonFlag {
			res := map[string]any{
				"configured": false,
				"message":    "sensor semântico não configurado (JEV_ENDPOINT / JEV_API_KEY ausentes)",
			}
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintln(stderr, "eval: sensor semântico não configurado (JEV_ENDPOINT e JEV_API_KEY ausentes)")
		}
		return 1
	}

	timeout := *timeoutFlag
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := semantic.NewClient(semantic.ClientConfig{
		Endpoint:     endpoint,
		APIKey:       apiKey,
		Timeout:      timeout,
		MaxBodyBytes: 10 * 1024 * 1024,
	})

	decision, err := client.EvalNoul(ctx, semantic.EvalRequest{
		State:     stateContent,
		Question:  question,
		Threshold: *thresholdFlag,
	})

	secrets := []string{apiKey, os.Getenv("JEV_API_KEY"), os.Getenv("PIPELINE_SEMANTIC_API_KEY")}

	if err != nil {
		sanitizedMsg := redactSecrets(err.Error(), secrets...)
		if *jsonFlag {
			res := map[string]any{
				"error": sanitizedMsg,
			}
			data, _ := json.MarshalIndent(res, "", "  ")
			fmt.Fprintln(stdout, string(data))
		} else {
			fmt.Fprintf(stderr, "eval: erro na avaliação semântica: %s\n", sanitizedMsg)
		}
		return 1
	}

	if *jsonFlag {
		data, err := json.MarshalIndent(decision, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "eval: erro ao serializar saída JSON: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, string(data))
		return 0
	}

	fmt.Fprintf(stdout, "Decisão: %v\nConfiança: %.2f\nRazão: %s\nLatência: %dms\n",
		decision.Decision, decision.Confidence, decision.Reason, decision.LatencyMs)
	return 0
}

func redactSecrets(msg string, secrets ...string) string {
	out := msg
	for _, s := range secrets {
		s = strings.TrimSpace(s)
		if s != "" && strings.Contains(out, s) {
			out = strings.ReplaceAll(out, s, "[REDACTED]")
		}
	}
	return out
}
