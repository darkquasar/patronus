# Language relevance in Codex (advisory, no hook)

Before language-specific work, inspect only relevant source and repository
manifests: go.mod for Go, Cargo.toml for Rust, package.json for JavaScript or
TypeScript, and pyproject.toml/requirements.txt for Python. Match current source
idioms and approved project commands rather than importing another language's
conventions. Do not install packages, execute dependencies or recursively scan
unrelated trees to identify a language.

Read applicable language guidance only when present, host-compatible and within
the task grant. Missing required guidance blocks that work; optional guidance is
not silently installed or replaced. Language detection grants no test execution,
network access, editing or broader ownership. Report the observed language and
any unavailable approved checks.

This is advisory guidance, not a SessionStart hook or automatic skill loader.
