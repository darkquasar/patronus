# Hooks and freshness notices

The Pi overlay ships no indexing hook. A notice that files changed means the
snapshot may be stale; it does not grant permission to rebuild it. Children must
not install commit hooks, alter instruction files or run a native integration
installer to make Graphify always-on.

Only a coordinator with explicit stage/budget and repository-owner authority may
consider a hook or scheduled refresh outside this static overlay. Review existing
hooks and ownership first, account for subprocess/provider cost, bound concurrency
and preserve prior bytes. Never replace unrelated hooks or infer cleanup authority
from their names. Automatic rebuild behavior requires its own reviewed deployment;
this guide does not enable it.

HEAD time and tracked status do not cover all untracked or externally modified
inputs. Use the snapshot inventory and current source to bound freshness claims;
see [update guidance](update.md). Core role skill pointers are guidance, not proof
of hook inheritance or actual MCP access.
