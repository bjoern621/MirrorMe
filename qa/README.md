# Tested hardware

Each release's test runs, one row per machine.
`task qa:record` writes a run's record under `runs/` and redraws this page ([releasing](../docs/releasing.md)).
A test count links its record, which names every failure and every skip with its reason.

## v0.13.1

| Machine | OS | CPU | GPU | Media stack | Tests |
| --- | --- | --- | --- | --- | --- |
| fractal | NixOS 26.11 (Zokor), kernel 7.2.11-zen1, amd64 | AMD Ryzen 7 7800X3D 8-Core Processor | NVIDIA Corporation GB203 [GeForce RTX 5080] (rev a1) (nvidia 615.78.08)<br>Advanced Micro Devices, Inc. [AMD/ATI] Raphael [Radeon 610M] (rev cb) (amdgpu) | GStreamer 1.28.7, FFmpeg 9.0.2, VA-API NVDEC driver [direct backend] | [1973 passed, 0 failed, 79 skipped](runs/v0.13.1/fractal.json) |

## v0.13.0

| Machine | OS | CPU | GPU | Media stack | Tests |
| --- | --- | --- | --- | --- | --- |
| fractal | NixOS 26.11 (Zokor), kernel 7.2.11-zen1, amd64 | AMD Ryzen 7 7800X3D 8-Core Processor | NVIDIA Corporation GB203 [GeForce RTX 5080] (rev a1) (nvidia 615.78.08)<br>Advanced Micro Devices, Inc. [AMD/ATI] Raphael [Radeon 610M] (rev cb) (amdgpu) | GStreamer 1.28.7, FFmpeg 9.0.2, VA-API NVDEC driver [direct backend] | [1966 passed, 15 failed, 79 skipped](runs/v0.13.0/fractal.json) |

## v0.12.1

| Machine | OS | CPU | GPU | Media stack | Tests |
| --- | --- | --- | --- | --- | --- |
| laptop-780m | NixOS 26.11 (Zokor), kernel 7.3.0, amd64 | AMD Ryzen 7 7840S with Radeon 780M Graphics | Advanced Micro Devices, Inc. [AMD/ATI] Phoenix1 (rev 02) (amdgpu) | GStreamer 1.28.7, FFmpeg 9.0.2, Mesa Gallium driver 26.2.5 | [1941 passed, 0 failed, 69 skipped](runs/v0.12.1/laptop-780m.json) |
