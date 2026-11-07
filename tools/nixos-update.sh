#!/usr/bin/env bash
# Moves the NixOS system flake's input on this repository to its newest commit, then switches to it.
# Run as `task nixos:update`.
# The system flake comes from NIXOS_FLAKE, a directory holding flake.nix, else from /etc/nixos.
# A system that is not NixOS, or a flake taking no input on this repository, stays as it is.
# A second run with nothing new upstream rebuilds an unchanged system.
set -euo pipefail

die() { echo "nixos:update: $*" >&2; exit 1; }

if [[ ! -e /etc/NIXOS ]]; then
  echo "nixos:update: not NixOS, nothing to update"
  exit 0
fi

# The dev shell's systemd-run is older than the one that the system's nixos-rebuild calls
# and refuses its --output=cat.
export PATH="/run/wrappers/bin:/run/current-system/sw/bin:$PATH"

flake=${NIXOS_FLAKE:-/etc/nixos}
[[ -f $flake/flake.nix ]] || die "no flake.nix in $flake: set NIXOS_FLAKE to the directory holding the system's flake.nix"
[[ -f $flake/flake.lock ]] || die "no flake.lock in $flake: build the system from it once first"
flake=$(cd "$flake" && pwd)

# Matched by source, since the input may take any name.
inputs=$(LOCK="$flake/flake.lock" nix eval --raw --impure --expr "$(cat <<'EOF'
let
  lock = builtins.fromJSON (builtins.readFile (builtins.getEnv "LOCK"));
  root = lock.nodes.${lock.root}.inputs;
  # GitHub takes a repository name in any case.
  # A lock file may name the former one, screen-sharing, which GitHub redirects.
  repo = "([Mm][Ii][Rr][Rr][Oo][Rr][Mm][Ee]|screen-sharing)";
  ours = name:
    let
      node = root.${name};
      o = if builtins.isString node then lock.nodes.${node}.original or { } else { };
    in
    (o.type or "" == "github" && o.owner or "" == "bjoern621" && builtins.match repo (o.repo or "") != null)
    || (o.type or "" == "git" && builtins.match "https://github.com/bjoern621/${repo}(\\.git)?" (o.url or "") != null);
in
builtins.concatStringsSep " " (builtins.filter ours (builtins.attrNames root))
EOF
)")
if [[ -z $inputs ]]; then
  echo "nixos:update: $flake takes no input on github:bjoern621/MirrorMe, nothing to update"
  exit 0
fi

# A lock file that root owns, as one under /etc/nixos does, takes sudo to rewrite.
sudo=()
[[ -w $flake/flake.lock ]] || sudo=(sudo)

for input in $inputs; do
  echo "nixos:update: updating input $input of $flake"
  "${sudo[@]}" nix flake update "$input" --flake "$flake"
done

echo "nixos:update: switching to $flake"
sudo nixos-rebuild switch --flake "$flake"
