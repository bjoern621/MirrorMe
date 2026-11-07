#!/usr/bin/env bash
# Writes packaging/nix/release.json for the release that VERSION names (0.14.0), as `task release` runs it after the cut.
# The flake reads it to name the version of a build from this source, the tag being out of its reach.
set -euo pipefail

source=$(nix eval --raw .#lib.sourceKey)
printf '{\n  "version": "%s",\n  "source": "%s"\n}\n' "${VERSION:?task release sets VERSION}" "$source" > packaging/nix/release.json
