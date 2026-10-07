# Source and invocation inventory

Codex adaptation at source revision `63abfb1d9f3e3445ec9ea3f99551069a7b251911`.
Legacy files below are product inputs, not foreign-host instructions.

- `artifacts/skills/writing-skills/NOTICE`: SHA-256 `675cf341b117ee867d608845612abea9edfb088aa18c63ae1812b2de52111cef`.
- `artifacts/skills/writing-skills/SKILL.md`: SHA-256 `441e225ba4982e37dc3a1fe3138919797ee94a684a10e0350dafebb42c782959`.
- `artifacts/skills/writing-skills/anthropic-best-practices.md`: SHA-256 `217629b356c09c9bd11017c9788e8fc654ca1b32c92d4a51cd490e16dd65e59a`.
- `artifacts/skills/writing-skills/examples/CLAUDE_MD_TESTING.md`: SHA-256 `0b379a3415e185d3c434b3ad283d8aa132f3022c2a4f210f168865b5986bcef0`.
- `artifacts/skills/writing-skills/graphviz-conventions.dot`: SHA-256 `e2890a593c91370e384b42f2f67b1a6232c9e69dddea7891a0c1c46d7b20b694`.
- `artifacts/skills/writing-skills/patronus.yaml`: SHA-256 `73879bac648e0cec7d91b5fbf7df287c513741fb1f7cd1cec35af73ea188d6aa`.
- `artifacts/skills/writing-skills/persuasion-principles.md`: SHA-256 `a51bc9bf75189ea73a27b3fb504a2fdfdb966fb1f7f1cdf03203230a216ccc03`.
- `artifacts/skills/writing-skills/render-graphs.js`: SHA-256 `ccda971a87bb185f8febf81c56b556a20d026fa980c17b35fa3e8824fbb37852`.
- `artifacts/skills/writing-skills/testing-skills-with-subagents.md`: SHA-256 `c711346852c911b24a84aa161e0cff06a4cd7f4e2fa9e9c0a266cead5afcbade`.

Bodies and sidecars adapt host calls and scope grants. Source placeholders
`{skillDir}` and `{skillsDir}` are expanded only by the installer.
Static route and placement checks do not prove native Codex execution.

## Active reads

- `tdd-cx/SKILL.md`: exact installed Codex companion, verify readable before use; not a stage grant.
- `verification-before-completion-cx/SKILL.md`: exact installed Codex companion, verify readable before use; not a stage grant.

The foreign-host authoring guide and testing/example sidecars were rewritten as
authoring-best-practices.md, testing-skills.md and examples/AGENTS_MD_TESTING.md.
Persuasion and diagram conventions are adapted. render-graphs.js is deliberately
not shipped or invoked: optional rendering needs separately approved tools/outputs.
No editorial skill port or downstream route is supplied.
