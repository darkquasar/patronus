# Source and provenance inventory

Created at source HEAD `c23d059ef03aec4e831c15d786808f2a619a726f` plus the
accepted uncommitted patch SHA-256
`59de1b8a053e28ec10b3a0cf79fcd878265abc57ae3966c57167b747d4d2a06f`. Hashes
below are SHA-256 of those accepted bytes. Upstream skill name
`dispatching-parallel-agents` (obra/superpowers, commit
`896224c4b1879920ab573417e68fd51d2ccc9072`) is kept here as history.

## Retired Patronus bundles

This skill replaces four bundles, removed with no alias or compatibility copy.
Pi's `dispatching-parallel-agents-pi` is separate and unchanged.

`artifacts/skills/dispatching-parallel-agents/` (targets codex, opencode;
version 2.0.0; vendored upstream body):

- `NOTICE`: `4b926b3ab8c751c98cda94b31479f54de3ac86fa2bb6b7687857877fe96de8e7`
- `SKILL.md`: `f0df13f584049059cc5619f90061405b89dcc6e28ab3f2a8517d27d99c7a46a6`
- `patronus.yaml`: `3a383e71db8e7fcf120075cee30c9ddc95c0e71256a1e6e506be77a8ee4927c5`

`artifacts/skills/dispatching-parallel-agents-cl/` (targets claude; version 2.2.0):

- `NOTICE`: `4b926b3ab8c751c98cda94b31479f54de3ac86fa2bb6b7687857877fe96de8e7`
- `SKILL.md`: `ce46b395fd16f79ef2a867ada139da8cf9ad9734e8c89155a61cb29da68c9e85`
- `patronus.yaml`: `868cf5126d8e74847fc1acb596e04cd6ec0903cc09ba311033c7b9149ddcdfe6`

`artifacts/skills/dispatching-parallel-agents-cx/` (targets codex; version 1.3.0):

- `LICENSE`: `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400`
- `NOTICE`: `e6bed6e6e8819dc08b6c7322d4bafe1a603f91072ccb4f23d7ed388a8f7e065f`
- `SKILL.md`: `f6a4dcbf31747e0f47194ee4319e398481109c65030603ba9eb994911dad8d5e`
- `SOURCE.md`: `267a72523403876f125e9fba3e3fed0eaf32b3bda684e08051ed89f44fe2655e`
- `patronus.yaml`: `645c1ca3252abde09f1f3632065fb4023061e329128925230db1d366a914cbcc`

`artifacts/skills/dispatching-parallel-agents-oc/` (targets opencode; version 1.3.0):

- `LICENSE`: `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400`
- `NOTICE`: `e6bed6e6e8819dc08b6c7322d4bafe1a603f91072ccb4f23d7ed388a8f7e065f`
- `SKILL.md`: `a82baa93c598b22e79ea49f7c74af27dbdb3a40419fcc92f0aaf86d4306cc2a4`
- `SOURCE.md`: `4740802a334465a01c8556868f8faa4778a7374008705d167fe5e3c48f617aeb`
- `patronus.yaml`: `f5f5fc84883a8d6d00d87dc6ca24256c0ca0c628eecbee0f1e140fa91019f3cd`

The retired `-oc` NOTICE described itself as a Codex adaptation; this NOTICE
records the OpenCode, Codex and Claude adaptation correctly.

## Files in this bundle and their prior paths

- `SKILL.md`: adapted, not copied. Common body and host sections built from
  `dispatching-parallel-agents-cx/SKILL.md` lines 8-54,
  `dispatching-parallel-agents-oc/SKILL.md` lines 8-70 and
  `dispatching-parallel-agents-cl/SKILL.md` lines 8-39. The Claude review,
  correction and completion sections (`-cl/SKILL.md` lines 52-83) and the
  upstream example were dropped; they belong to execution and review skills.
- `writer-brief.md`: adapted, made host-neutral, from
  `artifacts/skills/plan-execute-parallel-cl/TEAMMATE-TEMPLATE.md`
  (`7536383e1c55552b83c27b340eddcf10b11885343db9448049cee09b1989407b`),
  which is Patronus-authored under the repository GPL-3.0.
- `LICENSE`: byte copy of `artifacts/skills/dispatching-parallel-agents-cx/LICENSE`
  (`a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400`).
- `LICENSE-GPL-3.0`: byte copy of the repository root `LICENSE`
  (`3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986`), scoped
  to `writer-brief.md` by NOTICE.

## Dependencies and reads

No `requires`. Writer mode is entered only from an execution caller that itself
requires `using-git-worktrees`. Sidecars are named by paths relative to this
directory. Only the entry and manifest-declared files are distributed. No name
substitution, placement or content test proves native loading.
