#!/usr/bin/env bash
# Releases the next semantic version from the release branch (README.md).
# release.yml passes the settings through the environment.
# A step cut short runs again from the top: every step leaves a state already reached as it is.
set -euo pipefail
shopt -s inherit_errexit

TAG_PREFIX=${TAG_PREFIX-}
FORGE=${FORGE:-github}
VIA=${VIA:-push}
BRANCH=${BRANCH:-main}
PART=${PART:-}
VERSION=${VERSION:-}
NOTES=${NOTES:-}
CHANGELOG=${CHANGELOG:-}
FRAGMENTS=${FRAGMENTS:-}
PREPARE=${PREPARE:-}
CI=${CI:-false}
YES=${YES:-false}

semver="^${TAG_PREFIX}[0-9]+[.][0-9]+[.][0-9]+$"
# The heading kinds that a changelog section may group its entries under, in the order that a cut writes them.
kinds=(Added Changed Deprecated Removed Fixed Security)
kinds_re="^($(IFS='|' && echo "${kinds[*]}"))$"

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

on_github() { [[ $(git remote get-url origin) == *github.com[:/]* ]]; }

# Prints "<commit> <tag>" per version tag on origin.
# A local tag counts for nothing: a clone missing a tag would offer that tag again.
remote_tags() {
  git ls-remote --tags origin |
    sed -E 's#\^\{\}$##; s#refs/tags/##' |
    awk -v re="$semver" '$2 ~ re { print $1, $2 }'
}

newest() { awk '{ print $NF }' | sort -uV | tail -1; }

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

# Prints keepachangelog for a changelog headed "## [Unreleased]", plain for one headed "## Unreleased".
changelog_style() {
  if grep -qx '## \[Unreleased\]' "$CHANGELOG"; then
    echo keepachangelog
  elif grep -qx '## Unreleased' "$CHANGELOG"; then
    echo plain
  else
    die "$CHANGELOG holds no \"## [Unreleased]\" or \"## Unreleased\" section."
  fi
}

fragment_files() {
  [[ -n $FRAGMENTS && -d $FRAGMENTS ]] || return 0
  find "$FRAGMENTS" -maxdepth 1 -name "*.md" | sort
}

# Prints the lines under Unreleased, blank ones left out.
changelog_unreleased() {
  awk '
    /^## / { inside = ($0 ~ /^## \[?Unreleased\]?$/); next }
    /^\[[^]]+\]: / { inside = 0 }
    inside && NF
  ' "$CHANGELOG"
}

# Fails on a changelog or a fragment grouping entries under a kind outside $kinds.
changelog_check() {
  [[ -f $CHANGELOG ]] || die "CHANGELOG names $CHANGELOG, which does not exist."
  changelog_style >/dev/null
  local unknown
  # shellcheck disable=SC2046 # Fragment names come from find and hold no spaces.
  unknown=$(awk -v re="$kinds_re" '/^### / { kind = substr($0, 5); sub(/[ \t]+$/, "", kind); if (kind !~ re) print FILENAME ": " kind }' \
    "$CHANGELOG" $(fragment_files))
  [[ -z $unknown ]] || die "$(head -1 <<<"$unknown") lies outside ${kinds[*]}."
  if [[ -n $FRAGMENTS && -n $(changelog_unreleased) ]]; then
    die "$CHANGELOG holds lines under Unreleased. Move them into a fragment in $FRAGMENTS, then release."
  fi
}

# Prints the entries pending for the next release: the fragments with FRAGMENTS, the Unreleased section otherwise.
changelog_pending() {
  if [[ -n $FRAGMENTS ]]; then
    local files
    files=$(fragment_files)
    # shellcheck disable=SC2086 # Fragment names come from find and hold no spaces.
    [[ -z $files ]] || cat $files
  else
    changelog_unreleased
  fi
}

# Prints the part that the pending kinds bump, counting up from current.
# A removal takes a major from 1.0.0 on, any kind besides Fixed or Security a minor, Fixed or Security alone a patch.
changelog_part() {
  local pending found major
  pending=$(changelog_pending)
  found=$(awk '/^### / { kind = substr($0, 5); sub(/[ \t]+$/, "", kind); next } kind != "" && NF { print kind }' <<<"$pending" | sort -u)
  [[ -n $found ]] || die "Nothing waits for a release in ${FRAGMENTS:-$CHANGELOG}."
  major=${1#"$TAG_PREFIX"}
  major=${major%%.*}
  if grep -qx Removed <<<"$found" && ((${major:-0} >= 1)); then
    echo major
  elif grep -qvxE 'Fixed|Security' <<<"$found"; then
    echo minor
  else
    echo patch
  fi
}

# Prints the tag of every version section, newest first: "## [1.2.3] - date" and "## v1.2.3 (date)" alike.
changelog_versions() {
  awk -v prefix="$TAG_PREFIX" -v re="$semver" '
    /^## \[[0-9]+\.[0-9]+\.[0-9]+\]/ { v = substr($2, 2); sub(/\].*$/, "", v); print prefix v; next }
    /^## / && $2 ~ re { print $2 }
  ' "$CHANGELOG"
}

# Prints the body of a version's section without the blank lines around it.
changelog_entries() {
  awk -v tag="$1" -v version="${1#"$TAG_PREFIX"}" '
    /^## / { if (inside) exit; inside = ($2 == tag || index($0, "## [" version "]") == 1); next }
    /^\[[^]]+\]: / { if (inside) exit }
    inside { lines[++n] = $0 }
    END {
      first = 1; while (first <= n && lines[first] ~ /^[ \t]*$/) first++
      last = n; while (last >= first && lines[last] ~ /^[ \t]*$/) last--
      for (i = first; i <= last; i++) print lines[i]
    }
  ' "$CHANGELOG"
}

# Prints the fragments' lines under each kind, in the order of $kinds.
fragments_gathered() {
  local kind lines files out=""
  files=$(fragment_files)
  for kind in "${kinds[@]}"; do
    # shellcheck disable=SC2086 # Fragment names come from find and hold no spaces.
    lines=$(awk -v heading="### $kind" 'FNR == 1 { inside = 0 } $0 == heading { inside = 1; next } /^### / { inside = 0 } inside && NF' $files)
    [[ -z $lines ]] || out+="${out:+$'\n'}### $kind"$'\n\n'"$lines"$'\n'
  done
  printf '%s' "$out"
}

# Moves the pending entries under a section for the tag, previous naming the release before it.
# A Keep a Changelog file gets its compare links pointed at the new tag.
# A tag holding its section already stays as it is.
changelog_cut() {
  local tag=$1 previous=$2 version=${1#"$TAG_PREFIX"} style heading notes cut
  changelog_versions | grep -xF "$tag" >/dev/null && return
  style=$(changelog_style)
  if [[ $style == keepachangelog ]]; then
    heading="## [$version] - $(date -u +%F)"
  else
    heading="## $tag ($(date -u +%F))"
  fi
  notes=$(mktemp)
  [[ -z $FRAGMENTS ]] || fragments_gathered >"$notes"
  cut=$(awk -v heading="$heading" -v notes="$notes" -v version="$version" -v tag="$tag" -v previous="$previous" -v branch="$BRANCH" '
    /^## \[?Unreleased\]?$/ {
      print; print ""; print heading
      first = 1
      while ((getline line < notes) > 0) { if (first) print ""; first = 0; print line }
      next
    }
    /^\[Unreleased\]: / {
      base = $2; compare = "/compare/"; tags = "/releases/tag/"
      if (base ~ /\/-\/compare\//) { compare = "/-/compare/"; tags = "/-/tags/" }
      sub(/\/(-\/)?compare\/.*$/, "", base)
      print "[Unreleased]: " base compare tag "..." branch
      if (previous != "") print "[" version "]: " base compare previous "..." tag
      else print "[" version "]: " base tags tag
      next
    }
    { print }
  ' "$CHANGELOG")
  rm -f "$notes"
  printf '%s\n' "$cut" >"$CHANGELOG"
  if [[ -n $FRAGMENTS ]]; then
    # shellcheck disable=SC2046 # Fragment names come from find and hold no spaces.
    git rm --quiet $(fragment_files)
  fi
}

# Prints the version that the next release counts up from: the newest of the tags on origin and the changelog's sections.
current_version() {
  { awk '{ print $2 }' <<<"$1"; [[ -z $CHANGELOG ]] || changelog_versions; } | grep . | newest || true
}

# Prints the tag that a release from HEAD takes.
# A rerun pushing directly finishes the release that HEAD already carries: a tag on origin,
# or a changelog section cut but never tagged.
next_tag() {
  local tags current tagged cut part
  tags=$(remote_tags)
  if [[ -n $VERSION ]]; then
    [[ $VERSION =~ $semver ]] || die "VERSION takes ${TAG_PREFIX}X.Y.Z, not $VERSION."
    echo "$VERSION"
    return
  fi
  [[ -z $CHANGELOG ]] || changelog_check
  if [[ $VIA == push ]]; then
    tagged=$(awk -v head="$(git rev-parse HEAD)" '$1 == head' <<<"$tags" | newest)
    if [[ -n $tagged ]]; then
      echo "$tagged"
      return
    fi
    if [[ -n $CHANGELOG ]]; then
      cut=$(changelog_versions | head -1)
      if [[ -n $cut ]] && ! awk '{ print $2 }' <<<"$tags" | grep -xF "$cut" >/dev/null && newer "$cut" "$(newest <<<"$tags")"; then
        echo "$cut"
        return
      fi
    fi
  fi
  current=$(current_version "$tags")
  part=${PART:-minor}
  [[ -n $PART || -z $CHANGELOG ]] || part=$(changelog_part "$current")
  bump "$current" "$part"
}

release_branch() { echo "release-$1"; }

# Fails unless HEAD can take the release.
# Pushing directly, commits past origin pass alone as the release commit of a run cut short.
preflight() {
  local tag=$1 commit
  [[ $FORGE == github || $FORGE == tag ]] || die "FORGE takes github or tag, not $FORGE."
  [[ $VIA == push || $VIA == merge-request ]] || die "VIA takes push or merge-request, not $VIA."
  [[ $VIA == push || -n $CHANGELOG || -n $PREPARE ]] || die "VIA=merge-request needs CHANGELOG or PREPARE to commit something."
  [[ $(git branch --show-current) == "$BRANCH" ]] || die "A release runs from $BRANCH. Switch to $BRANCH, then release."
  [[ -z $(git status --porcelain) ]] || die "The working tree carries changes. Commit them, then release."
  git fetch --quiet origin "$BRANCH"
  git merge-base --is-ancestor "origin/$BRANCH" HEAD || die "$BRANCH lacks commits from origin. Pull, then release."
  for commit in $(git rev-list "origin/$BRANCH..HEAD"); do
    [[ $VIA == push && $(git log -1 --format=%s "$commit") == "Release $tag" ]] ||
      die "$BRANCH holds commits that origin lacks. Push them, then release."
  done
  if [[ $VIA == merge-request && -n $(git ls-remote --heads origin "$(release_branch "$tag")") ]]; then
    die "The branch $(release_branch "$tag") exists on origin. Merge or close its merge request, then release."
  fi
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
# none, finished, ok, a line naming each run or job with its state and the links to what failed.
ci_state() {
  local kind=$1 ref=$2
  if on_github; then
    local filter=--commit
    [[ $kind == tag ]] && filter=--branch
    gh run list "$filter" "$ref" --limit 50 --json name,status,conclusion,url --jq '
      def passed: .conclusion | IN("success", "skipped", "neutral");
      [ (length == 0), all(.status == "completed"), all(passed),
        (map(.name + " " + (if .status == "completed" then .conclusion else .status end)) | join(", ")),
        (map(select(.status == "completed" and (passed | not)) | .url) | join(" ")) ]
      | map(tostring) | join("\u001f")'
    return
  fi
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
  local tag=$1 notes=""
  gh release view "$tag" >/dev/null 2>&1 && return
  local args=(--verify-tag --title "$tag")
  if [[ -n $NOTES ]]; then
    args+=(--notes-file "$NOTES")
  elif [[ -n $CHANGELOG ]]; then
    notes=$(mktemp)
    changelog_entries "$tag" >"$notes"
    args+=(--notes-file "$notes" --generate-notes)
  else
    args+=(--generate-notes)
  fi
  gh release create "$tag" "${args[@]}" >/dev/null
  [[ -z $notes ]] || rm -f "$notes"
}

# Cuts the changelog, runs PREPARE and commits what both changed as the release commit.
commit_release() {
  local tag=$1 previous=$2
  [[ -z $CHANGELOG ]] || changelog_cut "$tag" "$previous"
  [[ -z $PREPARE ]] || TAG=$tag VERSION=${tag#"$TAG_PREFIX"} sh -c "$PREPARE"
  git add --update
  git diff --cached --quiet || git commit --quiet -m "Release $tag"
}

# Tags HEAD, pushes the commit and the tag together and publishes the release.
release_by_push() {
  local tag=$1
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

# Pushes the release commit to a branch of its own and opens its merge request.
# CI on the release branch publishes the tag once the merge request lands.
release_by_merge_request() {
  local tag=$1 branch title description
  branch=$(release_branch "$tag")
  title="Release $tag"
  description="Merging this releases $tag."
  [[ -z $CHANGELOG ]] || description="Merging this releases $tag with the notes under its section in $CHANGELOG."
  if on_github; then
    git push --quiet origin "HEAD:refs/heads/$branch"
    gh pr create --base "$BRANCH" --head "$branch" --title "$title" --body "$description"
  else
    git push --quiet origin "HEAD:refs/heads/$branch" \
      -o merge_request.create -o "merge_request.target=$BRANCH" -o merge_request.remove_source_branch \
      -o "merge_request.title=$title" -o "merge_request.description=$description"
  fi
  echo "Opened the merge request for $tag. Merging it releases $tag."
}

release() {
  local tag previous confirm question
  tag=$(next_tag)
  preflight "$tag"
  previous=$( (remote_tags | awk '{ print $2 }'; [[ -z $CHANGELOG ]] || changelog_versions) | grep -vxF "$tag" | newest || true)
  question="Release $tag from \"$(git log -1 --format=%s)\"?"
  [[ $VIA == push ]] || question="Open the merge request releasing $tag from \"$(git log -1 --format=%s)\"?"
  if [[ $YES != true ]]; then
    echo "$question [y/N]"
    read -r confirm
    [[ $confirm == [yY]* ]] || die "Stopped before anything changed."
  fi
  if [[ $FORGE == github ]] || on_github; then github_token; fi

  if [[ $CI == true ]]; then
    follow_ci "$BRANCH at $(git rev-parse --short "origin/$BRANCH")" sha "$(git rev-parse "origin/$BRANCH")"
  fi

  if [[ $VIA == merge-request ]]; then
    git switch --quiet --detach
    trap 'git switch --quiet "$BRANCH"' EXIT
    commit_release "$tag" "$previous"
    release_by_merge_request "$tag"
  else
    commit_release "$tag" "$previous"
    release_by_push "$tag"
  fi
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
