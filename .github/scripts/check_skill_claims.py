#!/usr/bin/env python3
"""Check that the smurf skill still describes the code that exists.

The skill drifted twice in a single session, and the failure mode was not
omission. It confidently described a bug that had already been fixed, and a
reader trusts a skill enough to work around a problem that is no longer there.

Behaviour cannot be verified from prose, so this checks the things that can be:
every file it names exists, every flag it mentions is declared somewhere in
cmd/, and every count it states is still true. Those are what go stale first
when code moves.

Deliberately conservative. A false failure here would train people to ignore
the check, which is worse than the drift it exists to catch, so anything
ambiguous is skipped rather than guessed at.
"""

import pathlib
import re
import subprocess
import sys

REPO = pathlib.Path(__file__).resolve().parents[2]
SKILL = REPO / ".claude" / "skills" / "smurf" / "SKILL.md"


def fail(problems: list[str]) -> int:
    print("Skill claims no longer match the code:\n")
    for p in problems:
        print(f"  - {p}")
    print(
        "\nUpdate .claude/skills/smurf/SKILL.md. A skill that describes code "
        "which has moved is worse than one that says nothing, because it is "
        "trusted."
    )
    return 1


def check_paths(text: str) -> list[str]:
    """Every repository path the skill names should exist."""
    problems = []
    pattern = re.compile(r"`([a-zA-Z0-9_./-]+\.(?:go|yml|yaml|sh|py|md))`")
    for match in sorted(set(pattern.findall(text))):
        # Bare filenames without a directory are usually generic references.
        if "/" not in match:
            continue
        if not (REPO / match).exists():
            problems.append(f"file does not exist: {match}")
    return problems


def check_flags(text: str) -> list[str]:
    """Every --flag the skill mentions should be declared under cmd/."""
    problems = []
    declared = subprocess.run(
        ["grep", "-rhoE", r'"[a-z][a-z-]+"', "cmd/"],
        cwd=REPO, capture_output=True, text=True,
    ).stdout
    known = set(re.findall(r'"([a-z][a-z-]+)"', declared))

    # Flags cobra provides itself, which no command declares.
    builtin = {"help", "version"}

    for flag in sorted(set(re.findall(r"`--([a-z][a-z-]+)`", text))):
        if flag in builtin or flag in known:
            continue
        problems.append(f"--{flag} is not declared anywhere in cmd/")
    return problems


def check_counts(text: str) -> list[str]:
    """Counts stated in prose go stale silently."""
    problems = []

    claimed = re.search(r"\*\*(\d+) subcommands\*\*", text)
    if claimed:
        binary = REPO / "smurf-skillcheck"
        build = subprocess.run(
            ["go", "build", "-o", str(binary), "."], cwd=REPO,
            capture_output=True, text=True,
        )
        if build.returncode == 0:
            try:
                actual = count_subcommands(binary)
                if actual != int(claimed.group(1)):
                    problems.append(
                        f"skill says {claimed.group(1)} subcommands, the binary has {actual}"
                    )
            finally:
                binary.unlink(missing_ok=True)
    return problems


def count_subcommands(binary: pathlib.Path) -> int:
    def children(path: list[str]) -> list[str]:
        out = subprocess.run(
            [str(binary), *path, "--help"], capture_output=True, text=True
        ).stdout
        names, collecting = [], False
        for line in out.splitlines():
            if line.strip().startswith("Available Commands"):
                collecting = True
                continue
            if collecting:
                if not line.strip():
                    break
                parts = line.split()
                if parts and not line.startswith("Flags"):
                    names.append(parts[0])
        return [n for n in names if n not in ("help", "completion")]

    total = 0
    stack = [[]]
    while stack:
        path = stack.pop()
        for child in children(path):
            total += 1
            stack.append(path + [child])
    return total


def main() -> int:
    if not SKILL.exists():
        print(f"skill not found at {SKILL}, nothing to check")
        return 0

    text = SKILL.read_text()
    problems = check_paths(text) + check_flags(text) + check_counts(text)

    if problems:
        return fail(problems)

    print("Skill claims check passed: paths, flags and counts all match the code.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
