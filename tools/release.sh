#!/usr/bin/env bash
# Publishes a release from main: cuts CHANGELOG.md, tags, creates the GitHub release with its text
# and follows the release workflow to its end (docs/releasing.md).
# Run as `task release`. `task release -- v0.13.0` names the version.
# A run cut short runs again from the top.
set -euo pipefail

root=$(git rev-parse --show-toplevel)
cd "$root"

die() { echo "release: $*" >&2; exit 1; }
step() { go -C backend run ./cmd/release -root "$root" "$@"; }

# Prints a line whenever the run or one of its jobs changes state, then the finished run.
# gh run watch redraws on the alternate screen, which keeps no scrollback.
follow() {
  local run=$1 seen="" state
  while :; do
    state=$(gh run view "$run" --json status,jobs --jq '.status + ": " + (.jobs
      | map(.name + " " + (if .status == "completed" then .conclusion else .status end)) | join(", "))')
    [[ $state == "$seen" ]] || echo "$state"
    seen=$state
    [[ $state == completed:* ]] && break
    sleep 10
  done
  GH_PAGER=cat gh run view "$run" --exit-status
}

command -v gh >/dev/null || die "gh is not on PATH: run inside nix develop"
gh auth status >/dev/null 2>&1 || die "gh has no login: run gh auth login or set GH_TOKEN"

[[ $(git branch --show-current) == main ]] || die "a release comes from main"
git diff --quiet HEAD || die "the checkout holds uncommitted changes to tracked files"
git fetch --quiet --tags origin main
git merge-base --is-ancestor origin/main HEAD || die "origin/main holds commits this checkout lacks: pull first"

version=${1:-}
if [[ -z $version ]]; then
  version=$(step next) || die "name the version to finish a release cut already: task release -- vX.Y.Z"
fi
echo "Version: $version"

# The check workflow answers for the last commit holding code.
# A commit touching nothing but the cut's two files, the cut among them, gets its answer from the one before it.
checked=HEAD
while git diff --quiet "$checked^" "$checked" -- . ':!CHANGELOG.md' ':!packaging/nix/release.json'; do
  checked="$checked^"
done
sha=$(git rev-parse "$checked")
git merge-base --is-ancestor "$sha" origin/main || die "push $sha first, so the check workflow can answer for it"

run=$(gh run list --workflow check.yml --commit "$sha" --limit 1 --json databaseId --jq '.[0].databaseId // empty')
[[ -n $run ]] || die "the check workflow holds no run for $sha"
follow "$run" || die "the check workflow failed on $sha: gh run view $run"

read -r -p "Release $version from $(git rev-parse --short HEAD)? [y/N] " confirm
[[ $confirm == y ]] || die "stopped before the cut"

step cut "$version" "$(date -u +%F)"
# The flake reads this to name the version of a build from this source, the tag being out of its reach.
source=$(nix eval --raw .#lib.sourceKey) || die "the flake names no source key"
printf '{\n  "version": "%s",\n  "source": "%s"\n}\n' "${version#v}" "$source" > packaging/nix/release.json
git diff --quiet -- CHANGELOG.md packaging/nix/release.json ||
  git commit --quiet -m "Release $version" -- CHANGELOG.md packaging/nix/release.json

if tagged=$(git rev-parse --verify --quiet "refs/tags/$version^{commit}"); then
  [[ $tagged == $(git rev-parse HEAD) ]] || die "$version tags $tagged, but HEAD is $(git rev-parse HEAD)"
else
  git tag -a "$version" -m "$version"
fi
git push --quiet origin main "refs/tags/$version"

notes=$(mktemp)
trap 'rm -f "$notes"' EXIT
step notes "$version" > "$notes"
if gh release view "$version" >/dev/null 2>&1; then
  gh release edit "$version" --notes-file "$notes"
else
  gh release create "$version" --verify-tag --title "$version" --notes-file "$notes"
fi

# The release workflow starts on the published release, a few seconds after it.
for _ in $(seq 30); do
  run=$(gh run list --workflow release.yml --event release --limit 10 --json databaseId,headBranch \
    --jq "map(select(.headBranch == \"$version\"))[0].databaseId // empty")
  [[ -n $run ]] && break
  sleep 2
done
[[ -n $run ]] || die "no release workflow run started for $version: gh run list --workflow release.yml"
follow "$run" || die "the release workflow failed: gh run view $run --log-failed"

echo "Published $version: $(gh release view "$version" --json url --jq .url)"
echo "Record a test run on each machine with task qa:record -- -version $version"
