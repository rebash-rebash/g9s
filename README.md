# G9S

**G9S — GCP Resource & Cost Explorer**

A read-only, keyboard-driven terminal UI for exploring Google Cloud resources, utilization, waste, and cost.

G9S is inspired by the interaction model of K9s, but its focus is broader: **resource discovery + utilization intelligence + cost visibility + optimization recommendations**.

## Product vision

G9S should answer four questions quickly:

1. **What exists?** — browse GCP resources.
2. **What is being used?** — inspect utilization and activity.
3. **What is costing money?** — understand spend and trends.
4. **What is wasteful?** — identify idle, unused, or oversized resources and estimate savings.

## Initial scope

The first release is intentionally read-only and focused on a small set of high-value services:

- Projects
- Compute Engine
- GKE
- Persistent disks
- Cloud SQL
- Cloud Storage
- VPC / networking basics
- Billing visibility
- Cloud Monitoring metrics
- Unused / underutilized resource detection

## Architecture

```text
                    Google Cloud
                         │
        ┌────────────────┼────────────────┐
        │                │                │
   Resource APIs      Monitoring        Billing
        │                │                │
        └────────────────┼────────────────┘
                         │
                    GCP Service Layer
                         │
                       Models
                         │
              ┌──────────┴──────────┐
              │                     │
          Analyzers                Cache
              │
     ┌────────┼─────────┐
     │        │         │
   Usage    Waste      Cost
     │        │         │
     └────────┼─────────┘
              │
        Recommendations
              │
             TUI
```

The UI must not call Google Cloud APIs directly. GCP clients are isolated behind the service layer so the analyzers and UI remain testable.

## Repository structure

```text
g9s/
├── cmd/
│   └── g9s/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── gcp/
│   │   ├── client.go
│   │   ├── projects.go
│   │   ├── compute.go
│   │   ├── gke.go
│   │   ├── disks.go
│   │   ├── sql.go
│   │   ├── storage.go
│   │   ├── monitoring.go
│   │   ├── billing.go
│   │   └── assets.go
│   ├── model/
│   │   ├── project.go
│   │   ├── vm.go
│   │   ├── cluster.go
│   │   ├── disk.go
│   │   ├── sql.go
│   │   ├── bucket.go
│   │   ├── metric.go
│   │   ├── cost.go
│   │   └── finding.go
│   ├── analyzer/
│   │   ├── utilization.go
│   │   ├── waste.go
│   │   ├── cost.go
│   │   └── recommendations.go
│   ├── cache/
│   │   └── cache.go
│   └── ui/
│       ├── app.go
│       ├── dashboard.go
│       ├── projects.go
│       ├── compute.go
│       ├── gke.go
│       ├── cost.go
│       ├── waste.go
│       └── styles.go
├── tests/
│   └── README.md
├── .github/
│   └── workflows/
│       └── ci.yml
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

## Design principles

### Read-only first

The first versions must not create, update, stop, start, resize, delete, or otherwise mutate GCP resources.

### API/UI separation

Google Cloud API calls belong in `internal/gcp`. TUI state and rendering belong in `internal/ui`.

### Analyzer-driven intelligence

Usage, waste, and cost logic belongs in `internal/analyzer`, independent of presentation.

### Testability

Service interfaces, models, and analyzers should be unit-testable without requiring a live GCP project.

### Safe defaults

No destructive commands should be introduced without an explicit product/security review.

## Development roadmap

- [ ] Phase 0 — repository and architecture
- [ ] Phase 1 — Google authentication / ADC
- [ ] Phase 2 — project discovery and selection
- [ ] Phase 3 — dashboard
- [ ] Phase 4 — Compute Engine browser
- [ ] Phase 5 — resource detail views
- [ ] Phase 6 — Cloud Monitoring metrics
- [ ] Phase 7 — utilization classification
- [ ] Phase 8 — unused / waste detection
- [ ] Phase 9 — billing and cost views
- [ ] Phase 10 — GKE explorer and utilization
- [ ] Phase 11 — recommendations
- [ ] Phase 12 — additional GCP services

## Planned keyboard model

The interaction model should feel familiar to K9s users:

- `↑` / `↓` — navigate
- `Enter` — open resource
- `Esc` — go back
- `/` — search / filter
- `:` — resource command / context navigation
- `d` — details
- `u` — utilization
- `c` — cost
- `w` — waste findings
- `r` — recommendations
- `q` — quit

## Local development

Prerequisites:

- Go
- Google Cloud CLI
- Application Default Credentials

Example authentication setup:

```bash
gcloud auth application-default login
```

The CLI will eventually support explicit project/context selection while continuing to use standard Google authentication mechanisms.

## License

TBD.
