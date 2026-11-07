# Changelog

What changes for somebody taking each release, newest first.
`task release` moves the entries under the new version and posts them as the release text ([releasing](docs/releasing.md)).
Releases before v0.11.0 carry no notes, and their commits sit in the [history](https://github.com/bjoern621/MirrorMe/commits/v0.10.3).

## Unreleased

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
