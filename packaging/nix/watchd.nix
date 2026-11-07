# The viewer service, as the machine beside the relay installs it (docs/viewer-reports.md).
#
# A package of its own for the reason groupd.nix is one:
# it lands on the relay's machine, and nothing a server serving JSON needs rides in the app's closure.
{
  lib,
  buildGoModule,
  version,
}:

buildGoModule {
  pname = "mirrorme-watchd";
  inherit version;

  # The Go tree alone, groupd.nix's filter verbatim: nothing here builds the shell.
  src = lib.cleanSourceWith {
    name = "mirrorme-watchd-source";
    src = ../../.;
    filter =
      path: _:
      let
        rel = lib.removePrefix "${toString ../../.}/" (toString path);
        top = lib.head (lib.splitString "/" rel);
      in
      lib.elem top [
        "api"
        "backend"
      ];
  };

  # The module cache rather than a vendor directory, for the reason packaging/nix/package.nix states.
  # The hash is the module's one dependency set, shared with the other binaries.
  proxyVendor = true;
  vendorHash = "sha256-3sMf233ihFSRiN1kr+uG3lljk0XUDrhkTbrGRe5Z1Bs=";

  modRoot = "backend";
  subPackages = [ "cmd/watchd" ];

  # The number every answer names, which is how a member's relay check reads what a deployment
  # is running (backend/internal/serving).
  ldflags = [ "-X main.version=${version}" ];

  # Nothing this binary imports reaches backend/internal/receive,
  # the one cgo and GStreamer package,
  # so it builds without a C compiler or GStreamer headers on the server.
  env.CGO_ENABLED = 0;

  meta = {
    description = "Viewer figures for the MirrorMe relay, carried to each stream's publisher";
    mainProgram = "watchd";
    platforms = lib.platforms.linux;
  };
}
