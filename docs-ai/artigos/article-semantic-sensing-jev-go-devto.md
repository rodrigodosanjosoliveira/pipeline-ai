---
title: "System 1 vs. System 2 in AI Engineering: Replacing Ambiguous LLM Judges with Calibrated Semantic Sensors (Jev) in Pure Go"
published: false
description: How we built a zero-dependency System 1 semantic sensor (EvalNoul, EvalChoice, EvalScore) in Go 1.24 with anti-hallucination guards, native fuzzing, MCP integration, and DORA telemetry.
tags: go, ai, softwareengineering, devops
---

In AI-assisted software engineering pipelines, there is a clear boundary between what compilers can verify and what requires semantic understanding.

For everything syntactic and structural — formatting (`gofmt`), static analysis (`go vet`), race-condition testing (`go test -race`), and vulnerability scanning (`govulncheck`) — deterministic gates are unbeatable. Either the code compiles and passes tests, or the gate halts the pipeline with `exit code 1`.

But what happens when a pipeline gate depends on **semantics**?
- *"Does this diff touch authentication logic or cryptographic surface area?"*
- *"Which specialized skill should handle this SQL migration failure: `data-model`, `migration-reviewer`, or `dev-implementation`?"*
- *"On a scale from 0 to 10, how well does this handoff report satisfy our architectural invariants?"*

For the past two years, the industry's default answer has been **"LLM-as-a-Judge"**: sending a massive prompt to a slow, generative reasoning model (System 2) and hoping it outputs "Yes" or picks a valid category in free text.

In production pipelines, traditional "LLM-as-a-Judge" suffers from four chronic pathologies: **high latency, stochastic verbosity, uncalibrated confidence, and category hallucination**.

In this article — consolidating deliveries **DEL-025, DEL-026, DEL-029, and DEL-030 (v0.10.0) of the Pipeline-AI framework** —, I share how we applied Daniel Kahneman's cognitive theory (*System 1 vs. System 2*) to autonomous agent engineering, building a **calibrated semantic sensing subsystem (Jev) in pure Go 1.24 (`internal/sensor/semantic`)** with zero external dependencies, three typed decision primitives (`EvalNoul`, `EvalChoice`, `EvalScore`), fail-closed security hardening, native fuzz testing, and DORA-integrated telemetry.

---

## 1. Why Traditional "LLM-as-a-Judge" (System 2) Fails in Autonomous Pipelines

In *Thinking, Fast and Slow*, Daniel Kahneman distinguishes between two cognitive modes:
- **System 2 (Slow, Deliberative, Generative):** Analyzes complex problems step-by-step, writes code, designs architectures, and authors documentation. This is where frontier generative LLMs shine.
- **System 1 (Fast, Classificatory, Calibrated):** Operates as a low-latency reflexive sensor that classifies patterns, routes signals, and emits calibrated confidence scores.

When we force a full generative LLM (System 2) to act as a pipeline sensor, we hit predictable failure modes:

| Dimension | Traditional LLM-as-a-Judge (System 2) | Calibrated Jev Semantic Sensor (System 1 in Pipeline-AI) |
|---|---|---|
| **Output Nature** | Free-form generative prose (*"Looking at the diff, it seems..."*). | Strictly typed structs (`NoulDecision`, `ChoiceDecision`, `ScoreDecision`). |
| **Confidence Calibration** | Uncalibrated verbal certainty (hallucinates "99% sure" in text). | Calibrated confidence (`Confidence` in `[0.0, 1.0]`) enforced against `Threshold`. |
| **Routing Hallucination** | Invents a 4th option when asked to choose among 3 valid skills. | **`SEC-01` Anti-Hallucination Guard**: rejects with `ErrInvalidChoice` any option outside the allowed set. |
| **Continuous Scoring** | Emits out-of-scale numbers without boundary guarantees. | **`SEC-02` Dual Guard**: validates `[0.0, MaxScore]`, rejects `NaN`/`Inf`, and requires both `Confidence >= Threshold` and `(Score/MaxScore) >= Threshold`. |
| **Failure Mode** | Guesses or proceeds silently (*fail-open*). | Strictly **Fail-Closed**: blocks the transition if confidence falls below threshold or transport fails. |

---

## 2. Three Semantic Decision Primitives in Pure Go (`internal/sensor/semantic`)

Upholding **Pipeline-AI**'s core invariant — **100% Go 1.24 standard library, zero external dependencies in `go.mod`** —, we built `internal/sensor/semantic` around three primitives:

### 2.1. `EvalNoul`: Calibrated Binary Decisions (`/v1/noul`)

`EvalNoul` evaluates a binary hypothesis over a state (`git diff`, source file, or handoff contract) and returns a `NoulDecision`:

```go
// internal/sensor/semantic/types.go
type NoulDecision struct {
    Decision   bool    `json:"decision"`
    Confidence float64 `json:"confidence"`
    Reason     string  `json:"reason,omitempty"`
}
```

Even if the model returns `Decision: true`, if `Confidence` is `0.68` and the gate requires `Threshold: 0.80`, `EvalNoul` returns `ErrInsufficientConfidence`. Statistical uncertainty is treated as a safe rejection.

### 2.2. `EvalChoice`: Multi-Choice Routing with Anti-Hallucination Guard (`/v1/choice`)

When an orchestrator needs to classify an incident or route a task to one of a finite set of skills, `EvalChoice` enforces **`SEC-01` (Anti-Hallucination)**: it requires at least 2 non-empty choices and verifies that the returned choice strictly belongs to the allowed input slice:

```go
// internal/sensor/semantic/client.go
func (c *Client) EvalChoice(ctx context.Context, input ChoiceInput) (ChoiceDecision, error) {
    // Validates state, question, and at least 2 non-empty choices...
    data, err := c.doPost(ctx, "/v1/choice", payload)
    if err != nil {
        return ChoiceDecision{}, err
    }

    dec, err := ParseChoiceResponse(data)
    if err != nil {
        return ChoiceDecision{}, err
    }

    // SEC-01: Strictly verify that dec.Choice belongs to input.Choices
    validChoice := false
    for _, ch := range input.Choices {
        if dec.Choice == ch {
            validChoice = true
            break
        }
    }
    if !validChoice {
        return dec, fmt.Errorf("%w: escolha %q não pertence às opções permitidas", ErrInvalidChoice, dec.Choice)
    }

    if dec.Confidence < threshold {
        return dec, fmt.Errorf("%w: confiança %.2f abaixo do threshold %.2f",
            ErrInsufficientConfidence, dec.Confidence, threshold)
    }

    return dec, nil
}
```

### 2.3. `EvalScore`: Continuous Scoring with Dual Calibration Guard (`/v1/score`)

For continuous quality evaluation on a `0..MaxScore` scale, `EvalScore` implements **`SEC-02`**:
1. **Numeric Boundary Enforcement:** Rejects `NaN`, `+Inf`, `-Inf`, negative numbers, or `Score > MaxScore` with `ErrInvalidScore`.
2. **Dual Approval Condition:** Requires **both** `Confidence >= Threshold` **and** normalized score `(Score / MaxScore) >= Threshold`:

```go
// internal/sensor/semantic/client.go
if dec.Score > maxScore {
    return dec, fmt.Errorf("%w: score %.2f excede max_score %.2f", ErrInvalidScore, dec.Score, maxScore)
}

if dec.Confidence < threshold {
    return dec, fmt.Errorf("%w: confiança %.2f abaixo do threshold %.2f",
        ErrInsufficientConfidence, dec.Confidence, threshold)
}

normalizedScore := dec.Score / maxScore
if normalizedScore < threshold {
    return dec, fmt.Errorf("%w: score normalizado %.2f (%.2f/%.2f) abaixo do threshold %.2f",
        ErrInsufficientConfidence, normalizedScore, dec.Score, maxScore, threshold)
}
```

---

## 3. Fail-Closed Security Hardening (`SEC-01`, `SEC-02`, `SEC-03`)

All HTTP communication in `internal/sensor/semantic` flows through the private `Client.doPost` helper, enforcing three mandatory security controls:

1. **Automatic Credential Redaction (`[REDACTED]`):** Any transport, DNS, or TLS error is scrubbed via `c.sanitizeError(err)`, replacing any occurrence of `APIKey` (or `PIPELINE_AI_SEMANTIC_API_KEY`) with `[REDACTED]` before surfacing in logs, CLI stderr, or MCP responses.
2. **DoS & Memory Exhaustion Protection (`io.LimitReader`):**
   - Input `state` in the CLI and MCP tool is capped at **512KB** (524,288 bytes) prior to any network call.
   - HTTP responses are bounded by `io.LimitReader(resp.Body, maxBytes+1)` (default **10MB**), returning `ErrResponseTooLarge` on oversized payloads.
3. **Mandatory Context Timeouts:** Every request runs under `context.WithTimeout` (10s default), preventing hung agent processes.

---

## 4. Native Go Fuzzing: 830,000+ Executions with Zero Panics

To guarantee that malformed or adversarial JSON payloads from external inference endpoints never trigger runtime panics, we wrote native Go fuzz targets (`FuzzParseNoulResponse`, `FuzzParseChoiceResponse`, `FuzzParseScoreResponse`):

```go
// internal/sensor/semantic/fuzz_test.go
func FuzzParseScoreResponse(f *testing.F) {
    f.Add([]byte(`{"score":0.85,"max_score":1.0,"confidence":0.92,"reason":"ok"}`))
    f.Add([]byte(`{"score":-1.0,"max_score":10.0,"confidence":0.8}`))
    f.Add([]byte(`{"score":15.0,"max_score":10.0,"confidence":0.9}`))
    f.Add([]byte(`{}`))
    f.Add([]byte(`not json`))

    f.Fuzz(func(t *testing.T, data []byte) {
        dec, err := ParseScoreResponse(data)
        if err == nil {
            if dec.Confidence < 0.0 || dec.Confidence > 1.0 {
                t.Errorf("valid parse produced out-of-range confidence: %v", dec.Confidence)
            }
            if dec.Score < 0.0 || dec.MaxScore <= 0.0 || dec.Score > dec.MaxScore {
                t.Errorf("valid parse produced invalid score bounds: score=%v max=%v", dec.Score, dec.MaxScore)
            }
        }
    })
}
```

Across **830,000+ randomized executions** (`421,119` on `FuzzParseChoiceResponse` and `410,163` on `FuzzParseScoreResponse`), the parsers achieved **100% stability with zero crashes**.

---

## 5. Dual Ergonomics (CLI + MCP) and Semantic DORA Telemetry

Both human engineers in the terminal and autonomous agents over **Model Context Protocol (MCP)** can invoke all three semantic modes seamlessly:

```bash
# 1. Binary Mode (default -mode noul):
$ pipeline-ai eval -state internal/auth/token.go -question "are tokens compared in constant time?" -threshold 0.85

# 2. Multi-Choice Routing (-mode choice):
$ pipeline-ai eval -mode choice -state "SQL lock timeout on migration" \
    -question "which skill should review this?" \
    -choices "migration-reviewer,query-performance,code-reviewer"

# 3. Continuous Scoring (-mode score):
$ pipeline-ai eval -mode score -state docs-ai/deliveries/DEL-030/report.md \
    -question "completeness of test evidence" -max-score 10 -threshold 0.8 -json
```

Every evaluation automatically records a `SemanticEvalEvent` in `.pipeline-state/metrics.json`, surfacing alongside the 4 DORA metrics in `pipeline-ai metrics summary`:

```bash
$ pipeline-ai metrics summary
=== Métricas DORA & Telemetria (Últimos 30 dias) ===
...
Sensor Semântico (Jev):
  Total de Avaliações:           42
  Taxa de Aprovação:             88.1%
  Confiança Média:               91.4%
  Por Modo:                      noul=24, choice=10, score=8
```

---

## Conclusion: Full Cognitive Orchestration

Mature AI Engineering isn't about throwing the largest generative model at every check — it's about **matching the right cognitive architecture to each decision**:

1. **Deterministic Toolchain Gates (`gofmt`, `go vet`, `go test -race`, `pipeline-validate`):** Binary truth for syntax, compilation, and static invariants.
2. **System 1 Calibrated Semantic Sensors (`Jev` — `EvalNoul`, `EvalChoice`, `EvalScore`):** Fast, typed, threshold-calibrated, hallucination-proof classification and scoring.
3. **System 2 Specialized Agents (`architecture-guardrails`, `test-writer`, `dev-implementation`, `security-review`):** Deep reasoning reserved for architectural design and code synthesis.

---

The complete **Pipeline-AI (v0.10.0)** source code is 100% open-source under the MIT license:

👉 **GitHub Repository:** [github.com/rodrigodosanjosoliveira/pipeline-ai](https://github.com/rodrigodosanjosoliveira/pipeline-ai)

How are you handling semantic routing and quality gates in your AI agent workflows? Let's discuss in the comments! 💬🧠🚀
