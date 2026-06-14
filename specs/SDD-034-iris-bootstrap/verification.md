---
tags: [spec, verification, iris, bootstrap]
created: "2026-05-17"
---

# Verification - SDD-034-iris-bootstrap

## Evidence

Map every acceptance criterion from `proposal.md` to concrete proof.

- [ ] Criterion 1 -> commit `<hash>` / test `<name>`
- [ ] Criterion 2 -> commit `<hash>` / test `<name>`

## Test status

- Test suite: `<command> -> <output / coverage %>`
- Manual smoke test: `make smoke-test` (per bootstrap-contract.md § 2) — output captured to `evidence/smoke-test.log`
- No regressions in existing test suite: yes / no

## Decisions made during implementation

-
-

## Promotion candidates

Before archiving, flag what (if anything) should be promoted to the vault.

- [ ] Lesson for `iris/90-lessons.md` (or kubelab vault until iris vault stack is complete)?
- [ ] ADR-worthy decision for `iris/30-architecture/`?
- [ ] New pattern candidate for `00_meta/patterns/`?

## Archive checklist

- [ ] `proposal.md` frontmatter set to `status: archived`
- [ ] Folder moved: `specs/SDD-034-iris-bootstrap/` -> `specs/archive/SDD-034-iris-bootstrap/`
- [ ] Backlog entry in `iris/11-tasks.md` ticked with PR link
- [ ] Promotions above executed (if any)
- [ ] `features.json` all `state: passing` with non-empty `evidence` (per pass-state gating)
