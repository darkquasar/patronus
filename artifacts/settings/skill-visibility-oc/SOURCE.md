# Source and qualification

Original Patronus computed setting artifact, version 1.0.0. Authored from
approved OpenCode core-lane spec OC-D7 and OC-R6, and plan OC-T6
(2026-10-09), with conservative refresh/retirement decision OP-02.

The shared descriptor rule discovers skills only in roots the selected target
reads. For each registered foreign suffix family with intersecting root paths
and detected skills, it emits one quoted pattern leaf under `permission.skill`
with value `deny`. It never emits a static placeholder or a blanket deny.
Preview/deploy messages name the detected count, family and settings file.
Re-run `patronus update skill-visibility-oc` after adding or removing foreign
lane skills: computed settings refresh even at the same artifact version.
Obsolete unchanged owned leaves restore their recorded prior value/presence;
edited leaves remain owned and produce a warning. Equal unmanaged rules are
not adopted, and removal affects owned leaves only.

Application tests use invented catalogs and roots. Native OpenCode skill
visibility and permission enforcement remain runtime-pending. This artifact
provides structural placement, not proof of native enforcement.
