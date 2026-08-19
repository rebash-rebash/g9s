# G9S UI Redesign Plan

## Goal

Make G9S feel like a real terminal operations tool, inspired by the density and interaction model of K9s rather than a collection of dashboard cards.

## Core UI contract

- One responsive outer application frame for the entire terminal.
- Fixed shell structure: header, content area, status/pagination bar, key-help footer.
- Use dense aligned rows and columns for resource views.
- Avoid hardcoded panel widths/heights; derive dimensions from terminal size.
- Keep backend, GCP services, pricing, analyzer logic, and read-only safety behavior unchanged during UI work.
- Responsive targets: 80x24, 120x40, and larger terminals.
- Consistent status colors: RUNNING/ACTIVE green; warning/underused yellow; TERMINATED/IDLE/UNATTACHED red.
- Consistent keyboard model: arrows navigate, Enter opens details, Esc goes back, `/` filters, `r` refreshes, `?` shows help, `q` quits.

## Screen designs

### 1. Application shell

Create the single outer rounded frame and responsive layout primitives. All screens eventually render inside this shell.

### 2. Dashboard

Replace the resource list/card presentation with a dense resource table containing resource, count, status, and relevant cost/health information.

### 3. Compute Engine

Use a table:

`NAME | ZONE | MACHINE TYPE | STATUS`

Add consistent selection, filtering, pagination/status information, and refresh behavior.

### 4. VM details

Use a compact two-column detail layout inside the same outer frame. Show identity, status, cost, CPU utilization, network I/O, disk I/O, and intelligence/recommendations without excessive vertical whitespace.

### 5. Persistent Disks

Use a table:

`NAME | ZONE | SIZE | TYPE | ATTACHED | COST`

Highlight unattached disks and savings candidates.

### 6. Cost Intelligence

Use dense operational tables rather than four large cards. Show running/stopped VMs, disk counts, monthly compute cost, disk cost where available, potential savings, machine-family cost breakdown, and top findings.

### 7. GKE

Follow the same shell and table conventions for clusters, nodes, workloads, utilization, and cost.

### 8. Cross-screen navigation

Make keyboard behavior consistent across all resources. Support resource-oriented navigation and later the planned `:`, `d`, `u`, `c`, `w`, and `r` shortcuts without changing the read-only safety model.

## Implementation order

- [x] Plan captured in repository
- [x] Step 1 — application frame and responsive shell
- [x] Step 2 — dashboard table
- [ ] Step 3 — Compute Engine table (in progress: dense aligned rows and responsive table sizing)
- [ ] Step 4 — VM detail layout
- [ ] Step 5 — Persistent Disk table
- [ ] Step 6 — Cost Intelligence table/layout
- [ ] Step 7 — GKE UI using the same primitives
- [ ] Step 8 — consistent keyboard/navigation model
- [ ] Step 9 — responsive polish and terminal-size testing

## Rules for each step

1. Implement one step at a time.
2. Do not mix unrelated backend changes into UI work.
3. Run `gofmt` on changed Go files.
4. Run `go test ./...` after each step.
5. Manually run `go run ./cmd/g9s` and inspect the terminal UI before moving to the next step.
6. Fix regressions before starting the next step.
