# Project instructions (AGENTS.md format)

This project follows the open **AGENTS.md** format: a single, predictable place for
the context and house rules a coding agent needs. Treat the project's `AGENTS.md`
(Claude Code falls back to it when there is no `CLAUDE.md`) as the source of truth,
and keep it organized into the sections below so any agent — Claude Code, Codex, or
OpenCode — finds what it needs in the same place.

When the corresponding section is empty, fill it in from the repository rather than
guessing; when it conflicts with what the code actually does, trust the code and flag
the drift.

## Dev environment
- The commands to install dependencies, build, and run the project locally.
- How the workspace is laid out (packages, where to make changes, what not to touch).
- Any environment setup an agent must do before its first edit.

## Testing
- The command that runs the full check suite, and how to scope it to one package/test.
- Where the CI plan lives (e.g. `.github/workflows/`) so local runs match CI.
- The expectation: changed code ships with tests, and the suite is green before a commit.

## Conventions
- Language, formatting, and naming rules the code already follows — match the
  surrounding code's idiom rather than importing a personal style.
- Structural rules (where things live, layering, what each module owns).

## PR / commit
- Title and message format.
- The checks (lint, test, typecheck) that must pass before committing.
- Anything required in the description (linked issue, rationale, screenshots).

Nested `AGENTS.md` files deeper in the tree refine these rules for their subtree; the
closest file to the code being edited wins.
