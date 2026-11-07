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

`task release` must run from a clean main with every commit pushed:

1. Takes the suggested version, or the one named after `--`: `task release -- v0.13.0`.
2. Waits for the check workflow on the last commit that touches more than `CHANGELOG.md`, and stops on red.
3. Asks before it changes anything.
4. Moves the Unreleased entries under `## v0.13.0 (date)` and commits `Release v0.13.0`.
5. Tags, pushes main and the tag, and creates the GitHub release with its text.
6. Follows the release workflow to its end.

Every step leaves a state already reached as it is.
A run stopped by a red build or a lost connection runs again with the version named: `task release -- v0.13.0`.

`gh` comes with the dev shell and must be logged in (`gh auth login` or `GH_TOKEN`).

## Recording a test run

After a release is out, each machine runs `task qa:record`:

1. Exports the newest tag to a temporary directory, so the run tests the release whatever the checkout holds.
2. Runs the backend suite with the `hardware` tag from that tree.
3. Reads the machine: OS, CPU, GPUs and their drivers, the media stack and more.
4. Asks the checklist: the checks that need a person at the running app.
5. Writes `qa/runs/<version>/<machine>.json`, redraws `qa/README.md` and commits both.

Pushing main publishes the record.
The notes workflow then redraws the "Tested on" table of that release.

| Option | Effect |
| --- | --- |
| `-version v0.12.1` | tests an older release |
| `-machine desk` | names the machine; else `QA_MACHINE`, else the host name |
| `-checklist=false` | runs the suite alone, for a headless machine such as a Raspberry Pi |

A second run on the same machine replaces its record.
A skip in the record names a hardware path that the machine left uncovered, with the test's own reason.

## Where the answers show

| Place | Shows |
| --- | --- |
| `qa/README.md` | every release, one row per machine |
| a GitHub release | its entries, its "Tested on" table and the compare link to the release before |
| a record | each failing test, each skip with its reason, each checklist answer |

The check workflow fails where `qa/README.md` differs from the records.
