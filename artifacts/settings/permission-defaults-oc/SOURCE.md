# Source and qualification

Original Patronus setting artifact, version 1.0.0. Authored from approved
OpenCode core-lane spec OC-D9 and OC-R6, and plan OC-T6 (2026-10-09).

The OpenCode adapter declares `splitLeaves: [permission.edit]`. This artifact
owns exactly three pattern leaves: `* = allow`, `*.env = deny`, and
`.env.* = deny`. It neither changes `permission.bash` nor denies all edits.
Sibling user permission rules remain user-owned; removal restores only owned
leaves. This is defence in depth beside the runner guards, not a credential
scanner or a complete permission policy.

Structural delivery and ownership are tested with invented data by the
application gate. Native OpenCode matching and enforcement remain
runtime-pending and require separately authorized qualification.
