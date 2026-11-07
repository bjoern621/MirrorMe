# tools

Probes for a running relay or backend, the renderer for the comparison crops, the release script and the NixOS update script.

Nothing here gets built or shipped.
`task backend` and `task backend:stop` run `backend-stop.ps1` on Windows.
`task release` runs `release.sh`.
`task nixos:update` runs `nixos-update.sh`.

## What is here

| Tool | Does |
| --- | --- |
| `bruno/` | the group service, membership and relay HTTP APIs as a [Bruno](https://usebruno.com) collection, with its own README |
| `backend-stop.ps1` | release the control pipe a previous `task backend` left held on Windows |
| `crops.sh` | render the comparison crops the settings tooltips show, from their manifest, through the real encoder |
| `release.sh` | cut, tag and publish a release, and follow its workflow ([releasing](../docs/releasing.md)) |
| `nixos-update.sh` | update the NixOS system flake's input on this repository and switch to it ([releasing](../docs/releasing.md)) |

The flake's dev shell carries what `crops.sh` runs on: ffmpeg with libx265 and jq.
