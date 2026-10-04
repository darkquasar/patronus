---
name: using-git-worktrees-pi
description: "Choose worktree ownership and cleanup authority before launch"
---

# Worktree choice before launch

The coordinator records source root, status, branch/HEAD, owner and intended files before any writer. One writer per worktree. An existing allocated cwd belongs to its recorded owner; leaves use it and do not allocate nested worktrees. Git dir/common-dir inequality alone is insufficient for submodules: inspect worktree list and superproject state too.

## Two explicit choices

- **Native managed allocation:** require clean source and a named baseRef resolved to recorded baseCommit. At pi-subagents 0.72.1 “preserve” means patch capture, not physical retention. Before launch obtain an exact cleanup grant for the runtime-owned temporary allocation/branch after successful capture and supported settlement checks: normal finalization may use forced worktree removal and branch deletion. Bind the grant to this allocation policy/run, source/base and permitted outputs; record actual paths from the native handoff as soon as known. Do not assume a raw SHA is an accepted baseRef. worktree.cleanup at this pin is plan-only, not a delayed physical-retention switch.
- **Coordinator-retained cwd:** select this BEFORE launch when physical retention, later cleanup approval or interactive private-LSP trust approval is required. The coordinator allocates the exact named branch/path under its Git grant and passes cwd. Do not set worktree:true as well. Retained cwd never inherits native auto-cleanup authority. No directory-name ownership inference.

## Failure and verification

No fallback-in-place, dirty-source stash/commit or automatic dependency installation. Setup commands need explicit bounded provisioning authority. Finite setup hooks must await all descendants; completion cannot prove absence of undisclosed processes. Unknown descendants/ownership, capture failure or drift preserves work and escalates. Record actual capture paths/hashes and verify readability before supported settlement cleanup. Do not delete unknown work based on a directory name or mere success status.

Run approved baseline tests with measured memory/container headroom and the project resource lock. Missing caches/tools or failing baseline stops for a decision, never an installer/model/protocol fallback. Native automatic worktrees use source reads until exact prelaunch private-LSP bootstrap is qualified. Interactive private-LSP approval in a retained cwd is separate; shared symbols/graphs describe their own checkout and may be stale. Never retarget shared services.

Integration, publication and discard are distinct grants. A cleanup grant does not grant any of them. Keep native mission state/reports outside temporary runtime storage and preserve patch, base/head and ownership evidence for recovery.
