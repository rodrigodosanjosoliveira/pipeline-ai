---
title: DORA Metrics for AI Agents: How We Built an Observability Engine in Pure Go
published: false
description: How to adapt the 4 DORA metrics (Deployment Frequency, Lead Time, Failure Rate, MTTR) for autonomous multi-agent coding pipelines using Go 1.24 and MCP.
tags: go, ai, devops, softwareengineering
---

The first phase of AI coding was dominated by awe at token generation speed: how many lines of code could Claude, Devin, or Antigravity produce in 60 seconds?

However, once autonomous coding agents enter production-grade engineering workflows, the fundamental question asked by Tech Leads and CTOs shifts dramatically:

> **"Are AI agents genuinely accelerating software delivery, or are they simply amplifying technical debt, rework rates, and pull-request chaos?"**

To answer this question quantitatively rather than intuitively, we don't need to invent new software metrics. We need to leverage the gold standard of modern engineering: **DORA Metrics (DevOps Research and Assessment)**.

In this article — celebrating release **DEL-020 of the Pipeline-AI framework** —, I share how we redesigned and implemented the four DORA metrics specifically for autonomous AI agents, building a **zero-dependency telemetry engine in pure Go 1.24 stdlib**, hardened with **fuzz testing**, and exposed directly to agents via the **Model Context Protocol (MCP)**.

---

## 1. Why DORA Metrics for AI Agents?

Developed by Google Cloud / DORA, the four key metrics balance two essential software delivery forces: **Throughput** and **Stability**.

When human keystrokes are replaced by a team of specialized AI agents (Triage, Architecture, Test Writer, Dev Implementation, Code Review, Security, and QA), the four DORA metrics take on an entirely new meaning at machine speed:

| DORA Metric | Human Engineering Meaning | Adaptation for Autonomous Agents (Pipeline-AI) |
|---|---|---|
| **Deployment Frequency (DF)** | How often code reaches production (weekly or monthly). | Number of fully validated micro-deliveries shipped per day at machine speed. |
| **Lead Time for Changes (LT)** | Time from first commit to production deployment (days/weeks). | Time elapsed from initial issue triage (`pm-triage`) to validated PR creation in `git-delivery` (measured in **minutes**). |
| **Change Failure Rate (CFR)** | Percentage of deployments causing a production failure. | Percentage of pipeline runs blocked by deterministic AI quality gates (*doctor*, *sensor*, *validate*). |
| **Time to Restore Service (MTTR)** | Mean time to recover from a downtime or production defect. | Time required for the agent pipeline to resolve a blocked gate (`STATUS: bloqueado`) and restore a passing state (`ok`). |

If an autonomous agent churns out 5,000 lines of code per hour but exhibits a **60% Change Failure Rate** on test and security gates, it is not delivering velocity: it is creating cognitive pollution for human reviewers.

---

## 2. Telemetry Architecture in Pure Go (`internal/metrics`)

In **Pipeline-AI**, we maintain a strict invariant: **Zero external dependencies**. Introducing third-party telemetry collectors or SaaS vendor SDKs would increase supply chain risk and compromise air-gapped or containerized execution.

We implemented `internal/metrics` exclusively with the Go 1.24 standard library:

```go
// internal/metrics/metrics.go
type DORAMetrics struct {
    DeploymentFrequency float64       `json:"deployment_frequency"` // Deploys per day
    DeploymentsTotal    int           `json:"deployments_total"`
    LeadTimeForChanges  time.Duration `json:"lead_time_for_changes"` // Average lead time
    ChangeFailureRate   float64       `json:"change_failure_rate"`   // Percentage (0.0 to 100.0)
    FailedDeployments   int           `json:"failed_deployments"`
    TimeToRestore       time.Duration `json:"time_to_restore"` // Mean time to recovery
    RestoresTotal       int           `json:"restores_total"`
}
```

### Atomic & Fail-Safe Persistence

Agent telemetry must be fault-tolerant: if a process aborts midway through a write, metrics files must never become corrupt.

We enforce atomic file writes using temporary files and `os.Rename`, synchronized via `sync.Mutex`:

```go
func Save(rootDir string, data *MetricsData) error {
    mu.Lock()
    defer mu.Unlock()

    stateDir := filepath.Join(rootDir, DirName)
    _ = os.MkdirAll(stateDir, 0755)

    tmpFile := filepath.Join(stateDir, fmt.Sprintf("%s.tmp.%d", FileName, time.Now().UnixNano()))
    if err := os.WriteFile(tmpFile, payload, 0644); err != nil {
        return err
    }
    // Atomic filesystem swap
    return os.Rename(tmpFile, targetFile)
}
```

If the metrics database is missing or empty, `Load` gracefully initializes default state structures. **Telemetry must never fail the build.**

---

## 3. Hardened with Native Fuzzing: 1.1M+ Executions

In LLM workflows, structured outputs can be interrupted, truncated, or injected with malformed tokens. To ensure our JSON parser never panics or triggers denial-of-service vulnerabilities, we built a native **Go Fuzz Target**:

```go
// internal/metrics/metrics_fuzz_test.go
func FuzzParseMetrics(f *testing.F) {
    f.Add([]byte(`{"version":1,"events":[],"stages":[],"deliveries":[]}`))
    f.Add([]byte(`{"version":1,"deliveries":[{"delivery_id":"DEL-001","success":true,"lead_time":600000000000}]}`))
    f.Add([]byte(``))
    f.Add([]byte(`null`))
    f.Add([]byte(`{"corrupted": true`))

    f.Fuzz(func(t *testing.T, data []byte) {
        md, err := ParseMetrics(data)
        if err != nil {
            return // Handled errors are expected for corrupt input
        }
        if md == nil {
            t.Fatalf("ParseMetrics returned nil without error on valid input")
        }
    })
}
```

We verified the parser across **1,132,000+ continuous executions** (`go test -fuzz=FuzzParseMetrics -fuzztime=10s`), confirming zero panics or unhandled edge cases.

---

## 4. Transparent Pipeline Runner Instrumentation

Rather than relying on agents to self-report when they begin or complete tasks, we instrumented the **Pipeline Runner (`internal/runner`)** at the engine level:

```go
// Excerpt from internal/runner/runner.go
docStart := time.Now()
docResults, docOK := doctor.Collect(absRoot)
_ = metrics.RecordStageDuration(absRoot, deliveryID, "doctor", time.Since(docStart), docOK)

sensorStart := time.Now()
sensorRes, err := sensor.Run(absRoot)
_ = metrics.RecordStageDuration(absRoot, deliveryID, "sensor", time.Since(sensorStart), sensorStep.Status == StatusPass)

valStart := time.Now()
valStep := validate.Run(...)
_ = metrics.RecordStageDuration(absRoot, deliveryID, "validate", time.Since(valStart), validateStep.Status == StatusPass)

// Record overarching gate outcome
_ = metrics.RecordEvent(absRoot, metrics.Event{
    Type:       metrics.EventGate,
    DeliveryID: deliveryID,
    Timestamp:  time.Now().UTC(),
    Duration:   time.Since(runStart),
    Success:    allOK,
    Stage:      "runner",
})
```

This delivers **zero-overhead passive observability**: agents execute their assigned skills while telemetry data is persisted in real time.

---

## 5. CLI & Model Context Protocol (MCP) Integration

Telemetry data is immediately queryable via both terminal CLI and AI agents over **MCP**:

### Terminal CLI:
```bash
$ pipeline-ai metrics summary
Métricas DORA do Pipeline:
  Deployment Frequency:    3.50 deploys/dia (total: 20)
  Lead Time for Changes:   8m 42s
  Change Failure Rate:     5.0% (falhas: 1)
  Time to Restore Service: 4m 15s (restaurações: 1)
```

Structured JSON for Grafana, Datadog, or GitHub Actions:
```bash
$ pipeline-ai metrics summary -json
{
  "deployment_frequency": 3.5,
  "deployments_total": 20,
  "lead_time_for_changes": 522000000000,
  "change_failure_rate": 5.0,
  "failed_deployments": 1,
  "time_to_restore": 255000000000,
  "restores_total": 1
}
```

### Native MCP Tool (`pipeline_get_metrics`):
AI agents invoke the MCP tool directly:
```json
{
  "name": "pipeline_get_metrics",
  "arguments": {
    "dir": "."
  }
}
```
If an agent detects its `Change Failure Rate` climbing, it can autonomously adapt: allocating more reasoning tokens to test generation (`test-writer`) or requesting additional guardrail checks.

---

## The Takeaway: Agent Reliability Engineering (ARE)

Generating code without telemetry is like driving a supercar with a disabled dashboard: you might feel fast, but you won't know the engine is overheating until it blows up.

By implementing **DORA Metrics for AI Agent Pipelines**, engineering teams achieve:
1. **Bottleneck Visibility:** Concrete latency data per skill (TDD Red vs. Dev Green vs. Code Review).
2. **Defeating the Speed Mirage:** Fast code that fails gates is immediately surfaced in the CFR metric.
3. **Auditable Industrial AI:** Shifting from casual AI prototyping to hardened, measurable **Agent Reliability Engineering (ARE)**.

---

The **Pipeline-AI** framework, telemetry engine, and complete 16-skill catalog are open-source under the MIT license:

👉 **GitHub:** [github.com/rodrigodosanjosoliveira/pipeline-ai](https://github.com/rodrigodosanjosoliveira/pipeline-ai)

How is your engineering team measuring the real-world impact and reliability of AI agents? Let's discuss in the comments! 💬📊🚀
