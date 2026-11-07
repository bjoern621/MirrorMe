# Tested hardware

Each release's test runs, one row per machine.
`task qa:record` writes a run's record under `runs/` and redraws this page ([releasing](../docs/releasing.md)).
A test count links its record, which names every failure and every skip with its reason.

## v0.12.1

| Machine | OS | CPU | GPU | Media stack | Tests | Checklist |
| --- | --- | --- | --- | --- | --- | --- |
| laptop-780m | NixOS 26.11 (Zokor), kernel 7.3.0, amd64 | AMD Ryzen 7 7840S with Radeon 780M Graphics | Advanced Micro Devices, Inc. [AMD/ATI] Phoenix1 (rev 02) (amdgpu) | GStreamer 1.28.7, FFmpeg 9.0.2, Mesa Gallium driver 26.2.5 | [1941 passed, 0 failed, 69 skipped](runs/v0.12.1/laptop-780m.json) | not run |
