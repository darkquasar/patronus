"""Data-driven OpenCode (-oc) content rules over the real catalog tree.

The rules table is scripts/tests/oc-content-rules.yaml (rejections and identities
only, no required-content phrases). Every check passes trivially when the tree
holds no `-oc` artifact, and the pinned-script check is presence-aware so the
harness can land before the guard scripts do.

The YAML reader below is stdlib-only and deliberately restricted (documented at
the top of the rules file): top-level scalars, top-level flow lists, and one
level of nested mapping whose values are scalars or flow lists. Comments start
at a `#` outside double quotes. The tree root is the checkout by default and
`PATRONUS_OC_CONTENT_ROOT` overrides it so fixture trees can be scanned.
"""
import fnmatch
import hashlib
import os
from pathlib import Path
import re
import unittest

ROOT = Path(os.environ.get("PATRONUS_OC_CONTENT_ROOT") or Path(__file__).resolve().parents[2])
RULES_FILE = Path(__file__).resolve().parent / "oc-content-rules.yaml"
CONTENT_SUFFIXES = (".md", ".yaml", ".json")
HANDOFF = re.compile(r"\{skillsDir\}/([^/\s}]+)/")


def _strip_comment(line):
    quoted = False
    for i, ch in enumerate(line):
        if ch == '"':
            quoted = not quoted
        elif ch == "#" and not quoted:
            return line[:i]
    return line


def _scalar(text):
    text = text.strip()
    if len(text) >= 2 and text[0] == '"' and text[-1] == '"':
        return text[1:-1]
    if text.isdigit():
        return int(text)
    return text


def _flow_list(text):
    text = text.strip()
    if not (text.startswith("[") and text.endswith("]")):
        raise ValueError("expected a flow list, got %r" % text)
    inner = text[1:-1].strip()
    if not inner:
        return []
    items, current, quoted = [], "", False
    for ch in inner:
        if ch == '"':
            quoted = not quoted
            current += ch
        elif ch == "," and not quoted:
            items.append(_scalar(current))
            current = ""
        else:
            current += ch
    items.append(_scalar(current))
    return items


def _value(text):
    text = text.strip()
    if text.startswith("["):
        return _flow_list(text)
    return _scalar(text)


def load_rules(path=RULES_FILE):
    rules, section = {}, None
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = _strip_comment(raw).rstrip()
        if not line.strip():
            continue
        indent = len(line) - len(line.lstrip(" "))
        key, sep, rest = line.strip().partition(":")
        if not sep:
            raise ValueError("unsupported rules line: %r" % raw)
        key = _scalar(key)
        if indent == 0:
            if rest.strip():
                rules[key] = _value(rest)
                section = None
            else:
                rules[key] = {}
                section = key
        elif indent == 2 and section is not None:
            rules[section][key] = _value(rest)
        else:
            raise ValueError("unsupported rules nesting: %r" % raw)
    for required in ("schema_version", "suffix", "forbidden_tokens", "exempt_paths", "neutral_handoffs", "pinned_scripts"):
        if required not in rules:
            raise ValueError("rules file missing %s" % required)
    return rules


def frontmatter(text):
    """Return (mapping, raw block) for a leading `---` block; values are strings."""
    text = text.replace("\r\n", "\n")
    if not text.startswith("---\n"):
        return None, ""
    end = text.find("\n---", 4)
    if end < 0:
        return None, ""
    block = text[4:end]
    mapping = {}
    for line in block.splitlines():
        if not line or line.startswith((" ", "\t")) or line.lstrip().startswith("#"):
            continue
        key, sep, rest = line.partition(":")
        if sep:
            mapping[key.strip()] = _scalar(rest)
    return mapping, block


def manifest_targets(text):
    """Return the `targets:` list of a manifest (flow or block form), or None."""
    lines = text.replace("\r\n", "\n").split("\n")
    for i, line in enumerate(lines):
        if not line.startswith("targets:"):
            continue
        rest = _strip_comment(line[len("targets:"):]).strip()
        if rest:
            return _flow_list(rest) if rest.startswith("[") else [_scalar(rest)]
        items = []
        for following in lines[i + 1:]:
            stripped = _strip_comment(following).strip()
            if not stripped:
                continue
            if following.startswith((" ", "\t")) and stripped.startswith("- "):
                items.append(_scalar(stripped[2:]))
                continue
            break
        return items
    return None


def frontmatter_has_map(block, key):
    """True when `key:` in the frontmatter block carries a mapping (inline or indented)."""
    lines = block.splitlines()
    for i, line in enumerate(lines):
        if not line.startswith(key + ":"):
            continue
        rest = _strip_comment(line[len(key) + 1:]).strip()
        if rest:
            return rest.startswith("{") and rest.endswith("}") and len(rest) > 2
        for following in lines[i + 1:]:
            if not following.strip():
                continue
            return following.startswith((" ", "\t")) and ":" in following and not following.lstrip().startswith("- ")
        return False
    return False


class OpenCodeContentTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.rules = load_rules()
        cls.suffix = "-" + str(cls.rules["suffix"])
        cls.root = ROOT

    def oc_dirs(self):
        base = self.root / "artifacts"
        if not base.is_dir():
            return []
        return sorted(
            p for p in base.glob("*/*" + self.suffix) if p.is_dir()
        )

    def oc_files(self, suffixes):
        for directory in self.oc_dirs():
            for path in sorted(directory.rglob("*")):
                if path.is_file() and path.suffix in suffixes:
                    yield path

    def rel(self, path):
        return path.relative_to(self.root).as_posix()

    def exempt(self, rel):
        return any(fnmatch.fnmatchcase(rel, pattern) for pattern in self.rules["exempt_paths"])

    def test_no_forbidden_host_tokens(self):
        violations = []
        tokens = [(host, token) for host, group in self.rules["forbidden_tokens"].items() for token in group]
        for path in self.oc_files(CONTENT_SUFFIXES):
            rel = self.rel(path)
            if self.exempt(rel):
                continue
            for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
                for host, token in tokens:
                    if token in line:
                        violations.append("%s:%d: %s token %r" % (rel, number, host, token))
        self.assertEqual([], violations, "forbidden host tokens:\n" + "\n".join(violations))

    def test_skill_identity(self):
        problems = []
        for directory in self.oc_dirs():
            if directory.parent.name != "skills":
                continue
            skill = directory / "SKILL.md"
            rel = self.rel(skill)
            if not skill.is_file():
                problems.append("%s: missing SKILL.md" % rel)
                continue
            meta, _ = frontmatter(skill.read_text(encoding="utf-8"))
            if meta is None:
                problems.append("%s: missing frontmatter" % rel)
                continue
            name = str(meta.get("name", ""))
            if name != directory.name:
                problems.append("%s: frontmatter name %r != directory %r" % (rel, name, directory.name))
            if not name.endswith(self.suffix):
                problems.append("%s: name %r does not end with %s" % (rel, name, self.suffix))
            if not str(meta.get("description", "")).startswith("OpenCode only"):
                problems.append("%s: description does not start with 'OpenCode only'" % rel)
        self.assertEqual([], problems, "skill identity:\n" + "\n".join(problems))

    def test_handoffs_name_present_oc_or_neutral(self):
        neutral = set(self.rules["neutral_handoffs"])
        problems = []
        for path in self.oc_files((".md",)):
            rel = self.rel(path)
            if self.exempt(rel):
                continue
            for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
                for name in HANDOFF.findall(line):
                    if name in neutral:
                        continue
                    if not name.endswith(self.suffix):
                        problems.append("%s:%d: handoff to non-%s destination %r" % (rel, number, self.suffix, name))
                        continue
                    present = any(
                        (self.root / "artifacts" / kind / name).is_dir() for kind in ("skills", "instructions")
                    )
                    if not present:
                        problems.append("%s:%d: handoff to absent artifact %r" % (rel, number, name))
        self.assertEqual([], problems, "handoffs:\n" + "\n".join(problems))

    def test_manifests_target_opencode_only(self):
        problems = []
        for directory in self.oc_dirs():
            manifest = directory / "patronus.yaml"
            rel = self.rel(manifest)
            if not manifest.is_file():
                problems.append("%s: missing manifest" % rel)
                continue
            targets = manifest_targets(manifest.read_text(encoding="utf-8"))
            if targets != ["opencode"]:
                problems.append("%s: targets %r != ['opencode']" % (rel, targets))
        self.assertEqual([], problems, "manifest targets:\n" + "\n".join(problems))

    def test_pinned_scripts_are_byte_identical_when_present(self):
        problems = []
        for rel, pinned in self.rules["pinned_scripts"].items():
            path = self.root / rel
            if not path.is_file():
                continue
            actual = hashlib.sha256(path.read_bytes()).hexdigest()
            if actual != pinned:
                problems.append("%s: sha256 %s != pinned %s" % (rel, actual, pinned))
        self.assertEqual([], problems, "pinned scripts:\n" + "\n".join(problems))

    def test_agent_frontmatter_shape(self):
        problems = []
        for directory in self.oc_dirs():
            if directory.parent.name != "agents" or not directory.name.startswith("patronus-"):
                continue
            agent = directory / "agent.md"
            rel = self.rel(agent)
            if not agent.is_file():
                problems.append("%s: missing agent.md" % rel)
                continue
            meta, block = frontmatter(agent.read_text(encoding="utf-8"))
            if meta is None:
                problems.append("%s: missing frontmatter" % rel)
                continue
            if meta.get("mode") != "subagent":
                problems.append("%s: mode %r != 'subagent'" % (rel, meta.get("mode")))
            if not frontmatter_has_map(block, "permission"):
                problems.append("%s: missing permission map" % rel)
            if not str(meta.get("description", "")).startswith("OpenCode only"):
                problems.append("%s: description does not start with 'OpenCode only'" % rel)
        self.assertEqual([], problems, "agent frontmatter:\n" + "\n".join(problems))


if __name__ == "__main__":
    unittest.main()
