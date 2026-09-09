#!/usr/bin/env python3
"""Render Formula/smurf.rb for a released tag.

The tap formula was maintained by hand and silently fell two releases behind:
v1.2.0 and v1.2.1 both published green while `brew install smurf` still served
1.1.9. README and docs/sm/docs/installation.md both tell people to install that
way, so the staleness reached users rather than sitting in a config file.

.goreleaser.yml carries a `brews:` block, but nothing runs it: release.yml never
invokes goreleaser, and `make release-dry-run` passes --skip=publish. This script
is what the release workflow uses instead, so the formula is generated from the
checksums the release actually published rather than retyped.

Usage: render_brew_formula.py <tag> <checksums.txt> <output.rb>
"""

import pathlib
import re
import sys

# on_macos/on_linux blocks in the order the formula declares them, mapped to the
# archive each one installs.
PLATFORMS = [
    ("macos", [("arm", "darwin-arm64"), ("intel", "darwin-amd64")]),
    ("linux", [("intel", "linux-amd64"), ("arm", "linux-arm64")]),
]

# No `version` stanza: brew scans the version out of the URL, and `brew audit`
# fails a formula that also states it ("version X is redundant with version
# scanned from URL"). The tap's own CI runs audit only on changed formulae,
# which is why the hand-written 1.1.9 formula carried one for a year without
# anyone noticing.
TEMPLATE = '''class Smurf < Formula
  desc "CloudNative CI/CD Management Tool"
  homepage "https://github.com/clouddrove/smurf"
  license "Apache-2.0"

{blocks}  def install
    bin.install "smurf"
  end

  test do
    system "#{{bin}}/smurf", "--version"
  end
end
'''

ARCH_BLOCK = '''    on_{arch} do
      url "https://github.com/clouddrove/smurf/releases/download/{tag}/smurf-{tag}-{suffix}.tar.gz"
      sha256 "{sha}"
    end
'''


def parse_checksums(path: pathlib.Path) -> dict[str, str]:
    sums = {}
    for line in path.read_text().splitlines():
        parts = line.split()
        if len(parts) == 2:
            sums[parts[1]] = parts[0]
    return sums


def main() -> int:
    if len(sys.argv) != 4:
        print(__doc__)
        return 2

    tag, checksums_path, out_path = sys.argv[1], sys.argv[2], sys.argv[3]

    if not re.fullmatch(r"v\d+\.\d+\.\d+", tag):
        print(f"refusing to render a formula for {tag!r}: expected vMAJOR.MINOR.PATCH")
        return 1

    sums = parse_checksums(pathlib.Path(checksums_path))

    # Every platform the formula claims to support must be present. A partial
    # release that quietly dropped an architecture would otherwise produce a
    # formula that 404s for whoever runs that architecture.
    rendered, missing = [], []
    for os_name, arches in PLATFORMS:
        arch_blocks = []
        for arch, suffix in arches:
            archive = f"smurf-{tag}-{suffix}.tar.gz"
            if archive not in sums:
                missing.append(archive)
                continue
            arch_blocks.append(
                ARCH_BLOCK.format(arch=arch, tag=tag, suffix=suffix, sha=sums[archive])
            )
        rendered.append(f"  on_{os_name} do\n" + "\n".join(arch_blocks) + "  end\n\n")

    if missing:
        print("checksums.txt is missing archives the formula needs:")
        for m in missing:
            print(f"  - {m}")
        return 1

    pathlib.Path(out_path).write_text(
        TEMPLATE.format(blocks="".join(rendered))
    )
    print(f"rendered {out_path} for {tag}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
