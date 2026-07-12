---
id: pat-resg
status: open
deps: []
links: []
created: 2026-07-12T08:24:36Z
type: bug
priority: 1
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
tags: [test-surface]
---
# install plan omits type:setting artifacts — it under-reports what --deploy will change

Found while migrating hardened_profile_integration_test.go (pat-1gyd). Verified: '--profile hardened --tool claude' (plan-only) prints 2 MERGE rows (context7, skills-dispatch-activate) and NO native-sandbox row, but the same install with --deploy DOES write sandbox:{enabled:true,autoAllowBashIfSandboxed:true} into ~/.claude/settings.json. Same on codex (sandbox_mode). The plan is the screen a user reviews before consenting to --deploy, so silently changing a security-relevant setting it never listed is a consent defect, not cosmetics. Likely in the diff/render path for type:setting artifacts (artifacts/settings/native-sandbox). NOTE: this is why hardened's Class-B sandbox guarantee cannot be asserted plan-only.

## Acceptance Criteria

A plan-only 'install --profile hardened --tool claude' shows a row for native-sandbox targeting ~/.claude/settings.json; today it shows none, yet --deploy writes settings.sandbox={enabled:true,...}


## Notes

**2026-07-12T08:27:59Z**

WIDER THAN type:setting — the plan under-reports settings.json MERGEs generally. Verified on 'install --profile safe-git --tool claude' (plan-only): it lists all 7 hook SCRIPTS as CREATE rows, but only ONE ~/.claude/settings.json MERGE row (skills-dispatch-activate). Yet --deploy folds SEVEN entries into settings.json (the deployed test asserts 4 PreToolUse + 3 SessionStart + statusLine). So a user reviewing the plan sees one settings.json change and gets seven. Same class of defect as the missing native-sandbox row: the consent screen does not describe the change. Both are why the Class-B profile tests could not be made plan-only and had to assert against profile.Resolve / the lock instead.
