# Changelog

What changes for somebody taking each release, newest first.
`task release` moves the entries under the new version and posts them as the release text ([releasing](docs/releasing.md)).
Releases before v0.11.0 carry no notes, and their commits sit in the [history](https://github.com/bjoern621/MirrorMe/commits/v0.10.3).

## Unreleased

### Fixed

- The status band leaves "Checking for updates" once the check at start has ended, where it could stay on that line until MirrorMe was restarted.
- An update of the Windows zip or the Linux tarball removes the previous version's folder from beside the install, where it could stay and take up disk until the next update.

## v0.15.0 (2026-11-08)

### Changed

- Screenshots and screen recordings on Windows show MirrorMe's windows, where they used to leave them out.
  A stream of the screen MirrorMe is on now shows its own tile inside the picture.

### Fixed

- On Windows the tray icon's right-click menu is the system's own menu, light or dark as Windows is set, where it was drawn in MirrorMe's dark style.
- Stopping a stream comes back within seconds when the graphics driver stops responding, where MirrorMe stayed frozen until it was restarted.
- The Windows zip installs its updates, where it failed to replace its own folder and stayed closed.
  An update also waits out a file held open by a virus scanner, and MirrorMe starts again after one that could not install.
- Closing a stream gives its graphics driver threads and memory back, where each one opened kept them until MirrorMe exited.
- A stream sent over SRT on a fast connection to the relay reaches viewers whole, where its keyframes got lost and a tile stayed on "Connecting" or showed moving fragments on black.

## v0.14.1 (2026-11-07)

### Fixed

- A window switcher on a Wayland desktop shows MirrorMe's name and icon for its windows, where it could show another app's.

## v0.14.0 (2026-11-07)

### Added

- "Start when you sign in" in the settings dialog starts MirrorMe when you sign in, on Linux and Windows, off by default.

### Changed

- The settings dialog leaves out the Development section and its test streams, which run in a development build alone.
- The first start opens without the crash report question.
  "Send crash reports" in the settings dialog starts off.
  An install that left the question unanswered keeps it off.

## v0.13.1 (2026-11-07)

### Fixed

- Linking Discord on Windows completes, where it stopped on "a link start names the nonce its landing carries".
- Lossless NVENC on the GStreamer engine starts on NVIDIA 615.
  Picking lossless H.264 moves the chroma to 4:4:4, since the driver refuses lossless H.264 in 4:2:0.
- The GStreamer engine offers its video formats on a launch that finds its plugin cache cold.
  The Nix package keeps a plugin cache of its own, which other GStreamer programs leave as it was.
- A GStreamer publish that fails to start names the element that refused and its reason.

## v0.13.0 (2026-11-07)

### Added

- In the publish settings, "When upload falls behind" lets you choose between skipping to the next keyframe (the default) and slowing the encoder.

### Changed

- The project lives at [github.com/bjoern621/MirrorMe](https://github.com/bjoern621/MirrorMe).
  The old address forwards there.
  A Nix flake takes it as `github:bjoern621/MirrorMe`.
- The relay images publish as `ghcr.io/bjoern621/mirrorme-relay`, `-proxy`, `-groupd`, `-discordd` and `-watchd`.
- The relay proxy reads `MIRRORME_DOMAIN`, `MIRRORME_GROUPD`, `MIRRORME_WATCHD`, `MIRRORME_DISCORDD`, `MIRRORME_RELAY_WEBRTC` and `MIRRORME_RELAY_HLS` in place of the `SCREENSHARE_` names.
  The relay looks for its certificate under `/var/lib/mirrorme-tls`.

### Fixed

- A slow upload stalls the local preview no more.
  Viewers see a freeze up to the next keyframe, and every frame they get decodes.

## v0.12.1 (2026-11-05)

### Added

- The viewers tab reads each viewer's latency from the publisher's encoder to that viewer's decoder.

### Changed

- Rows and tiles slide when a list changes.
- Steps, tabs, flags and labels fade in as they appear, and a fold scrolls into view as it opens.
- Buttons, rows, chips, toggles and menus answer a press.

### Fixed

- SRT holds its watch window once, so a viewer at a 300 ms window plays 0.6 s behind instead of 1.5 s.
- The delay figures hold on RTMP and HLS, whose frames arrive ahead of their timestamps.
- A join over SRT with sound starts at any point between two keyframes.

## v0.12.0 (2026-11-04)

### Added

- The viewers tab names every viewer watching from the app, with the latency, frame rate and dropped frames that viewer's own app measures, over every transport.
  A relay runs a new service for it, watchd, beside the group service.
- A tile zooms with the wheel, a pinch or a drag, and pans with two fingers on a touchpad.
- The strip under the stage hides.
- A muted tile carries a mark beside its name.

### Changed

- Every state change in the window animates, and folds, dialogs and panels fade in and out.
- Every received stream plays at one loudness and opens at half volume.
- The stats panel shows one stage at a time in plain sentences, and stays off a tile too narrow for it.
- Setup moves field help behind a hover, names both ends of every encoder effort ladder and names the cursor modes.
- Every text step and hue clears 4.5:1 contrast.
- Decoding into dmabuf feeds the GL render chain without a copy.

### Fixed

- VA encoders on radeonsi take the target usage they read.

## v0.11.1 (2026-11-03)

### Fixed

- The window stays open when an update check succeeds after a failed one, such as a check run while a release is still uploading.

## v0.11.0 (2026-11-02)

### Changed

- Streams play on a stage: one stream large, the rest in a strip below it.
- Setup asks four steps of questions, prices each choice before it is picked and shows comparison crops of real encodes.
- Settings open one section at a time, and the watch settings open as a panel over the stage.
- Insights lead with a verdict on the running stream, three figure cards and one chart.
- Every message the app shows is worded in the app, and an error names its cause beside the relay's or the encoder's own words.

### Fixed

- A GStreamer publish over SRT delivers frames to a relay behind a 1280-byte link.
- Watching over RTSP works against an encrypted relay.
- Watching over HLS plays without flooding the relay with playlist requests.
