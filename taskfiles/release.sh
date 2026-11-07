#!/usr/bin/env bash
# Releases the next semantic version from the release branch (README.md).
# release.yml passes the settings through the environment.
# A step cut short runs again from the top: every step leaves a state already reached as it is.
set -euo pipefail

TAG_PREFIX=${TAG_PREFIX-}
FORGE=${FORGE:-github}
BRANCH=${BRANCH:-main}
PART=${PART:-minor}
VERSION=${VERSION:-}
NOTES=${NOTES:-}
CHANGELOG=${CHANGELOG:-}
PREPARE=${PREPARE:-}
CI=${CI:-false}
YES=${YES:-false}

semver="^${TAG_PREFIX}[0-9]+[.][0-9]+[.][0-9]+$"
# The heading kinds that a changelog section may group its entries under (keepachangelog.com).
kinds="Added|Changed|Deprecated|Removed|Fixed|Security"

die() { echo "release: $*" >&2; exit 1; }

# Runs a tool from the PATH where a dev shell puts it, from nixpkgs otherwise.
tool() {
  local name=$1
  shift
  if type -P "$name" >/dev/null; then
    command "$name" "$@"
  else
    nix shell "nixpkgs#$name" --command "$name" "$@"
  fi
}
gh() { tool gh "$@"; }
glab() { tool glab "$@"; }
jq() { tool jq "$@"; }

# Prints "<commit> <tag>" per version tag on origin.
# A local tag counts for nothing: a clone missing a tag would offer that tag again.
remote_tags() {
  git ls-remote --tags origin |
    sed -E 's#\^\{\}$##; s#refs/tags/##' |
    awk -v re="$semver" '$2 ~ re { print $1, $2 }'
}

newest() { awk '{ print $2 }' | sort -uV | tail -1; }

# Answers whether version a sorts past version b, an empty b included.
newer() { [[ $1 != "$2" && $(printf '%s\n%s\n' "$2" "$1" | sort -V | tail -1) == "$1" ]]; }

bump() {
  local version=${1#"$TAG_PREFIX"}
  version=${version:-0.0.0}
  local major=${version%%.*} rest=${version#*.}
  local minor=${rest%%.*} patch=${rest#*.}
  case $2 in
    major) echo "$TAG_PREFIX$((major + 1)).0.0" ;;
    minor) echo "$TAG_PREFIX$major.$((minor + 1)).0" ;;
    patch) echo "$TAG_PREFIX$major.$minor.$((patch + 1))" ;;
    *) die "PART takes major, minor or patch, not $2." ;;
  esac
}

# Fails on a changelog without an Unreleased section or with a kind outside $kinds.
changelog_check() {
  [[ -f $CHANGELOG ]] || die "CHANGELOG names $CHANGELOG, which does not exist."
  grep -qx '## Unreleased' "$CHANGELOG" || die "$CHANGELOG holds no \"## Unreleased\" section."
  local unknown
  unknown=$(awk -v re="^($kinds)$" '/^### / { kind = substr($0, 5); sub(/[ \t]+$/, "", kind); if (kind !~ re) print kind }' "$CHANGELOG")
  [[ -z $unknown ]] || die "$CHANGELOG groups entries under $(echo "$unknown" | head -1), outside ${kinds//|/, }."
}

# Prints each kind under Unreleased that holds at least one line of text.
changelog_pending() {
  awk '
    /^## / { inside = ($0 == "## Unreleased"); kind = ""; next }
    inside && /^### / { kind = substr($0, 5); sub(/[ \t]+$/, "", kind); next }
    inside && kind != "" && NF { found[kind] = 1 }
    END { for (k in found) print k }
  ' "$CHANGELOG"
}

# Prints the newest version that the changelog holds a section for.
changelog_newest() {
  awk -v re="$semver" '/^## / && $2 ~ re { print $2; exit }' "$CHANGELOG"
}

# Prints the part that the pending kinds bump, counting up from current.
# A removal takes a major from 1.0.0 on, any kind besides a fix a minor, fixes alone a patch.
changelog_part() {
  local pending major
  pending=$(changelog_pending)
  [[ -n $pending ]] || die "$CHANGELOG holds no entries under Unreleased."
  major=${1#"$TAG_PREFIX"}
  major=${major%%.*}
  if grep -qx Removed <<<"$pending" && ((${major:-0} >= 1)); then
    echo major
  elif grep -qvxE 'Fixed|Security' <<<"$pending"; then
    echo minor
  else
    echo patch
  fi
}

# Prints the body of a version's section without the blank lines around it.
changelog_entries() {
  awk -v version="$1" '
    /^## / { if (inside) exit; inside = ($2 == version); next }
    inside { lines[++n] = $0 }
    END {
      first = 1; while (first <= n && lines[first] ~ /^[ \t]*$/) first++
      last = n; while (last >= first && lines[last] ~ /^[ \t]*$/) last--
      for (i = first; i <= last; i++) print lines[i]
    }
  ' "$CHANGELOG"
}

# Moves the Unreleased entries under a section for the tag. A tag holding its section already stays.
changelog_cut() {
  awk -v re="^## $1( |$)" '$0 ~ re { found = 1 } END { exit !found }' "$CHANGELOG" && return
  local cut
  cut=$(awk -v heading="## $1 ($(date -u +%F))" '{ print } $0 == "## Unreleased" { print ""; print heading }' "$CHANGELOG")
  printf '%s\n' "$cut" >"$CHANGELOG"
}

# Prints the tag that a release from HEAD takes.
# A rerun finishes the release that HEAD already carries: a tag on origin, or a changelog section cut
# but never tagged.
next_tag() {
  local tags current tagged cut
  tags=$(remote_tags)
  if [[ -n $VERSION ]]; then
    [[ $VERSION =~ $semver ]] || die "VERSION takes ${TAG_PREFIX}X.Y.Z, not $VERSION."
    echo "$VERSION"
    return
  fi
  tagged=$(awk -v head="$(git rev-parse HEAD)" '$1 == head' <<<"$tags" | newest)
  if [[ -n $tagged ]]; then
    echo "$tagged"
    return
  fi
  current=$(newest <<<"$tags")
  if [[ -n $CHANGELOG ]]; then
    changelog_check
    cut=$(changelog_newest)
    if [[ -n $cut ]] && ! awk '{ print $2 }' <<<"$tags" | grep -qxF "$cut" && newer "$cut" "$current"; then
      echo "$cut"
      return
    fi
    bump "$current" "$(changelog_part "$current")"
    return
  fi
  bump "$current" "$PART"
}

# Fails unless HEAD can take the release.
# Commits past origin pass alone as the release commit of a run cut short.
preflight() {
  local tag=$1 commit
  [[ $FORGE == github || $FORGE == tag ]] || die "FORGE takes github or tag, not $FORGE."
  [[ $(git branch --show-current) == "$BRANCH" ]] || die "A release runs from $BRANCH. Switch to $BRANCH, then release."
  [[ -z $(git status --porcelain) ]] || die "The working tree carries changes. Commit them, then release."
  git fetch --quiet origin "$BRANCH"
  git merge-base --is-ancestor "origin/$BRANCH" HEAD || die "$BRANCH lacks commits from origin. Pull, then release."
  for commit in $(git rev-list "origin/$BRANCH..HEAD"); do
    [[ $(git log -1 --format=%s "$commit") == "Release $tag" ]] ||
      die "$BRANCH holds commits that origin lacks. Push them, then release."
  done
  [[ -z $NOTES || -s $NOTES ]] || die "NOTES names $NOTES, which is missing or empty."
  [[ -z $CHANGELOG ]] || changelog_check
}

github_token() {
  [[ -n ${GH_TOKEN:-} ]] && return
  gh auth status >/dev/null 2>&1 && return
  GH_TOKEN=$(printf 'protocol=https\nhost=github.com\n\n' | git credential fill | sed -n 's/^password=//p')
  export GH_TOKEN
}

# Prints the CI state of a commit (sha) or a tag (tag) as five fields split by \x1f:
# none, finished, ok, a line naming each run or job with its state, and the links to what failed.
ci_state() {
  local kind=$1 ref=$2
  case $(git remote get-url origin) in
    *github.com[:/]*)
      local filter=--commit
      [[ $kind == tag ]] && filter=--branch
      gh run list "$filter" "$ref" --limit 50 --json name,status,conclusion,url --jq '
        def passed: .conclusion | IN("success", "skipped", "neutral");
        [ (length == 0), all(.status == "completed"), all(passed),
          (map(.name + " " + (if .status == "completed" then .conclusion else .status end)) | join(", ")),
          (map(select(.status == "completed" and (passed | not)) | .url) | join(" ")) ]
        | map(tostring) | join("\u001f")'
      ;;
    *)
      local query="ref=$ref" pipeline jobs
      [[ $kind == sha ]] && query="sha=$ref&ref=$BRANCH"
      pipeline=$(glab api "projects/:id/pipelines?$query&per_page=1" | jq -c '.[0] // empty')
      if [[ -z $pipeline ]]; then
        printf 'true\x1ffalse\x1ffalse\x1f\x1f\n'
        return
      fi
      jobs=$(glab api "projects/:id/pipelines/$(jq -r .id <<<"$pipeline")/jobs?per_page=100" |
        jq -r 'map(.name + " " + .status) | join(", ")')
      jq -r --arg jobs "$jobs" '
        [ false, (.status | IN("success", "failed", "canceled", "skipped", "manual")), (.status == "success"),
          ("pipeline " + .status + (if $jobs == "" then "" else ": " + $jobs end)),
          (if .status == "success" then "" else .web_url end) ]
        | map(tostring) | join("\u001f")' <<<"$pipeline"
      ;;
  esac
}

# Prints a line whenever a run or job changes state, until CI on the ref has finished.
# Fails unless everything passed.
# A run that starts after the others finished counts as long as it shows up by the next check.
follow_ci() {
  local name=$1 kind=$2 ref=$3 seen="" waited=0 settled=0 none finished ok line urls
  while :; do
    if ! IFS=$'\x1f' read -r none finished ok line urls < <(ci_state "$kind" "$ref") || [[ -z $none ]]; then
      echo "release: CI state of $name unreadable, asking again" >&2
      sleep 10
      continue
    fi
    if [[ $none == true ]]; then
      ((waited < 120)) || die "No CI run started for $name."
      sleep 5
      waited=$((waited + 5))
      continue
    fi
    [[ $line == "$seen" ]] || echo "$name: $line"
    seen=$line
    if [[ $finished == true ]]; then
      ((++settled < 2)) || break
    else
      settled=0
    fi
    sleep 10
  done
  [[ $ok == true ]] || die "CI failed for $name: $urls"
}

publish_github() {
  local tag=$1 notes
  gh release view "$tag" >/dev/null 2>&1 && return
  local args=(--verify-tag --title "$tag")
  if [[ -n $NOTES ]]; then
    args+=(--notes-file "$NOTES")
  elif [[ -n $CHANGELOG ]]; then
    notes=$(mktemp)
    trap 'rm -f "$notes"' EXIT
    changelog_entries "$tag" >"$notes"
    args+=(--notes-file "$notes" --generate-notes)
  else
    args+=(--generate-notes)
  fi
  gh release create "$tag" "${args[@]}" >/dev/null
}

release() {
  local tag confirm
  tag=$(next_tag)
  preflight "$tag"
  if [[ $YES != true ]]; then
    read -r -p "Release $tag from \"$(git log -1 --format=%s)\"? [y/N] " confirm
    [[ $confirm == [yY]* ]] || die "Stopped before anything changed."
  fi
  [[ $FORGE == github ]] && github_token

  if [[ $CI == true ]]; then
    follow_ci "$BRANCH at $(git rev-parse --short "origin/$BRANCH")" sha "$(git rev-parse "origin/$BRANCH")"
  fi

  [[ -z $CHANGELOG ]] || changelog_cut "$tag"
  [[ -z $PREPARE ]] || TAG=$tag VERSION=${tag#"$TAG_PREFIX"} sh -c "$PREPARE"
  git add --update
  git diff --cached --quiet || git commit --quiet -m "Release $tag"

  if git rev-parse --quiet --verify "refs/tags/$tag" >/dev/null; then
    [[ $(git rev-parse "$tag^{commit}") == $(git rev-parse HEAD) ]] ||
      die "$tag tags $(git rev-parse --short "$tag^{commit}"), not HEAD. Delete the local tag, then release."
  else
    git tag -a "$tag" -m "$tag"
  fi
  if [[ $(git rev-parse HEAD) == $(git rev-parse "origin/$BRANCH") ]]; then
    git push --quiet origin "refs/tags/$tag"
  else
    git push --quiet --atomic origin "HEAD:refs/heads/$BRANCH" "refs/tags/$tag"
  fi
  [[ $FORGE == github ]] && publish_github "$tag"

  [[ $CI == true ]] && follow_ci "$tag" tag "$tag"
  echo "Published $tag."
}

case ${1:-} in
  version)
    current=$(remote_tags | newest)
    next=$(next_tag)
    echo "published ${current:-nothing}, next $next"
    ;;
  notes)
    current=$(remote_tags | newest)
    [[ -n $current ]] || die "No release exists yet."
    [[ $FORGE == github ]] || die "FORGE=$FORGE keeps no release notes on the forge."
    github_token
    gh release view "$current" --json body --jq .body
    ;;
  release) release ;;
  *) die "name a step: version, notes or release" ;;
esac
