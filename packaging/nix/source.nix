# The inputs the app's two builds read, and nothing else.
# A working tree also holds the .NET intermediate and output directories,
# so a source carrying those would be gigabytes and would change on every local build.
#
# Its store path names the build: two commits holding the same files here compile to the same app,
# and packaging/nix/release.json names the release a store path was published as (flake.nix).
{ lib }:

lib.cleanSourceWith {
  name = "mirrorme-source";
  src = ../../.;
  filter =
    path: type:
    let
      rel = lib.removePrefix "${toString ../../.}/" (toString path);
      top = lib.head (lib.splitString "/" rel);
    in
    lib.elem top [
      "api"
      "backend"
      "avalonia"
    ]
    && !(
      type == "directory"
      && lib.elem (baseNameOf path) [
        "obj"
        "bin"
      ]
    );
}
