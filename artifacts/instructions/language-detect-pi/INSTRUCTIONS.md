# Language relevance (advisory, no hook)

Before language work, inspect only the relevant source files and repository/workspace manifests: go.mod for Go; Cargo.toml for Rust; package.json for JavaScript/TypeScript; pyproject.toml/requirements.txt for Python. Match actual source idioms rather than importing another language's conventions. Do not run dependencies, recursively scan unrelated trees or install missing tools to detect a language.

For Go source authoring/review or Go-specific design, read the installed go-style-uber-pi SKILL.md and needed relative references. That skill is a required delivered dependency; its inclusion does not require loading it for unrelated work. If required guidance is missing, report the prerequisite failure. Other language guidance is optional only when discovered and separately authorized. This is advisory content, not a registered legacy language-detect hook or an automatic loader.
