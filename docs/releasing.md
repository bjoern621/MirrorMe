# Releasing

How a release gets its number, its notes and its record of the machines that tested it.
`CHANGELOG.md` holds the notes, `qa/runs` the test records.
Every GitHub release text renders from those two.

## Changelog entries

A user-visible change must add its entry to `CHANGELOG.md` under Unreleased, in the same commit.
An entry must sit under Added, Changed, Fixed or Removed.
Another kind fails the parse and every release step with it.
A test, a CI step or a refactor takes no entry.

The kinds pending decide the next number:

| Pending | Bump | Example |
| --- | --- | --- |
| Removed, from v1.0.0 on | major | v1.4.2 to v2.0.0 |
| Added, Changed, or Removed before v1.0.0 | minor | v0.12.1 to v0.13.0 |
| Fixed alone | patch | v0.12.1 to v0.12.2 |

## Cutting a release

`task release` comes from the shared [taskfiles](https://github.com/bjoern621/taskfiles) release, copied into `taskfiles/`.
It must run from a clean main with every commit pushed:

1. Takes the version that `task version` prints, or the one named as `task release VERSION=v0.13.0`.
2. Asks before it changes anything.
3. Waits for every workflow on the tip of main, the check workflow among them, and stops on red.
4. Moves the Unreleased entries under `## v0.13.0 (date)`.
5. Binds the version to the app's source in `packaging/nix/release.json` (`tools/release-json.sh`).
6. Commits both as `Release v0.13.0`, tags it, pushes main and the tag and creates the GitHub release with the changelog section.
7. Follows the workflows on the tag to their end: the release workflow builds the packages, the notes workflow renders the full text.

Every step leaves a state already reached as it is.
A run stopped by a red build or a lost connection runs again as `task release` and finishes the release that it started.

Take a newer shared release with the copy command in the taskfiles README.
`gh` comes with the dev shell and must be logged in (`gh auth login` or `GH_TOKEN`).

## Updating a NixOS machine

A NixOS machine whose system flake takes this repository as an input runs `task nixos:update` to install a release.
It moves that input to the newest commit and switches the system to it.
The installed app then runs the released code, as long as main holds no commit past the tag.
Its window shows the release's version as long as the app's source stays as the release left it, and the commit otherwise.
`NIXOS_FLAKE` names the flake's directory where it sits outside `/etc/nixos`, as in `NIXOS_FLAKE=~/git/nixos-config/hosts/fractal task nixos:update`.

## Recording a test run

After a release is out, each machine runs `task qa:record`, which asks nothing:

1. Exports the newest tag to a temporary directory, so the run tests the release whatever the checkout holds.
2. Runs the backend suite with the `hardware` tag from that tree.
3. Reads the machine: OS, CPU, GPUs and their drivers, the media stack and more.
4. Writes `qa/runs/<version>/<machine>.json`, redraws `qa/README.md` and commits both.

Pushing main publishes the record.
The notes workflow then redraws the "Tested on" table of that release.

| Option | Effect |
| --- | --- |
| `-version v0.12.1` | tests an older release |
| `-machine desk` | names the machine; else `QA_MACHINE`, else the host name |

A second run on the same machine replaces its record.
A skip in the record names a hardware path that the machine left uncovered, with the test's own reason.

## Where the answers show

| Place | Shows |
| --- | --- |
| `qa/README.md` | every release, one row per machine |
| a GitHub release | its entries, its "Tested on" table and the compare link to the release before |
| a record | each failing test and each skip with its reason |

The check workflow fails where `qa/README.md` differs from the records.
