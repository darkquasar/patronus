# Export, serving and benchmark boundaries

Children query existing data and write only assigned reports. HTML, wiki, vault,
SVG, GraphML, database exports, benchmarks and MCP-server startup are not queries.
Do not generate them automatically because a graph exists or exceeds a size limit.

Coordinator-only actions require explicit stage and resource budget authority.
Before any export, record graph hash/root/revision, exact tool version, format,
destination owner, input/output limits and retention policy. Large visualizations
need measured memory admission; benchmark claims require observed measurements,
not upstream token-savings promises. Protect the original graph on failure.

Remote database push is publication and needs separate destination/network/auth
approval; an indexing grant alone is insufficient. Never put credentials into
commands, reports or distributable examples. Use only the operator's approved
secret injection mechanism, and preserve partial evidence without secret values.

Serving is a coordinator lifecycle operation, not a child fallback when tools are
missing. Bind only the separately approved interface and expose the qualified
query-only surface. An HTTP client's exit must not stop that shared service.
