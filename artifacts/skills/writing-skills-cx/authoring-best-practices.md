# Authoring and progressive disclosure

Adapted from the legacy authoring reference as design guidance, not a foreign
runtime contract. Codex-specific behavior still needs native qualification.

- Give a concrete trigger in the description, then keep the entry's mechanism
  and expected outcome easy to scan. Use stable names and consistent vocabulary.
- Include one representative example with inputs, steps and expected result.
  Avoid multiple languages that duplicate the same lesson.
- Keep essential guards in the entry. Put longer examples and domain references
  one level down; show when to read each file. Avoid circular reference chains.
- Resolve local sidecars through `{skillDir}`. Exact companion reads use
  installed paths such as `{skillsDir}/tdd-cx/SKILL.md`. Declare every distributed sidecar and required
  companion in the manifest. Paths in snippets describing a user's application
  are examples, not bundled skill files.
- Match freedom to risk: a flexible heuristic for ordinary design, an explicit
  precondition and refusal for destructive work. Do not infer scope or authority
  from a tool being present.
- Distinguish authoring, static validation, installation and runtime activation.
  Record what was checked and what was not. Test with invented data; live service,
  auth, installation or model calls require their own grant.
- Check the whole bundle after edits. Preserve licensing and source provenance,
  and update the artifact version when changing a released payload.

The optional diagram convention lives at `{skillDir}/graphviz-conventions.dot`.
No renderer is shipped in this port: generating SVG would need an approved output
path and separately available tools, not an automatic install into the skill.
