# Source and provenance

Adapted on 2026-10-13 from a retained rust-style-pi handoff. Original snapshot is
unchanged. Its source record
is retained in references/upstream-source.md with its author-machine path removed.

The 156 selected rules retain their upstream examples. Microsoft guidelines and
both MIT license files are byte-identical. See NOTICE.
New portable instructions
remove the predecessor's project-specific edition/MSRV, unsafe, dependency,
feature and protocol assumptions. They do not change the vendored rule examples.
The predecessor curated 13 categories; 49 optional related-rule links referenced
37 absent targets in excluded categories. Those broken Markdown links become
plain labels marked not bundled. All remaining distributed links resolve; no
automatic fetch or unnecessary extra rule categories are introduced.
One explicit portability erratum is added to proj-msrv-declare.md and its index
entry: the retained edition-2024/Rust-1.80 example requires Rust 1.85 or newer.
The incompatible upstream example is retained with the warning, not recommended
as a working manifest. No general edition/MSRV choice is imposed.

Compilation guidance generalizes reported cache-isolation and resource-admission
experience. It promises no fixed speedup or universal resource setting. Historical
case-study details remain outside the distributed instructions. No Rust scripts,
caches, semaphore or compiler experiment is shipped.

Patronus owns one shared guidance/reference artifact with thin harness entrypoints
rust-style (Claude), rust-style-cx, rust-style-pi and rust-style-oc. No core profile
membership. This local source delivery is not registry publication or observed
native compliance. Original entrypoint remains in the handoff snapshot for audit.

New authored text follows Patronus's GPL-3.0 license (LICENSE); vendored material
retains its separate MIT terms. Upstream version/edition claims in the original
source record describe the retrieved snapshot, not current Rust advice.
