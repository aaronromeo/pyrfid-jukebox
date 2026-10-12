# SoundSprout Web Portal — Implementation Roadmap

This is the umbrella plan for the SoundSprout web portal. The full design lives in
`docs/superpowers/specs/2026-10-12-web-portal-design.md`. The work is split into three
plans, each of which produces working, testable software on its own and can be shipped
independently.

Execute them **in order** — each builds on the artifacts of the previous one.

| # | Plan | Goal | Depends on |
|---|------|------|------------|
| 1 | [Framework](./2026-10-12-web-portal-1-framework.md) | `soundsprout-server serve` launches a Hello World HTTP portal (supervised, documented). | — |
| 2 | [Card manager](./2026-10-12-web-portal-2-cards.md) | Manage `server/rfid_map.json` from the web UI; harden the Python map writer. | Plan 1 |
| 3 | [Bluetooth](./2026-10-12-web-portal-3-bluetooth.md) | Pair/switch/reconnect Bluetooth audio from the web UI. | Plans 1, 2 |

## Why this split

- Each plan is a vertical slice ending in something a reviewer can accept or reject on
  its own: Plan 1 ships a running (if empty) portal; Plan 2 adds card management; Plan 3
  adds Bluetooth.
- Plan 1 establishes the framework (`portal` package, `serve` subcommand, supervisor
  entry, docs) so Plans 2 and 3 only add features.
- The Python map-write hardening (spec §8) rides with Plan 2 because it is the same data
  (`rfid_map.json`) the card manager writes; the two writers must honor one lock contract.

## Conventions shared by all three plans

- Go module: `aaronromeo.com/soundsprout`; Go floor 1.21 (CI) — no 1.22 `ServeMux`
  method-pattern routing.
- Run Go tests locally with `ASDF_GOLANG_VERSION=1.25.6` (repo pins 1.23.1, not installed).
- **Go tests must live in external `_test` packages** (e.g. `package portal_test`) — the
  `testpackage` linter is enabled in `system/soundsprout/.golangci.yml`.
- No shell: `exec.Command(name, args...)` / `exec.CommandContext` only.
- Python: black line-length 79, flake8 max-line-length 80.

## Final reconciliation

After Plan 3, all 15 sections of the spec are implemented. The spec's §15 validation
checklist is resolved by Plan 1 (serve), Plan 2 (Python + map), and Plan 3 (Bluetooth).
