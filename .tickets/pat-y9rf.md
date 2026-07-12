---
id: pat-y9rf
status: open
deps: []
links: []
created: 2026-07-12T02:28:55Z
type: task
priority: 2
assignee: darkquasar
---
# Remove cmd/patronus/testdata/tk once the fixture catalog lands


## Notes

**2026-07-12T02:38:13Z**

Depends on the fixture-catalog refactor, which is being specced separately (not tracked as a ticket). Not actionable until that lands — the vendored testdata/tk is what makes the ~19 core-profile integration tests pass offline, since they build from the real recipes/ tree and so inherit the real pin.
