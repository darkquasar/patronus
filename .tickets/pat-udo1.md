---
id: pat-udo1
status: open
deps: []
links: []
created: 2026-07-12T04:47:11Z
type: task
priority: 1
assignee: darkquasar
external-ref: docs/specs/01-lifecycle-and-test-surface
parent: pat-zr8z
tags: [test-surface]
---
# Harden TestBuildProducesLoadableIndex into a catalog validity gate (Class C)

Plan Task 8. Files: cmd/patronus/build_test.go. Reads the catalog's SHAPE, never its PINS: digests well-formed hex, every requires: resolves, every profile slot names a real item. Use Catalog.ItemNames() (exists). ProfileLayers has SEVEN slots and Memory is a SCALAR — miss one and the gate silently stops covering that layer. build-registry.yml is paths:-gated and does NOT run on a Go-only PR, so zero canaries would ship a typo green.

## Acceptance Criteria

Commenting a real item out of profiles/core.yaml makes the gate FAIL — verified live, not asserted

