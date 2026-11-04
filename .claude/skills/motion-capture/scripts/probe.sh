#!/usr/bin/env bash
# Writes a throwaway probe project against an archive of HEAD. Run from the repository root.
# Usage: probe.sh <scratch>
set -eu

scratch=$1
here=$(dirname "$0")

mkdir -p "$scratch/tree" "$scratch/probe"
git archive HEAD avalonia api | tar -x -C "$scratch/tree"

# The test fakes stay internal in the repository.
echo '[assembly: System.Runtime.CompilerServices.InternalsVisibleTo("probe")]' \
  >"$scratch/tree/avalonia/ScreenShare.App.Tests/ProbeAccess.cs"

cp "$here/../probe/probe.csproj" "$here/../probe/Program.cs" "$scratch/probe/"
echo "$scratch/probe"
