#!/usr/bin/env python3
"""Consistency checks for the agentic coding scaffold.

Verifies gate ordering, acceptance criteria, context budgets, documentation sprawl,
version agreement, and release tags. Standard library only; runs on Python 3.9 or later.

Usage:
    python3 scripts/check_scaffold.py [--root PATH] [--quiet]

Exit code 0 when there are no errors, 1 otherwise. Warnings do not fail the check.
"""

from __future__ import annotations

import argparse
import datetime
import json
import re
import shutil
import subprocess
import sys
import urllib.parse
from pathlib import Path

# Official pattern from https://semver.org/spec/v2.0.0.html
SEMVER_RE = re.compile(
    r"^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)"
    r"(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)"
    r"(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?"
    r"(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$"
)

LEDGER_PREFIX_RE = re.compile(r"^(G\d+|R):")
LEDGER_RE = re.compile(
    r"^(?P<key>G(?P<num>\d+)|R):\s+(?P<state>APPROVED|REJECTED)\s+by\s+(?P<who>.+?),\s+"
    r"(?P<date>\d{4}-\d{2}-\d{2})\."
    r"(?:\s+Version:\s+(?P<version>\S+))?"
    r"(?:\s+Notes:.*)?\s*$"
)

SPRAWL_RE = re.compile(
    r"(summary|report|progress|handoff|validation|results|findings|session"
    r"|phase[-_ ]?\d|gate[-_ ]?\d)",
    re.IGNORECASE,
)

PLACEHOLDER_RE = re.compile(r"<(PROJECT_NAME|OWNER|REPO|LICENSE|ONE_PARAGRAPH)\b[^>]*>")

DEFAULT_CONFIG = {
    "limits": {
        "agents_md_max_lines": 150,
        "agents_md_max_bytes": 24000,
        "claude_md_max_lines": 40,
        "status_md_max_lines": 80,
    },
    "root_markdown_allowlist": [
        "AGENTS.md",
        "CLAUDE.md",
        "README.md",
        "SCAFFOLD.md",
        "STATUS.md",
        "GATES.md",
        "CHANGELOG.md",
        "CONTRIBUTING.md",
        "LICENSE.md",
        "SECURITY.md",
        "CODE_OF_CONDUCT.md",
    ],
    "sprawl_scan_dirs": ["docs"],
    "sprawl_exempt_dirs": ["docs/archive"],
    "sprawl_allow": [],
    "forbidden_imports": ["STATUS.md", "GATES.md", "CHANGELOG.md", "docs/archive", "artifacts"],
    "version_mirrors": [],
    "readme_version_badge": True,
    "readme_phase_badge": True,
    "check_tags": True,
}


class Report:
    """Collects check results."""

    def __init__(self) -> None:
        self.errors: list[str] = []
        self.warnings: list[str] = []
        self.passed: list[str] = []

    def error(self, message: str) -> None:
        self.errors.append(message)

    def warn(self, message: str) -> None:
        self.warnings.append(message)

    def ok(self, message: str) -> None:
        self.passed.append(message)


def read_text(path: Path) -> str | None:
    """Return the file's text, or None if it does not exist."""
    try:
        return path.read_text(encoding="utf-8")
    except FileNotFoundError:
        return None


def lines_outside_fences(text: str) -> list[str]:
    """Return the lines of a Markdown text that are not inside code fences."""
    out: list[str] = []
    fenced = False
    for line in text.splitlines():
        if line.lstrip().startswith("```"):
            fenced = not fenced
            continue
        if not fenced:
            out.append(line)
    return out


def load_config(root: Path, report: Report) -> dict:
    """Load .scaffold.json over the defaults."""
    config = json.loads(json.dumps(DEFAULT_CONFIG))
    text = read_text(root / ".scaffold.json")
    if text is None:
        report.warn(".scaffold.json not found; using defaults")
        return config
    try:
        user = json.loads(text)
    except json.JSONDecodeError as exc:
        report.error(f".scaffold.json is not valid JSON: {exc}")
        return config
    for key, value in user.items():
        if key == "limits" and isinstance(value, dict):
            config["limits"].update(value)
        else:
            config[key] = value
    return config


# --- Semantic Versioning -------------------------------------------------------------


def is_semver(value: str) -> bool:
    return SEMVER_RE.match(value) is not None


def semver_key(value: str) -> tuple:
    """Return a sort key implementing SemVer 2.0.0 precedence (build metadata ignored)."""
    match = SEMVER_RE.match(value)
    if match is None:
        raise ValueError(f"not a semantic version: {value}")
    major, minor, patch, pre, _build = match.groups()
    if pre is None:
        return (int(major), int(minor), int(patch), 1, [])
    identifiers = []
    for part in pre.split("."):
        identifiers.append((0, int(part), "") if part.isdigit() else (1, 0, part))
    return (int(major), int(minor), int(patch), 0, identifiers)


def to_pep440(value: str) -> str:
    """Convert a SemVer string to its PEP 440 form, or raise ValueError."""
    match = SEMVER_RE.match(value)
    if match is None:
        raise ValueError(f"not a semantic version: {value}")
    major, minor, patch, pre, build = match.groups()
    core = f"{major}.{minor}.{patch}"
    if build:
        raise ValueError("build metadata has no PEP 440 equivalent in this scaffold")
    if pre is None:
        return core
    pre_match = re.fullmatch(r"(alpha|beta|rc)\.(0|[1-9]\d*)", pre)
    if pre_match is None:
        raise ValueError(
            f"pre-release '{pre}' cannot be mapped to PEP 440; use alpha.N, beta.N, or rc.N"
        )
    label = {"alpha": "a", "beta": "b", "rc": "rc"}[pre_match.group(1)]
    return f"{core}{label}{pre_match.group(2)}"


# --- Parsers -------------------------------------------------------------------------


def parse_phase(status_text: str) -> int | None:
    for line in lines_outside_fences(status_text):
        match = re.match(r"^Phase:\s*(\d+)\s*$", line)
        if match:
            return int(match.group(1))
    return None


def parse_plan_phases(plan_text: str) -> set[int]:
    phases = set()
    for line in lines_outside_fences(plan_text):
        match = re.match(r"^## Phase (\d+)\b", line)
        if match:
            phases.add(int(match.group(1)))
    return phases


def parse_ledger(gates_text: str, report: Report) -> tuple[dict[int, dict], list[str]]:
    """Parse GATES.md.

    Returns:
        A mapping of gate number to its latest APPROVED entry (gates whose latest entry
        is REJECTED are absent), and the approved versions in ledger order.
    """
    latest: dict[int, dict] = {}
    versions: list[str] = []
    for line in lines_outside_fences(gates_text):
        line = line.strip()
        if not LEDGER_PREFIX_RE.match(line):
            continue
        match = LEDGER_RE.match(line)
        if match is None:
            report.error(f"GATES.md: malformed ledger line: {line!r}")
            continue
        entry = match.groupdict()
        try:
            datetime.date.fromisoformat(entry["date"])
        except ValueError:
            report.error(f"GATES.md: invalid date in line: {line!r}")
            continue
        version = entry["version"].rstrip(".") if entry["version"] else None
        entry["version"] = version
        if entry["state"] == "APPROVED":
            if version is None:
                report.error(
                    f"GATES.md: APPROVED line needs 'Version: X.Y.Z' or 'Version: none': {line!r}"
                )
                continue
            if version != "none":
                if not is_semver(version):
                    report.error(f"GATES.md: '{version}' is not a valid SemVer 2.0.0 version")
                    continue
                versions.append(version)
            elif entry["key"] == "R":
                report.error("GATES.md: an 'R:' release line must carry a version")
                continue
        if entry["num"] is not None:
            number = int(entry["num"])
            if entry["state"] == "APPROVED":
                latest[number] = entry
            else:
                latest.pop(number, None)
    return latest, versions


def badge_message(readme_text: str, label: str) -> str | None:
    """Return the decoded message of a static Shields badge with the given label."""
    match = re.search(
        r"img\.shields\.io/badge/" + re.escape(label) + r"-([^)\s?\"']+)", readme_text
    )
    if match is None:
        return None
    raw = match.group(1).replace("--", "\x00")
    if "-" not in raw:
        return None
    message = raw.rsplit("-", 1)[0]
    message = message.replace("\x00", "-").replace("__", "\x01").replace("_", " ")
    return urllib.parse.unquote(message.replace("\x01", "_"))


def pyproject_version(text: str) -> str | None:
    table = re.search(r"(?ms)^\[project\]\s*$(.*?)(?=^\[|\Z)", text)
    if table is None:
        return None
    match = re.search(r"(?m)^version\s*=\s*[\"']([^\"']+)[\"']", table.group(1))
    return match.group(1) if match else None


# --- Checks --------------------------------------------------------------------------


def check_gates(root: Path, report: Report) -> tuple[int | None, list[str]]:
    """Check phase ordering against the gate ledger. Returns (phase, approved versions)."""
    status_text = read_text(root / "STATUS.md")
    gates_text = read_text(root / "GATES.md")
    plan_text = read_text(root / "docs" / "PLAN.md")
    if status_text is None:
        report.error("STATUS.md is missing")
    if gates_text is None:
        report.error("GATES.md is missing")
    if plan_text is None:
        report.error("docs/PLAN.md is missing")
    if status_text is None or gates_text is None or plan_text is None:
        return None, []

    phase = parse_phase(status_text)
    if phase is None:
        report.error("STATUS.md: no 'Phase: <number>' line found")
    approved, versions = parse_ledger(gates_text, report)
    if phase is None:
        return None, versions

    if phase not in parse_plan_phases(plan_text):
        report.error(f"docs/PLAN.md has no '## Phase {phase}' section for the current phase")

    missing = [n for n in range(phase) if n not in approved]
    if missing:
        names = ", ".join(f"G{n}" for n in missing)
        report.error(
            f"STATUS.md is at phase {phase} but {names} not APPROVED in GATES.md. "
            "A phase may not start before the previous gate is approved by a human."
        )
    else:
        report.ok(f"gates: phase {phase}; gates before it are approved")

    if phase in approved:
        report.warn(
            f"G{phase} is approved but STATUS.md is still at phase {phase}. "
            "Run the gate close routine in docs/process/gates.md."
        )

    check_acceptance(root, report, set(approved))
    return phase, versions


def check_acceptance(root: Path, report: Report, approved_gates: set[int]) -> None:
    path = root / "docs" / "acceptance.json"
    text = read_text(path)
    if text is None:
        report.error("docs/acceptance.json is missing")
        return
    try:
        data = json.loads(text)
    except json.JSONDecodeError as exc:
        report.error(f"docs/acceptance.json is not valid JSON: {exc}")
        return
    criteria = data.get("criteria") if isinstance(data, dict) else None
    if not isinstance(criteria, list):
        report.error("docs/acceptance.json: expected an object with a 'criteria' list")
        return

    seen: set[str] = set()
    problems = 0
    for index, item in enumerate(criteria):
        where = f"docs/acceptance.json criteria[{index}]"
        if not isinstance(item, dict):
            report.error(f"{where}: expected an object")
            problems += 1
            continue
        ident = item.get("id")
        if not isinstance(ident, str) or not ident:
            report.error(f"{where}: 'id' must be a non-empty string")
            problems += 1
            continue
        if ident in seen:
            report.error(f"docs/acceptance.json: duplicate id '{ident}'")
            problems += 1
        seen.add(ident)
        if type(item.get("phase")) is not int:
            report.error(f"{ident}: 'phase' must be an integer")
            problems += 1
            continue
        if not isinstance(item.get("description"), str) or not item["description"].strip():
            report.error(f"{ident}: 'description' must be a non-empty string")
            problems += 1
        if type(item.get("passes")) is not bool:
            report.error(f"{ident}: 'passes' must be true or false")
            problems += 1
            continue
        evidence = item.get("evidence", "")
        if item["passes"] and not (isinstance(evidence, str) and evidence.strip()):
            report.error(f"{ident}: passes is true but 'evidence' is empty")
            problems += 1
        if item["phase"] in approved_gates and not item["passes"]:
            report.error(
                f"{ident}: G{item['phase']} is approved but this criterion does not pass"
            )
            problems += 1
    if problems == 0:
        report.ok(f"acceptance: {len(criteria)} criteria well-formed")


def check_version(root: Path, config: dict, report: Report, approved_versions: list[str]) -> None:
    version_text = read_text(root / "VERSION")
    if version_text is None:
        report.error("VERSION is missing")
        return
    version = version_text.strip()
    if not is_semver(version):
        report.error(f"VERSION '{version}' is not a valid SemVer 2.0.0 version")
        return

    before = len(report.errors)

    for earlier, later in zip(approved_versions, approved_versions[1:]):
        if semver_key(later) <= semver_key(earlier):
            report.error(
                f"GATES.md: approved version {later} does not come after {earlier}"
            )

    expected = approved_versions[-1] if approved_versions else "0.0.0"
    if version != expected:
        if approved_versions:
            report.error(
                f"VERSION is {version} but the last approved version in GATES.md is "
                f"{expected}. Versions change only through an approved gate or release."
            )
        else:
            report.error(
                f"VERSION is {version} but GATES.md has no approved version; expected 0.0.0"
            )

    changelog = read_text(root / "CHANGELOG.md")
    if changelog is None:
        report.error("CHANGELOG.md is missing")
    else:
        lines = lines_outside_fences(changelog)
        if not any(re.match(r"^## \[Unreleased\]", line, re.IGNORECASE) for line in lines):
            report.error("CHANGELOG.md has no '## [Unreleased]' section")
        releases = []
        for line in lines:
            match = re.match(r"^## \[([^\]]+)\](?:\s+-\s+(\S+))?", line)
            if match and match.group(1).lower() != "unreleased":
                releases.append((match.group(1), match.group(2)))
        for number, date in releases:
            if not is_semver(number):
                report.error(f"CHANGELOG.md: '[{number}]' is not a valid SemVer version")
            try:
                datetime.date.fromisoformat(date or "")
            except ValueError:
                report.error(f"CHANGELOG.md: release [{number}] needs an ISO date (YYYY-MM-DD)")
        if version == "0.0.0":
            if releases:
                report.error("CHANGELOG.md lists releases but VERSION is 0.0.0")
        elif not releases:
            report.error(f"CHANGELOG.md has no release heading for VERSION {version}")
        elif releases[0][0] != version:
            report.error(
                f"CHANGELOG.md newest release is [{releases[0][0]}] but VERSION is {version}"
            )

    readme = read_text(root / "README.md")
    if readme is None:
        report.error("README.md is missing")
    elif config.get("readme_version_badge", True):
        badge = badge_message(readme, "version")
        if badge is None:
            report.error("README.md has no static Shields version badge")
        elif badge != version:
            report.error(f"README.md version badge shows {badge} but VERSION is {version}")

    for mirror in config.get("version_mirrors", []):
        check_mirror(root, mirror, version, report)

    if len(report.errors) == before:
        report.ok(f"version: {version} is consistent")


def git(root: Path, *args: str) -> str | None:
    """Run git in root. Returns stdout, or None when git fails."""
    try:
        result = subprocess.run(
            ["git", "-C", str(root), *args],
            capture_output=True,
            text=True,
            check=False,
        )
    except OSError:
        return None
    return result.stdout if result.returncode == 0 else None


def check_tags(root: Path, config: dict, report: Report, approved_versions: list[str]) -> None:
    """Check that every vX.Y.Z tag is an approved version on a matching release commit."""
    if not config.get("check_tags", True):
        report.ok("tags: check disabled in .scaffold.json")
        return
    if shutil.which("git") is None:
        report.ok("tags: git not available; skipped")
        return
    toplevel = git(root, "rev-parse", "--show-toplevel")
    if toplevel is None or Path(toplevel.strip()).resolve() != root.resolve():
        report.ok("tags: not the top of a git work tree; skipped")
        return
    listing = git(root, "tag", "--list", "v*")
    if listing is None:
        report.error("tags: 'git tag --list' failed")
        return

    approved = set(approved_versions)
    tagged: set[str] = set()
    before = len(report.errors)
    for tag in listing.split():
        version = tag[1:]
        if not is_semver(version):
            continue
        tagged.add(version)
        if version not in approved:
            report.error(
                f"tag {tag} names a version that is not APPROVED in GATES.md. "
                "Tags are created only for approved releases."
            )
            continue
        at_tag = git(root, "show", f"refs/tags/{tag}^{{commit}}:VERSION")
        if at_tag is not None and at_tag.strip() != version:
            report.error(
                f"tag {tag} points at a commit whose VERSION is {at_tag.strip()}; "
                "tag the release commit"
            )

    for version in approved_versions:
        if version != "0.0.0" and version not in tagged:
            report.warn(
                f"approved version {version} has no tag v{version} here. If it was released, "
                "push or fetch its tag (see docs/process/gates.md)."
            )

    if len(report.errors) == before:
        report.ok(f"tags: {len(tagged)} release tag(s) consistent with GATES.md")


def check_mirror(root: Path, mirror: dict, version: str, report: Report) -> None:
    rel = mirror.get("path", "")
    kind = mirror.get("kind", "")
    text = read_text(root / rel)
    if text is None:
        report.error(f"version mirror {rel} is missing")
        return

    expected = version
    found: str | None = None
    if kind in ("pyproject", "python_dunder"):
        try:
            expected = to_pep440(version)
        except ValueError as exc:
            report.error(f"VERSION {version}: {exc}")
            return
        if kind == "pyproject":
            found = pyproject_version(text)
        else:
            match = re.search(r"(?m)^__version__\s*=\s*[\"']([^\"']+)[\"']", text)
            found = match.group(1) if match else None
    elif kind == "package_json":
        try:
            found = json.loads(text).get("version")
        except json.JSONDecodeError as exc:
            report.error(f"{rel} is not valid JSON: {exc}")
            return
    elif kind == "regex":
        try:
            match = re.search(mirror.get("pattern", ""), text, re.MULTILINE)
        except re.error as exc:
            report.error(f"version mirror {rel}: bad pattern: {exc}")
            return
        found = match.group(1) if match and match.groups() else None
    else:
        report.error(f"version mirror {rel}: unknown kind '{kind}'")
        return

    if found is None:
        report.error(f"version mirror {rel}: no version found")
    elif found != expected:
        report.error(f"version mirror {rel} has {found}; expected {expected}")


def check_readme_phase(root: Path, config: dict, report: Report, phase: int | None) -> None:
    readme = read_text(root / "README.md")
    if readme is None or phase is None:
        return
    if config.get("readme_phase_badge", True):
        badge = badge_message(readme, "phase")
        number = re.match(r"\s*(\d+)", badge) if badge else None
        if number is None:
            report.error("README.md has no static Shields phase badge starting with a number")
        elif int(number.group(1)) != phase:
            report.error(
                f"README.md phase badge shows {number.group(1)} but STATUS.md is at phase {phase}"
            )
    if phase >= 1:
        left = sorted(set(PLACEHOLDER_RE.findall(readme)))
        if left:
            report.warn("README.md still contains placeholders: " + ", ".join(left))


def check_context(root: Path, config: dict, report: Report) -> None:
    limits = config["limits"]
    before = len(report.errors)

    agents = read_text(root / "AGENTS.md")
    if agents is None:
        report.error("AGENTS.md is missing")
    else:
        count = len(agents.splitlines())
        size = len(agents.encode("utf-8"))
        if count > limits["agents_md_max_lines"]:
            report.error(
                f"AGENTS.md has {count} lines (limit {limits['agents_md_max_lines']}). "
                "Move detail to docs/process/ or a profile."
            )
        if size > limits["agents_md_max_bytes"]:
            report.error(f"AGENTS.md is {size} bytes (limit {limits['agents_md_max_bytes']})")

    status = read_text(root / "STATUS.md")
    if status is not None:
        count = len(status.splitlines())
        if count > limits["status_md_max_lines"]:
            report.error(
                f"STATUS.md has {count} lines (limit {limits['status_md_max_lines']}). "
                "Rewrite it in place; move finished detail to docs/archive/."
            )

    claude = read_text(root / "CLAUDE.md")
    if claude is not None:
        count = len(claude.splitlines())
        if count > limits["claude_md_max_lines"]:
            report.error(f"CLAUDE.md has {count} lines (limit {limits['claude_md_max_lines']})")
        forbidden = config.get("forbidden_imports", [])
        for line in lines_outside_fences(claude):
            line = re.sub(r"`[^`]*`", "", line)
            for target in re.findall(r"(?<![\w@])@([\w./~-]+)", line):
                clean = target.lstrip("./")
                if any(clean == f or clean.startswith(f.rstrip("/") + "/") for f in forbidden):
                    report.error(
                        f"CLAUDE.md imports @{target}. Imports load in every session; "
                        "import only AGENTS.md."
                    )

    if len(report.errors) == before:
        report.ok("context: always-loaded files are within budget")


def check_sprawl(root: Path, config: dict, report: Report) -> None:
    before = len(report.errors)
    allow_root = set(config.get("root_markdown_allowlist", []))
    allow = set(config.get("sprawl_allow", []))

    for path in sorted(root.glob("*.md")):
        if path.name not in allow_root and path.name not in allow:
            report.error(
                f"{path.name}: Markdown file in the repository root is not on the allowlist. "
                "Put the information in the file that owns it (docs/process/context.md), "
                "or add the file to root_markdown_allowlist in .scaffold.json."
            )

    exempt = [Path(p) for p in config.get("sprawl_exempt_dirs", [])]
    for directory in config.get("sprawl_scan_dirs", []):
        base = root / directory
        if not base.is_dir():
            continue
        for path in sorted(base.rglob("*.md")):
            rel = path.relative_to(root)
            if any(ex == rel or ex in rel.parents for ex in exempt):
                continue
            if rel.as_posix() in allow:
                continue
            if SPRAWL_RE.search(path.stem):
                report.error(
                    f"{rel.as_posix()}: looks like a generated report or progress file. "
                    "Fold it into the file that owns that information, move it to "
                    "docs/archive/, or add it to sprawl_allow in .scaffold.json."
                )

    if len(report.errors) == before:
        report.ok("docs: no stray report or summary files")


def run_checks(root: Path) -> Report:
    """Run every check against the repository at root."""
    report = Report()
    config = load_config(root, report)
    phase, approved_versions = check_gates(root, report)
    check_version(root, config, report, approved_versions)
    check_tags(root, config, report, approved_versions)
    check_readme_phase(root, config, report, phase)
    check_context(root, config, report)
    check_sprawl(root, config, report)
    return report


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument(
        "--root",
        type=Path,
        default=Path(__file__).resolve().parent.parent,
        help="repository root (default: the parent of this script's directory)",
    )
    parser.add_argument("--quiet", action="store_true", help="print only errors and warnings")
    args = parser.parse_args(argv)

    report = run_checks(args.root)
    if not args.quiet:
        for message in report.passed:
            print(f"ok     {message}")
    for message in report.warnings:
        print(f"warn   {message}")
    for message in report.errors:
        print(f"ERROR  {message}")
    print(f"\n{len(report.errors)} error(s), {len(report.warnings)} warning(s)")
    return 1 if report.errors else 0


if __name__ == "__main__":
    sys.exit(main())
