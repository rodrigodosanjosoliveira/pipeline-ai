// Package runner executa os gates de qualidade e governança (doctor, sensor, validate) de forma integrada.
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/config"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/doctor"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/metrics"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/sensor/semantic"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/state"
	"github.com/rodrigodosanjosoliveira/pipeline-ai/internal/validate"
)

// Status define o estado de execução de um gate ou do runner global.
type Status string

const (
	StatusPass Status = "PASS"
	StatusFail Status = "FAIL"
	StatusSkip Status = "SKIP"
	StatusWarn Status = "WARN"
)

// StepResult descreve a execução de um step individual do pipeline (doctor, sensor, validate).
type StepResult struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
	Details any    `json:"details,omitempty"`
}

// Result consolida os resultados de todos os gates executados.
type Result struct {
	Overall    Status       `json:"overall"`
	OK         bool         `json:"ok"`
	DeliveryID string       `json:"delivery_id,omitempty"`
	Steps      []StepResult `json:"steps"`
	Message    string       `json:"message,omitempty"`
}

// Options configura a execução do runner.
type Options struct {
	Dir              string `json:"dir"`
	DeliveryID       string `json:"delivery_id,omitempty"`
	ReportPath       string `json:"report_path,omitempty"`
	BaseRef          string `json:"base_ref,omitempty"`
	NoGit            bool   `json:"no_git"`
	MaxOutputBytes   int    `json:"max_output_bytes,omitempty"`
	VerboseOutput    bool   `json:"verbose_output,omitempty"`
	IncludeVuln      bool   `json:"include_vuln,omitempty"`
	SemanticGate     bool   `json:"semantic_gate,omitempty"`
	SemanticProvider string `json:"semantic_provider,omitempty"`
	SemanticModel    string `json:"semantic_model,omitempty"`
	SemanticEndpoint string `json:"semantic_endpoint,omitempty"`
	SemanticAPIKey   string `json:"-"`
}

// Run executa os gates integrados do pipeline.
func Run(opts Options) (*Result, error) {
	dir := opts.Dir
	if strings.TrimSpace(dir) == "" {
		dir = "."
	}
	absRoot, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("caminho inválido: %w", err)
	}

	cfg, err := config.Load(absRoot)
	if err != nil {
		cfg = &config.Config{
			DeliveriesDir: "docs-ai/deliveries",
			IndexFile:     "docs-ai/deliveries/INDEX.md",
			BaseRef:       "origin/main",
		}
	}
	if cfg.DeliveriesDir == "" {
		cfg.DeliveriesDir = "docs-ai/deliveries"
	}

	runStart := time.Now()

	deliveryID := strings.TrimSpace(opts.DeliveryID)
	reportPath := strings.TrimSpace(opts.ReportPath)

	if deliveryID == "" && reportPath == "" {
		if active, err := state.Get(absRoot); err == nil && active != nil && active.DeliveryID != "" {
			deliveryID = active.DeliveryID
		}
	}

	if reportPath == "" && deliveryID != "" {
		reportPath = filepath.Join(absRoot, cfg.DeliveriesDir, deliveryID, "report.md")
	} else if reportPath != "" && !filepath.IsAbs(reportPath) {
		reportPath = filepath.Join(absRoot, reportPath)
	}

	if deliveryID == "" && reportPath != "" {
		parentDir := filepath.Base(filepath.Dir(reportPath))
		if strings.HasPrefix(parentDir, "DEL-") {
			deliveryID = parentDir
		} else if active, err := state.Get(absRoot); err == nil && active != nil && active.DeliveryID != "" {
			deliveryID = active.DeliveryID
		}
	}

	var steps []StepResult

	// 1. doctor
	docStart := time.Now()
	docResults, docOK := doctor.Collect(absRoot)
	docDur := time.Since(docStart)
	docStep := StepResult{
		Name:    "doctor",
		Details: docResults,
	}
	if !docOK {
		docStep.Status = StatusFail
		var failures []string
		for _, r := range docResults {
			if r.Status == doctor.StatusFail {
				failures = append(failures, fmt.Sprintf("%s: %s", r.Name, r.Message))
			}
		}
		docStep.Message = fmt.Sprintf("falhas no ambiente (%d): %s", len(failures), strings.Join(failures, "; "))
	} else {
		docStep.Status = StatusPass
		docStep.Message = "Ambiente validado com sucesso"
	}
	steps = append(steps, docStep)
	_ = metrics.RecordStageDuration(absRoot, deliveryID, "doctor", docDur, docOK)

	// 2. sensor
	sensorStart := time.Now()
	sensorRes, err := sensor.RunWithOptions(absRoot, sensor.Options{
		MaxOutputBytes: opts.MaxOutputBytes,
		VerboseOutput:  opts.VerboseOutput,
		IncludeVuln:    opts.IncludeVuln,
	})
	sensorDur := time.Since(sensorStart)
	sensorStep := StepResult{
		Name:    "sensor",
		Details: sensorRes,
	}
	if err != nil {
		sensorStep.Status = StatusFail
		sensorStep.Message = fmt.Sprintf("erro ao executar sensores: %v", err)
	} else if sensorRes == nil || !sensorRes.OK || sensorRes.Overall != sensor.StatusPass {
		sensorStep.Status = StatusFail
		if sensorRes != nil && sensorRes.Message != "" {
			sensorStep.Message = fmt.Sprintf("sensores reprovados: %s", sensorRes.Message)
		} else {
			sensorStep.Message = "sensores reprovados"
		}
	} else {
		sensorStep.Status = StatusPass
		sensorStep.Message = fmt.Sprintf("sensores aprovados (%d testes executados)", sensorRes.TestsRun)
	}
	steps = append(steps, sensorStep)
	_ = metrics.RecordStageDuration(absRoot, deliveryID, "sensor", sensorDur, sensorStep.Status == StatusPass)

	// 3. validate
	valStart := time.Now()
	validateStep := StepResult{
		Name: "validate",
	}

	if reportPath == "" {
		validateStep.Status = StatusSkip
		validateStep.Message = "Nenhuma delivery ativa ou relatório informado; validação ignorada"
	} else {
		if fi, err := os.Stat(reportPath); err != nil {
			validateStep.Status = StatusFail
			if os.IsNotExist(err) {
				validateStep.Message = fmt.Sprintf("relatório de delivery não encontrado: %s", reportPath)
			} else {
				validateStep.Message = fmt.Sprintf("erro ao acessar relatório: %v", err)
			}
		} else if fi.IsDir() {
			validateStep.Status = StatusFail
			validateStep.Message = fmt.Sprintf("caminho do relatório aponta para um diretório: %s", reportPath)
		} else {
			var outBuf, errBuf bytes.Buffer
			baseRef := opts.BaseRef
			if baseRef == "" {
				baseRef = cfg.BaseRef
			}
			code := validate.RunJSON(reportPath, baseRef, opts.NoGit, &outBuf, &errBuf)
			var valRes validate.ValidationResult
			if err := json.Unmarshal(outBuf.Bytes(), &valRes); err != nil {
				validateStep.Status = StatusFail
				validateStep.Message = fmt.Sprintf("falha ao interpretar resultado da validação: %v (stderr: %s)", err, errBuf.String())
			} else {
				validateStep.Details = valRes
				if !valRes.OK || code != 0 {
					validateStep.Status = StatusFail
					var msgs []string
					for _, f := range valRes.Findings {
						if f.Level == "ERRO" {
							msgs = append(msgs, f.Message)
						}
					}
					validateStep.Message = fmt.Sprintf("validação reprovada com %d erro(s): %s", valRes.Errors, strings.Join(msgs, "; "))
				} else {
					validateStep.Status = StatusPass
					validateStep.Message = fmt.Sprintf("validação aprovada (%d erros, %d avisos)", valRes.Errors, valRes.Warnings)
				}
			}
		}
	}
	valDur := time.Since(valStart)
	steps = append(steps, validateStep)
	_ = metrics.RecordStageDuration(absRoot, deliveryID, "validate", valDur, validateStep.Status == StatusPass)

	// 4. semantic (gate opcional advisory)
	if opts.SemanticGate {
		semStart := time.Now()
		semStep := runSemanticGate(absRoot, deliveryID, reportPath, opts)
		semDur := time.Since(semStart)
		steps = append(steps, semStep)
		_ = metrics.RecordStageDuration(absRoot, deliveryID, "semantic", semDur, semStep.Status == StatusPass || semStep.Status == StatusSkip)
	}

	allOK := true
	for _, s := range steps {
		if s.Status == StatusFail {
			allOK = false
			break
		}
	}

	overall := StatusPass
	msg := "Pipeline executado com sucesso"
	if !allOK {
		overall = StatusFail
		msg = "Pipeline reprovado em um ou mais gates"
	}

	totalDur := time.Since(runStart)
	_ = metrics.RecordEvent(absRoot, metrics.Event{
		Type:       metrics.EventGate,
		DeliveryID: deliveryID,
		Timestamp:  time.Now().UTC(),
		Duration:   totalDur,
		Success:    allOK,
		Stage:      "runner",
		Message:    msg,
	})

	res := &Result{
		Overall:    overall,
		OK:         allOK,
		DeliveryID: deliveryID,
		Steps:      steps,
		Message:    msg,
	}

	return res, nil
}

func runSemanticGate(absRoot, deliveryID, reportPath string, opts Options) StepResult {
	step := StepResult{Name: "semantic"}

	provider := strings.ToLower(strings.TrimSpace(opts.SemanticProvider))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(os.Getenv("PIPELINE_SEMANTIC_PROVIDER")))
	}

	model := strings.TrimSpace(opts.SemanticModel)
	endpoint := strings.TrimSpace(opts.SemanticEndpoint)
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("PIPELINE_SEMANTIC_ENDPOINT"))
	}

	if model == "" && (provider == semantic.ProviderGemini || strings.Contains(endpoint, "generativelanguage.googleapis.com")) {
		model = semantic.DefaultGeminiModel
	}

	if endpoint == "" {
		if provider == semantic.ProviderGemini {
			gemModel := model
			if gemModel == "" {
				gemModel = semantic.DefaultGeminiModel
			}
			endpoint = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", gemModel)
		} else {
			endpoint = strings.TrimSpace(os.Getenv("JEV_ENDPOINT"))
		}
	}

	apiKey := strings.TrimSpace(opts.SemanticAPIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("PIPELINE_SEMANTIC_API_KEY"))
	}
	if apiKey == "" {
		if provider == semantic.ProviderGemini || strings.Contains(endpoint, "generativelanguage.googleapis.com") {
			apiKey = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
			if apiKey == "" {
				apiKey = strings.TrimSpace(os.Getenv("JEV_API_KEY"))
			}
		} else {
			apiKey = strings.TrimSpace(os.Getenv("JEV_API_KEY"))
			if apiKey == "" {
				apiKey = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
			}
		}
	}

	if apiKey == "" || (provider != semantic.ProviderGemini && endpoint == "") || endpoint == "" {
		step.Status = StatusSkip
		step.Message = "gate semântico ignorado (credenciais ausentes)"
		return step
	}

	_, question, err := semantic.ResolvePreset("yagni")
	if err != nil {
		question = "A mudança proposta respeita o princípio YAGNI e evita complexidade desnecessária?"
	}

	evalState := ""
	if reportPath != "" {
		if raw, err := os.ReadFile(reportPath); err == nil {
			evalState = strings.TrimSpace(string(raw))
		}
	}
	if evalState == "" {
		if deliveryID != "" {
			evalState = "delivery " + deliveryID
		} else {
			evalState = "delivery"
		}
	}

	thinkingLevel := ""
	if provider == semantic.ProviderGemini || strings.Contains(endpoint, "generativelanguage.googleapis.com") {
		thinkingLevel = semantic.DefaultGeminiThinkingLevel
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := semantic.NewClient(semantic.ClientConfig{
		Endpoint:      endpoint,
		APIKey:        apiKey,
		Timeout:       15 * time.Second,
		MaxBodyBytes:  10 * 1024 * 1024,
		Provider:      provider,
		Model:         model,
		ThinkingLevel: thinkingLevel,
	})

	start := time.Now()
	dec, evalErr := client.EvalNoul(ctx, semantic.EvalRequest{
		Mode:      "noul",
		State:     evalState,
		Question:  question,
		Threshold: 0.7,
	})

	var conf float64
	if dec != nil {
		conf = dec.Confidence
	}
	_ = metrics.RecordSemanticEvalWithProvider(absRoot, deliveryID, "noul", provider, model, time.Since(start), evalErr == nil, conf)

	if evalErr != nil {
		secrets := []string{
			apiKey,
			os.Getenv("GEMINI_API_KEY"),
			os.Getenv("PIPELINE_SEMANTIC_API_KEY"),
			os.Getenv("JEV_API_KEY"),
		}
		sanitized := redactSecrets(evalErr.Error(), secrets...)
		step.Status = StatusWarn
		step.Message = fmt.Sprintf("gate semântico indisponível (advisory): %s", sanitized)
		return step
	}

	if dec != nil && dec.Decision {
		step.Status = StatusPass
		step.Message = fmt.Sprintf("gate semântico aprovado (confiança %.2f)", dec.Confidence)
		return step
	}

	reason := "sem justificativa"
	if dec != nil && strings.TrimSpace(dec.Reason) != "" {
		reason = strings.TrimSpace(dec.Reason)
	}
	step.Status = StatusWarn
	step.Message = fmt.Sprintf("gate semântico em alerta (YAGNI): %s", reason)
	return step
}

func redactSecrets(msg string, secrets ...string) string {
	out := msg
	for _, s := range secrets {
		if s != "" {
			out = strings.ReplaceAll(out, s, "[REDACTED]")
		}
	}
	return out
}
