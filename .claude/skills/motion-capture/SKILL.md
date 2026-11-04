---
name: motion-capture
description: Records the screen and reads motion out of the frames, then reproduces it in a headless probe. Use when the user asks to capture the screen or wants to show a layout shift, flicker or animation bug in MirrorMe.
---

The user drives the app and the agent watches.
Motion moves by frame, so every claim names a time and a value read off a frame or a probe.

Files land in `$CLAUDE_JOB_DIR/tmp` (a `mktemp -d` outside a job), since a recording shows everything on the screens.

## 1. Record

1. Tell the user how long the lead and the recording last and that every monitor gets captured.
2. Run `scripts/record.sh <out> [seconds] [lead]` in the background (defaults 15 and 6).
   It writes `<out>/<monitor>.mp4` at 60 fps and shows a Hyprland notification at both ends.

Done when every monitor has a file and the log shows no failure.

## 2. Find the moments

1. Run `scripts/moments.sh <out>/<monitor>.mp4` per file.
   It lists each scene change with its score and writes `<monitor>-sheet.png`, one frame a second.
2. Read the sheet of the monitor holding the changes.

A lone high score marks a snap.
Scores falling off over several frames mark an ease.
One action repeated by the user gives one pattern per repeat, and the pattern is the bug's signature.

Done when each repeat of the action has a start time.

## 3. Read the frames

Take the window's crop from `hyprctl clients`: its `size` and `at`, minus the monitor's origin from `hyprctl monitors`.

Run `scripts/strip.sh <video> <start> <seconds> <nth> <W:H:X:Y> <out.png>` from just before a start time through the settled state.
Each frame carries its time.
`<nth>` 1 keeps every frame and 3 suits a 260 ms change.

Read every region of the window: bands, body, strip and the elements that stayed still.
Compare a settled rectangle against the same rectangle mid-way to tell a shift from a zoom.

Done when the bug reads as "element, value, time" and the opposite direction of the action is read too.

## 4. Reproduce

Follow [probe.md](probe.md).
The probe's numbers are the failing test: run it red on the bug, then green on the fix, with the sequence around the fix sampled too.

## 5. Report

State what the recording showed, what the probe measured and whether the real app ran the fix.
A recording stays out of the repository.
