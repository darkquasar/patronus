from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[2]
CURRENT_GUIDANCE = (
    "README.md",
    "docs/agent-artifacts.mdx",
    "docs/pi-core.md",
    "docs/package-delivery.md",
    "artifacts/skills/web-research-pi/SKILL.md",
    "artifacts/skills/web-research-pi/SOURCE.md",
    "artifacts/skills/web-research-pi/references/qualification.md",
    "artifacts/skills/web-research-pi/references/runbook.md",
    "artifacts/instructions/web-research-pointer-pi/INSTRUCTIONS.md",
)
FORBIDDEN_CURRENT_TEXT = (
    ".patronus/packages/pi-web-access",
    "packages/pi-web-access",
    "receipt-owned subagents/web payloads",
    "receipt-derived native source",
    "Static delivery is placed",
    "selected local source",
    "Planned Pi role example",
    "planned local-source researcher",
    "In the planned workflow",
    "planned identities",
)


class PiNativeContentTest(unittest.TestCase):
    def test_obsolete_static_qualification_programs_are_absent(self):
        for relative in (
            "scripts/qualification/core-smoke",
            "scripts/qualification/web-smoke",
            "scripts/qualification/qp03",
            "docs/pi-qualification/functional-smoke-results.json",
        ):
            self.assertFalse((ROOT / relative).exists(), relative)

    def test_current_guidance_uses_pi_owned_package_lifecycle(self):
        combined = "\n".join((ROOT / relative).read_text() for relative in CURRENT_GUIDANCE)
        for stale in FORBIDDEN_CURRENT_TEXT:
            self.assertNotIn(stale, combined)
        self.assertIn("npm:pi-subagents@0.72.1", combined)
        self.assertIn("npm:pi-web-access@0.35.0", combined)
        self.assertIn("Pi/npm own", combined)

    def test_pi_core_describes_the_current_profile(self):
        text = (ROOT / "docs/pi-core.md").read_text()
        self.assertIn("`core-profile-pi` 2.0.0", text)
        self.assertIn("npm:pi-subagents@0.72.1", text)
        self.assertIn("npm:pi-web-access@0.35.0", text)
        for stale in (
            "three global recipes",
            "dependency archives are locally built",
            "unpublished bytes",
            "input manifests/lock/receipts",
            "all three compatible",
        ):
            self.assertNotIn(stale, text)

    def test_generic_directory_package_docs_distinguish_pi_manager_recipes(self):
        text = (ROOT / "docs/package-delivery.md").read_text()
        self.assertIn("manager: pi", text)
        self.assertIn("Pi/npm own", text)

    def test_corrected_guidance_is_asserted_per_document(self):
        agent_docs = (ROOT / "docs/agent-artifacts.mdx").read_text()
        self.assertIn("npm:pi-subagents@0.72.1", agent_docs)
        self.assertIn("## Pi role example", agent_docs)

        source = (ROOT / "artifacts/skills/web-research-pi/SOURCE.md").read_text()
        self.assertIn("source-hash", source)
        self.assertIn("required Pi-installed package", source)

        qualification = (
            ROOT / "artifacts/skills/web-research-pi/references/qualification.md"
        ).read_text()
        self.assertIn("source-hash record", qualification)

        pointer = (
            ROOT / "artifacts/instructions/web-research-pointer-pi/INSTRUCTIONS.md"
        ).read_text()
        self.assertIn("npm:pi-web-access@0.35.0", pointer)
        self.assertIn("Pi/npm own", pointer)


if __name__ == "__main__":
    unittest.main()
