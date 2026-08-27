#!/usr/bin/env python3
"""Fail the build on govulncheck findings whose vulnerable symbols are reachable.

govulncheck reports at three levels of confidence. Only the strongest one,
where a vulnerable *symbol* appears in the call graph, is treated as a gate.
Findings where the module is merely required, or the package merely imported,
are reported but do not fail: acting on those means chasing code that is never
executed.

Reads the concatenated-JSON stream produced by `govulncheck -format json`.
Each finding looks like:

    {"finding": {"osv": "GO-...", "trace": [{"module": ..., "package": ...,
                                             "function": ...}]}}

A trace whose first frame carries "function" is symbol-level.

Exit codes: 0 clean or fully allowlisted, 1 gate failed, 2 bad input.
"""

import json
import os
import sys

# Reachable findings that are accepted rather than fixed. Every entry needs a
# reason and should be re-checked when the dependency is upgraded.
#
# GO-2026-4887  Moby AuthZ plugin bypass via oversized request bodies
# GO-2026-4883  Moby off-by-one in plugin privilege validation
#
# Both are daemon-side. smurf is a pure Docker API client: it imports only
# client, api/types* and the jsonmessage struct, never daemon/, builder/ or
# pkg/archive, and never creates or execs a container. The Go vulnerability
# database carries no symbol list for these advisories, so govulncheck treats
# every symbol in the affected packages as vulnerable and every ordinary client
# call (ImageBuild, ImagePush, ServerVersion) shows up as a trace. Neither has a
# fixed version published, so there is nothing to upgrade to.
ALLOWLIST = {
    "GO-2026-4887",
    "GO-2026-4883",
}


def parse_stream(text):
    """Yield the objects in a concatenated-JSON document."""
    decoder = json.JSONDecoder()
    i, n = 0, len(text)
    while i < n:
        while i < n and text[i] in " \n\r\t":
            i += 1
        if i >= n:
            return
        obj, i = decoder.raw_decode(text, i)
        yield obj


def classify(findings):
    """Map each OSV id to its strongest evidence level."""
    rank = {"module": 0, "package": 1, "symbol": 2}
    worst = {}
    for finding in findings:
        trace = finding.get("trace") or [{}]
        frame = trace[0]
        if frame.get("function"):
            level = "symbol"
        elif frame.get("package"):
            level = "package"
        else:
            level = "module"
        osv = finding.get("osv")
        if osv and rank[level] > rank.get(worst.get(osv, "module"), -1):
            worst[osv] = level
    return worst


def main():
    raw = sys.stdin.read()
    if not raw.strip():
        print("govulncheck produced no output", file=sys.stderr)
        return 2

    try:
        objects = list(parse_stream(raw))
    except ValueError as exc:
        print(f"could not parse govulncheck JSON: {exc}", file=sys.stderr)
        return 2

    findings = [o["finding"] for o in objects if "finding" in o]
    summaries = {
        o["osv"]["id"]: o["osv"].get("summary", "").strip()
        for o in objects
        if "osv" in o and "id" in o["osv"]
    }

    levels = classify(findings)
    reachable = sorted(k for k, v in levels.items() if v == "symbol")
    blocking = [k for k in reachable if k not in ALLOWLIST]
    allowed = [k for k in reachable if k in ALLOWLIST]
    lower = sorted(k for k, v in levels.items() if v != "symbol")

    def describe(osv):
        text = summaries.get(osv, "")
        return f"{osv}  {text[:100]}" if text else osv

    lines = ["### govulncheck", ""]
    lines.append(
        f"{len(blocking)} blocking, {len(allowed)} allowlisted, "
        f"{len(lower)} lower-confidence"
    )

    if blocking:
        lines += ["", "**Blocking (vulnerable symbol is reachable):**", ""]
        lines += [f"- {describe(k)}" for k in blocking]
    if allowed:
        lines += ["", "Allowlisted (see .github/scripts/govulncheck_gate.py):", ""]
        lines += [f"- {describe(k)}" for k in allowed]
    if lower:
        lines += ["", "Lower confidence (package imported or module required "
                      "only, no reachable symbol):", ""]
        lines += [f"- {describe(k)}" for k in lower]

    report = "\n".join(lines)
    print(report)

    summary_path = os.environ.get("GITHUB_STEP_SUMMARY")
    if summary_path:
        with open(summary_path, "a") as fh:
            fh.write(report + "\n")

    if blocking:
        print(
            f"\nFAIL: {len(blocking)} reachable vulnerabilit"
            f"{'y' if len(blocking) == 1 else 'ies'} with no allowlist entry.",
            file=sys.stderr,
        )
        return 1

    print("\nOK: no unallowlisted reachable vulnerabilities.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
