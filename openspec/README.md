# SemLink OpenSpec

SemLink uses OpenSpec change sets as the planning contract for product-boundary,
protocol, mesh, and command-safety changes that are bigger than a local bug fix.
The documents here are human-readable, but they should stay concrete enough for
tickets, tests, demos, and upstream SemStreams or SemOps asks to trace back to
explicit requirements.

## Layout

- `config.yaml`: active OpenSpec 1.5 schema selection, standing project
  context, and artifact rules.
- `changes/<change-id>/proposal.md`: why the change exists, what changes, and
  expected impact.
- `changes/<change-id>/design.md`: design decisions, trade-offs, rollout, and
  open questions.
- `changes/<change-id>/tasks.md`: implementation checklist in dependency order.
- `changes/<change-id>/specs/*/spec.md`: capability requirements and scenarios.

When a change is accepted and implemented, its spec deltas can be promoted into
long-lived baseline specs under `openspec/specs`.
