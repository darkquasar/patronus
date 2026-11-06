# Source and invocation inventory

Shared worktree skill, version 3.0.0, for Claude, Codex and OpenCode. It
replaces the 2.0.0 unsuffixed bundle and the retired `using-git-worktrees-cl`,
`using-git-worktrees-cx` and `using-git-worktrees-oc` bundles, and becomes the
single home of the allocation-only script. Prepared at source revision
`c23d059ef03aec4e831c15d786808f2a619a726f` plus the accepted uncommitted
baseline (tracked patch SHA-256
`59de1b8a053e28ec10b3a0cf79fcd878265abc57ae3966c57167b747d4d2a06f`). Historical
paths below are provenance, not invocations.

## SKILL.md (MIT, adapted)

The body adapts the corrected Codex contract of
`artifacts/skills/using-git-worktrees-cx/SKILL.md` (lines 8-67), itself an
adaptation of obra/superpowers `skills/using-git-worktrees/SKILL.md` at
`896224c4b1879920ab573417e68fd51d2ccc9072` (Copyright (c) 2025 Jesse Vincent,
MIT). Changes against that contract: prepared baselines and capture against the
prepared baseline; host notes for Claude, Codex and OpenCode; the allocator is
named by a path relative to this skill's directory; companion routes name each
host's skills without installer placeholders, so the rendered bytes are the
same for every host. The legacy upstream behaviors of the 2.0.0 and `-cl`
bodies (automatic ignore-file edit and commit, in-place fallback, automatic
dependency setup) are dropped. `LICENSE` is a byte copy of
`using-git-worktrees-cx/LICENSE`.

Prior inputs (SHA-256):

| Prior path | SHA-256 |
|------------|---------|
| `artifacts/skills/using-git-worktrees/SKILL.md` (2.0.0) | `e2c3ec142e52868a51af246c620cd76ab648dcf27d6900d47e6ffd07159a9794` |
| `artifacts/skills/using-git-worktrees/NOTICE` (2.0.0) | `fccd606cffc811f31881f53faea0a82b1118361558c98d61dfdde8b7e259c7a6` |
| `artifacts/skills/using-git-worktrees/patronus.yaml` (2.0.0) | `09acb98d42b5019599ee6404c0ea4bc2b5df4d5b207771a3770df2b18b681057` |
| `artifacts/skills/using-git-worktrees-cl/SKILL.md` | `fb7f35b3173c5b3beccc50fe573bd965bfbcdadf12a55a1dc6d54fa9abc7b529` |
| `artifacts/skills/using-git-worktrees-cl/NOTICE` | `fccd606cffc811f31881f53faea0a82b1118361558c98d61dfdde8b7e259c7a6` |
| `artifacts/skills/using-git-worktrees-cl/patronus.yaml` | `c5e78e444928a931fcb17246a70cca6d15ae66dbc45f54c900128e0d590f5eb8` |
| `artifacts/skills/using-git-worktrees-cx/SKILL.md` | `feccae384837e9b20bca7924d7f1ea7af7093c61dc937839a1debafbd6522762` |
| `artifacts/skills/using-git-worktrees-cx/LICENSE` | `a37e0e9697144819e1d965176ac4ae5bc3fa02d11e7812036bbcadf6dafe2400` |
| `artifacts/skills/using-git-worktrees-cx/NOTICE` | `bc44969cd46f0331359bb0ecb68bd6b26e7acfb11437c832dd2e1207ad4e6c7a` |
| `artifacts/skills/using-git-worktrees-cx/SOURCE.md` | `b2c77881767b5425213a2384414e6bc34eeade33e6023fb64379b282708dab80` |
| `artifacts/skills/using-git-worktrees-cx/patronus.yaml` | `3beae654614d02f6a9b646ff8cb9d72ccb50bf4a2b68369258146c8dc25f1863` |
| `artifacts/skills/using-git-worktrees-oc/SKILL.md` | `60757aa5ed274987e6ed25fa9e205466f527fd561c614ce8597be3addaa7db75` |
| `artifacts/skills/using-git-worktrees-oc/SOURCE.md` | `1226a38ef9a744e6d7bc227200b8f93bf702d4163cf6a6d5f542c428f7adb500` |
| `artifacts/skills/using-git-worktrees-oc/patronus.yaml` | `5e0ae803948198e719289126e4a2e63cfb4a1e7bc686756a277e72c61d466c73` |

## Relocated allocator files (GPL-3.0, Patronus-authored)

Moved from `artifacts/skills/plan-execute-parallel-cx`, whose recorded source
is `artifacts/skills/plan-execute-parallel` at
`46247c13c3de38008c59995c5b1b697f78d3f9e5` (Patronus contributors, GPL-3.0).
`LICENSE-GPL-3.0` is a byte copy of `plan-execute-parallel-cx/LICENSE`. The
OpenCode mirror `plan-execute-parallel-oc/scripts/prepare-worktrees.py` had the
same bytes as the script; one copy remains, so the mirror-parity test is
retired with the mirror.

| New path | Prior path | Prior SHA-256 | Change |
|----------|------------|---------------|--------|
| `scripts/prepare-worktrees.py` | `plan-execute-parallel-cx/scripts/prepare-worktrees.py` | `678e3e59c740a24421a3afd184d3a896cf0db4882d4c6b94c824d495031def1a` | none, bytes unchanged |
| `references/worktree-allocation.md` | `plan-execute-parallel-cx/references/worktree-allocation.md` | `a81a73def38a508bedff9b0416b6ffa760785283180f3885a4a6834171cc704b` | invocation names `scripts/prepare-worktrees.py` relative to this skill's directory |
| `fixtures/test_prepare_worktrees.py` | `plan-execute-parallel-cx/fixtures/test_prepare_worktrees.py` | `505aca06ee34321cf64625207506b7b85ce710fd1d6e2bf1a229e765d4ed8898` | single-copy suite: OpenCode mirror path, its subclass and the byte-parity test removed; `CX_SCRIPT` renamed `SCRIPT`, class renamed `PrepareWorktrees` |
| `LICENSE-GPL-3.0` | `plan-execute-parallel-cx/LICENSE` | `3972dc9744f6499f0f9b2dbf76696f2ae7ad8af9b23dde66d6af86c9dfb36986` | none, bytes unchanged |

Allocator behavior is unchanged: allocation only, with no predecessor
application, agent launch, integration, retry or cleanup. The fixture runs
invented local repositories only; it is not native runtime qualification.

## Active reads

- `scripts/prepare-worktrees.py` and `references/worktree-allocation.md`:
  bundled allocation-only utility, no supervisor dependency or agent launch.
- `plan-execute-cl`, `plan-execute-cx`, `plan-execute-oc` and the matching
  `finishing-a-development-branch-*`: optional host companions; verify
  installed and readable before use, not stage grants or mandatory
  dependencies.

This skill declares no `requires`. Static route and placement checks do not
prove native loading on any host.
