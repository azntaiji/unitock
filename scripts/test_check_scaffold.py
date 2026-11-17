"""Tests for check_scaffold.py. Standard library only.

Run from the repository root:
    python3 -m unittest discover -s scripts -p "test_*.py"

The tests copy the scaffold's control files into a temporary directory, reset them to a
known state, and mutate them, so they pass at any phase of a real project.
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import check_scaffold as cs  # noqa: E402

REPO = Path(__file__).resolve().parent.parent

STATUS = "# STATUS\n\nPhase: {phase}\nPhase name: Test\nGate: G{phase} not requested\n"
GATES = "# GATES\n\n## Ledger\n\n{ledger}\n"
PLAN = "# Plan\n\n## Phase 0 — Foundations\n\n## Phase 1 — Skeleton\n\n## Phase 2 — More\n"
CHANGELOG = "# Changelog\n\n## [Unreleased]\n{releases}"
README = (
    "# Demo\n\n"
    "[![version](https://img.shields.io/badge/version-{badge}-blue)](CHANGELOG.md)\n"
    "[![phase](https://img.shields.io/badge/phase-{phase}%20test-orange)](docs/PLAN.md)\n"
)
G0 = "G0: APPROVED by Ada Example, 2026-01-15. Version: {version}. Notes: ok"


def criteria(passes: bool) -> str:
    item = {
        "id": "P0-01",
        "phase": 0,
        "description": "Spec is complete.",
        "verify": "Read it.",
        "passes": passes,
        "evidence": "read docs/SPEC.md; no placeholders" if passes else "",
    }
    return json.dumps({"criteria": [item]})


class ScaffoldCase(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = Path(tempfile.mkdtemp())
        self.addCleanup(shutil.rmtree, self.tmp, ignore_errors=True)
        (self.tmp / "docs").mkdir()
        self.write("AGENTS.md", "# AGENTS.md\n")
        self.write("CLAUDE.md", "@AGENTS.md\n")
        self.write("docs/PLAN.md", PLAN)
        self.write(".scaffold.json", "{}")
        self.state()

    def write(self, rel: str, text: str) -> None:
        path = self.tmp / rel
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text, encoding="utf-8")

    def state(
        self,
        phase: int = 0,
        ledger: str = "",
        version: str = "0.0.0",
        badge: str | None = None,
        releases: str = "",
        passes: bool = False,
    ) -> None:
        self.write("STATUS.md", STATUS.format(phase=phase))
        self.write("GATES.md", GATES.format(ledger=ledger))
        self.write("VERSION", version + "\n")
        self.write("CHANGELOG.md", CHANGELOG.format(releases=releases))
        shown = (badge if badge is not None else version).replace("-", "--")
        self.write("README.md", README.format(badge=shown, phase=phase))
        self.write("docs/acceptance.json", criteria(passes))

    def released(self, **overrides) -> None:
        """Phase 1 with G0 approved at 0.1.0 and everything consistent."""
        args = dict(
            phase=1,
            ledger=G0.format(version="0.1.0"),
            version="0.1.0",
            releases="\n## [0.1.0] - 2026-01-15\n",
            passes=True,
        )
        args.update(overrides)
        self.state(**args)

    def errors(self) -> list[str]:
        return cs.run_checks(self.tmp).errors

    def assertClean(self) -> None:
        self.assertEqual(self.errors(), [])

    def assertErrorContains(self, fragment: str) -> None:
        errors = self.errors()
        self.assertTrue(
            any(fragment in e for e in errors), f"no error containing {fragment!r} in {errors}"
        )


class TemplateTest(unittest.TestCase):
    def test_repository_passes_its_own_check(self) -> None:
        self.assertEqual(cs.run_checks(REPO).errors, [])


class GateTests(ScaffoldCase):
    def test_fresh_project_is_clean(self) -> None:
        self.assertClean()

    def test_phase_ahead_of_gate_fails(self) -> None:
        self.state(phase=1)
        self.assertErrorContains("G0 not APPROVED")

    def test_full_release_flow_is_clean(self) -> None:
        self.released()
        self.assertClean()

    def test_approved_gate_with_failing_criterion_fails(self) -> None:
        self.released(passes=False)
        self.assertErrorContains("does not pass")

    def test_rejection_after_approval_revokes_gate(self) -> None:
        ledger = (
            G0.format(version="none")
            + "\nG0: REJECTED by Ada Example, 2026-01-16. Notes: found a gap"
        )
        self.state(phase=1, ledger=ledger, passes=True)
        self.assertErrorContains("G0 not APPROVED")

    def test_malformed_ledger_line_fails(self) -> None:
        self.state(ledger="G0: approved, looks good")
        self.assertErrorContains("malformed ledger line")

    def test_approval_without_version_field_fails(self) -> None:
        self.state(ledger="G0: APPROVED by Ada Example, 2026-01-15. Notes: ok")
        self.assertErrorContains("needs 'Version:")

    def test_ledger_examples_in_code_fences_are_ignored(self) -> None:
        self.write("GATES.md", "# GATES\n\n```\n" + G0.format(version="9.9.9") + "\n```\n")
        self.assertClean()

    def test_passing_criterion_needs_evidence(self) -> None:
        data = json.loads(criteria(True))
        data["criteria"][0]["evidence"] = ""
        self.write("docs/acceptance.json", json.dumps(data))
        self.assertErrorContains("'evidence' is empty")

    def test_missing_plan_section_fails(self) -> None:
        self.write("docs/PLAN.md", "# Plan\n\n## Phase 1 — Skeleton\n")
        self.assertErrorContains("no '## Phase 0' section")


class VersionTests(ScaffoldCase):
    def test_version_bump_without_approval_fails(self) -> None:
        self.state(version="0.1.0", releases="\n## [0.1.0] - 2026-01-15\n")
        self.assertErrorContains("no approved version")

    def test_approved_version_not_yet_applied_fails(self) -> None:
        self.released(version="0.0.0", releases="")
        self.assertErrorContains("last approved version in GATES.md is 0.1.0")

    def test_changelog_must_match_version(self) -> None:
        self.released(releases="\n## [0.0.9] - 2026-01-15\n")
        self.assertErrorContains("newest release is [0.0.9]")

    def test_readme_badge_must_match_version(self) -> None:
        self.released(badge="0.0.9")
        self.assertErrorContains("version badge shows 0.0.9")

    def test_prerelease_badge_with_escaped_dash(self) -> None:
        self.released(
            ledger=G0.format(version="1.0.0-rc.1"),
            version="1.0.0-rc.1",
            releases="\n## [1.0.0-rc.1] - 2026-01-15\n",
        )
        self.assertClean()

    def test_approved_versions_must_increase(self) -> None:
        ledger = G0.format(version="0.2.0") + "\nR: APPROVED by Ada Example, 2026-01-16. Version: 0.1.0."
        self.released(ledger=ledger, releases="\n## [0.1.0] - 2026-01-16\n")
        self.assertErrorContains("does not come after")

    def test_invalid_version_fails(self) -> None:
        self.state(version="1.0")
        self.assertErrorContains("not a valid SemVer")

    def test_python_mirrors_use_pep440(self) -> None:
        self.released(
            ledger=G0.format(version="1.2.0-rc.1"),
            version="1.2.0-rc.1",
            releases="\n## [1.2.0-rc.1] - 2026-01-15\n",
        )
        self.write(
            ".scaffold.json",
            json.dumps(
                {
                    "version_mirrors": [
                        {"path": "pyproject.toml", "kind": "pyproject"},
                        {"path": "src/demo/__init__.py", "kind": "python_dunder"},
                    ]
                }
            ),
        )
        self.write("pyproject.toml", '[project]\nname = "demo"\nversion = "1.2.0rc1"\n')
        self.write("src/demo/__init__.py", '__version__ = "1.2.0rc1"\n')
        self.assertClean()
        self.write("pyproject.toml", '[project]\nname = "demo"\nversion = "1.2.0-rc.1"\n')
        self.assertErrorContains("expected 1.2.0rc1")

    def test_package_json_and_regex_mirrors(self) -> None:
        self.released()
        self.write(
            ".scaffold.json",
            json.dumps(
                {
                    "version_mirrors": [
                        {"path": "package.json", "kind": "package_json"},
                        {
                            "path": "version.go",
                            "kind": "regex",
                            "pattern": 'Version\\s*=\\s*"([^"]+)"',
                        },
                    ]
                }
            ),
        )
        self.write("package.json", '{"name": "demo", "version": "0.1.0"}')
        self.write("version.go", 'package version\n\nconst Version = "0.1.0"\n')
        self.assertClean()
        self.write("package.json", '{"name": "demo", "version": "0.0.1"}')
        self.assertErrorContains("package.json has 0.0.1")


class ContextTests(ScaffoldCase):
    def test_status_over_budget_fails(self) -> None:
        self.write("STATUS.md", STATUS.format(phase=0) + "line\n" * 100)
        self.assertErrorContains("STATUS.md has")

    def test_agents_over_budget_fails(self) -> None:
        self.write("AGENTS.md", "rule\n" * 200)
        self.assertErrorContains("AGENTS.md has 200 lines")

    def test_claude_md_must_not_import_status(self) -> None:
        self.write("CLAUDE.md", "@AGENTS.md\n@STATUS.md\n@docs/archive/phase-1.md\n")
        errors = self.errors()
        self.assertEqual(sum("CLAUDE.md imports" in e for e in errors), 2)

    def test_stray_root_markdown_fails(self) -> None:
        self.write("PHASE_2_SUMMARY.md", "done\n")
        self.assertErrorContains("PHASE_2_SUMMARY.md")

    def test_report_file_under_docs_fails_but_archive_is_exempt(self) -> None:
        self.write("docs/archive/phase-0.md", "archived\n")
        self.assertClean()
        self.write("docs/validation-results.md", "raw output\n")
        self.assertErrorContains("docs/validation-results.md")

    def test_readme_phase_badge_must_match_status(self) -> None:
        self.released()
        readme = (self.tmp / "README.md").read_text(encoding="utf-8")
        self.write("README.md", readme.replace("phase-1%20test", "phase-0%20test"))
        self.assertErrorContains("phase badge shows 0")


@unittest.skipUnless(shutil.which("git"), "git not available")
class TagTests(ScaffoldCase):
    def setUp(self) -> None:
        super().setUp()
        self.run_git("init", "-q")

    def run_git(self, *args: str) -> None:
        # Ignore the developer's global and system git config (signing, hooks, etc.).
        env = dict(os.environ, GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM="1")
        subprocess.run(
            ["git", "-C", str(self.tmp), "-c", "user.name=Test", "-c", "user.email=t@example.com",
             *args],
            check=True,
            capture_output=True,
            env=env,
        )

    def commit(self, message: str) -> None:
        self.run_git("add", "-A")
        self.run_git("commit", "-q", "-m", message)

    def tag(self, name: str) -> None:
        self.run_git("tag", "-a", name, "-m", name)

    def warnings(self) -> list[str]:
        return cs.run_checks(self.tmp).warnings

    def test_approved_release_tag_is_clean(self) -> None:
        self.released()
        self.commit("chore(release): v0.1.0 (close G0)")
        self.tag("v0.1.0")
        self.assertClean()
        self.assertFalse(any("has no tag" in w for w in self.warnings()))

    def test_tag_for_unapproved_version_fails(self) -> None:
        self.released()
        self.commit("chore(release): v0.1.0 (close G0)")
        self.tag("v0.1.0")
        self.tag("v0.2.0")
        self.assertErrorContains("tag v0.2.0 names a version that is not APPROVED")

    def test_tag_on_wrong_commit_fails(self) -> None:
        self.commit("chore: start")
        self.tag("v0.1.0")
        self.released()
        self.commit("chore(release): v0.1.0 (close G0)")
        self.assertErrorContains("tag v0.1.0 points at a commit whose VERSION is 0.0.0")

    def test_approved_version_without_tag_only_warns(self) -> None:
        self.released()
        self.commit("chore(release): v0.1.0 (close G0)")
        self.assertClean()
        self.assertTrue(any("0.1.0 has no tag" in w for w in self.warnings()))

    def test_non_release_tags_are_ignored(self) -> None:
        self.commit("chore: start")
        self.tag("vnext")
        self.assertClean()

    def test_check_can_be_disabled(self) -> None:
        self.write(".scaffold.json", json.dumps({"check_tags": False}))
        self.commit("chore: start")
        self.tag("v9.9.9")
        self.assertClean()

    def test_directory_inside_another_repository_is_skipped(self) -> None:
        self.commit("chore: start")
        self.tag("v9.9.9")
        nested = self.tmp / "nested"
        nested.mkdir()
        for name in ("AGENTS.md", "CLAUDE.md", "STATUS.md", "GATES.md", "VERSION",
                     "CHANGELOG.md", "README.md", ".scaffold.json"):
            shutil.copy(self.tmp / name, nested / name)
        shutil.copytree(self.tmp / "docs", nested / "docs")
        self.assertEqual(cs.run_checks(nested).errors, [])


class SemverTests(unittest.TestCase):
    def test_precedence_follows_the_spec(self) -> None:
        ordered = [
            "1.0.0-alpha",
            "1.0.0-alpha.1",
            "1.0.0-alpha.beta",
            "1.0.0-beta",
            "1.0.0-beta.2",
            "1.0.0-beta.11",
            "1.0.0-rc.1",
            "1.0.0",
            "1.0.1",
            "1.1.0",
            "2.0.0",
        ]
        self.assertEqual(sorted(reversed(ordered), key=cs.semver_key), ordered)

    def test_validity(self) -> None:
        for good in ("0.0.0", "1.2.3", "1.0.0-rc.1", "1.0.0+build.5", "1.0.0-x.7.z.92"):
            self.assertTrue(cs.is_semver(good), good)
        for bad in ("1.0", "v1.0.0", "01.0.0", "1.0.0-", "1.0.0-01", "1.2.3.4"):
            self.assertFalse(cs.is_semver(bad), bad)

    def test_pep440_mapping(self) -> None:
        self.assertEqual(cs.to_pep440("1.2.0"), "1.2.0")
        self.assertEqual(cs.to_pep440("1.2.0-alpha.1"), "1.2.0a1")
        self.assertEqual(cs.to_pep440("1.2.0-beta.3"), "1.2.0b3")
        self.assertEqual(cs.to_pep440("1.2.0-rc.2"), "1.2.0rc2")
        for unmappable in ("1.2.0-preview.1", "1.2.0+build.1"):
            with self.assertRaises(ValueError):
                cs.to_pep440(unmappable)


if __name__ == "__main__":
    unittest.main()
