#!/usr/bin/env bash
# Headless smoke test for a generated explanation notebook.
#
# Two checks:
#   1. Export to HTML (runs every cell top to bottom) and confirm no cell errored
#      and each view's payload made it into the output.
#   2. Mermaid syntax lint (OPTIONAL) — extract every Mermaid block and parse it
#      with `mmdc`. Mermaid renders in the BROWSER, so a syntax error exports
#      "clean" and only breaks on screen; this catches it headlessly. `mmdc` is a
#      heavy Node+Chromium dependency, so if it's absent we skip with a clear
#      message rather than failing — you just don't get the Mermaid guarantee.
#
# Usage:  bash verify.sh <notebook.py> [path-to-.venv]
# Note: no `pipefail` — `grep -o ... | wc -l` legitimately has grep exit 1 on
# zero matches, which under pipefail would abort the script mid-count.
set -eu

NB="${1:?usage: verify.sh <notebook.py> [venv]}"
VENV="${2:-.venv}"
MARIMO="$VENV/bin/marimo"
PY="$VENV/bin/python"
OUT="$(mktemp -t explverify.XXXXXX).html"

[ -x "$MARIMO" ] || { echo "no marimo at $MARIMO — bootstrap the env first"; exit 2; }

# The notebook imports its sibling helpers; make sure they're present.
NBDIR="$(cd "$(dirname "$NB")" && pwd)"
for h in excalidraw_scene.py markmap_html.py; do
  [ -f "$NBDIR/$h" ] || echo "WARN: $h not found next to notebook (Excalidraw/Markmap import may fail)"
done

echo "Exporting $NB ..."
"$MARIMO" export html "$NB" -o "$OUT" >/dev/null 2>&1 || { echo "FAIL: export errored"; exit 1; }

err=$(grep -o "Traceback (most recent" "$OUT" | wc -l | tr -d ' ')
mde=$(grep -o "MultipleDefinitionError" "$OUT" | wc -l | tr -d ' ')
mermaid=$(grep -o "mermaid" "$OUT" | wc -l | tr -d ' ')
markmap=$(grep -o "markmap-autoloader" "$OUT" | wc -l | tr -d ' ')
exca=$(grep -o "excalidraw" "$OUT" | wc -l | tr -d ' ')

echo "  tracebacks:            $err   (want 0)"
echo "  MultipleDefinitionErr: $mde   (want 0)"
echo "  mermaid payload:       $mermaid"
echo "  markmap payload:       $markmap"
echo "  excalidraw payload:    $exca"
echo "  export: $OUT"

# ---- Mermaid syntax lint (optional; needs mmdc) --------------------------------
# Extract Mermaid blocks: triple-quoted strings whose first meaningful line is a
# known Mermaid diagram keyword. Best-effort — diagrams built by f-string/concat
# won't be seen (reported as a count so you know how many were checked).
mmfail=0
MMDIR="$(mktemp -d -t explmmd.XXXXXX)"
mmcount=$("$PY" - "$NB" "$MMDIR" <<'PY' 2>/dev/null || echo ERR
import re, sys, textwrap, pathlib
nb = pathlib.Path(sys.argv[1]).read_text()
out = pathlib.Path(sys.argv[2])
TYPES = ("sequenceDiagram", "flowchart", "graph", "quadrantChart", "timeline",
         "mindmap", "stateDiagram-v2", "stateDiagram", "erDiagram", "classDiagram",
         "gantt", "journey", "pie", "gitGraph", "C4Context", "xychart-beta",
         "requirementDiagram")
n = 0
for block in re.findall(r'(?:"""|\'\'\')(.*?)(?:"""|\'\'\')', nb, re.S):
    lines = block.splitlines()
    first = next((l.strip() for l in lines if l.strip() and not l.strip().startswith("%%")), "")
    head = first.split("(")[0].split()[0] if first else ""
    if head in TYPES or any(first.startswith(t) for t in TYPES):
        n += 1
        (out / f"block{n}.mmd").write_text(textwrap.dedent("\n".join(lines)).strip("\n") + "\n")
print(n)
PY
)

echo ""
if [ "$mmcount" = "ERR" ]; then
  echo "Mermaid lint: SKIPPED (could not scan the notebook for Mermaid blocks)."
elif [ "$mmcount" = "0" ]; then
  echo "Mermaid lint: no Mermaid blocks detected (nothing to check)."
elif ! command -v mmdc >/dev/null 2>&1; then
  # Graceful degradation — surfaced plainly so it can be relayed to the user.
  echo "Mermaid lint: SKIPPED — we couldn't check Mermaid consistency because mmdc is missing on this system."
  echo "  ($mmcount Mermaid block(s) were detected but NOT validated; a syntax error would only show on screen.)"
  echo "  To enable strict Mermaid checking: npm i -g @mermaid-js/mermaid-cli"
else
  printf '{"args":["--no-sandbox"]}' > "$MMDIR/puppeteer.json"
  echo "Mermaid lint: checking $mmcount block(s) with mmdc ..."
  for f in "$MMDIR"/block*.mmd; do
    if mmdc -p "$MMDIR/puppeteer.json" -i "$f" -o "$f.svg" >/dev/null 2>"$f.err"; then
      echo "  OK:   $(basename "$f")"
    else
      mmfail=$((mmfail + 1))
      echo "  FAIL: $(basename "$f") —"
      sed 's/^/        /' "$f.err" | grep -v '^ *$' | head -6
      echo "        --- diagram source ---"
      sed 's/^/        /' "$f" | head -30
    fi
  done
  [ "$mmfail" = "0" ] && echo "  Mermaid: all $mmcount block(s) parse."
fi
rm -rf "$MMDIR"

# ---- YAML fenced-block lint ----------------------------------------------------
# Scans the notebook AND any sibling .md files it loads (read_doc) for ```yaml
# fenced blocks and parses each. Catches the skewed/mis-indented YAML that renders
# as a garbled block on screen. Needs PyYAML; degrades gracefully if absent.
echo ""
yamlfail=0
yamlout="$("$PY" - "$NB" "$NBDIR" <<'PY' 2>/dev/null || echo "__ERR__"
import re, sys, pathlib
try:
    import yaml
except Exception:
    print("SKIP"); sys.exit(0)
nb = pathlib.Path(sys.argv[1]); nbdir = pathlib.Path(sys.argv[2])
sources = [nb] + sorted(nbdir.glob("*.md"))
fence = re.compile(r"```ya?ml\s*\n(.*?)```", re.S)
checked = fails = 0
for src in sources:
    try:
        text = src.read_text(encoding="utf-8")
    except Exception:
        continue
    for i, block in enumerate(fence.findall(text), 1):
        checked += 1
        try:
            list(yaml.safe_load_all(block))
        except Exception as e:
            fails += 1
            msg = str(e).splitlines()[0][:160]
            print(f"FAIL\t{src.name} block#{i}\t{msg}")
print(f"COUNT\t{checked}\t{fails}")
PY
)"
if [ "$yamlout" = "__ERR__" ]; then
  echo "YAML lint: SKIPPED (scan failed)."
elif printf '%s\n' "$yamlout" | grep -q '^SKIP$'; then
  echo "YAML lint: SKIPPED — PyYAML not installed (pip install pyyaml to enable)."
else
  _yc="$(printf '%s\n' "$yamlout" | awk -F'\t' '/^COUNT/{print $2}')"
  yamlfail="$(printf '%s\n' "$yamlout" | awk -F'\t' '/^COUNT/{print $3}')"
  if [ "${_yc:-0}" = "0" ]; then
    echo "YAML lint: no yaml blocks detected."
  elif [ "${yamlfail:-0}" = "0" ]; then
    echo "YAML lint: all $_yc yaml block(s) parse."
  else
    echo "YAML lint: $yamlfail of $_yc yaml block(s) FAILED to parse —"
    printf '%s\n' "$yamlout" | awk -F'\t' '/^FAIL/{printf "  %s: %s\n", $2, $3}'
  fi
fi

# ---- Verdict -------------------------------------------------------------------
if [ "$err" != "0" ] || [ "$mde" != "0" ]; then
  echo "FAIL: notebook has Python errors"; exit 1
fi
if [ "$mmfail" != "0" ]; then
  echo "FAIL: $mmfail Mermaid block(s) have syntax errors — fix before serving."; exit 1
fi
if [ "${yamlfail:-0}" != "0" ]; then
  echo "FAIL: $yamlfail YAML block(s) are malformed — fix before serving."; exit 1
fi
echo "OK: notebook runs clean."
