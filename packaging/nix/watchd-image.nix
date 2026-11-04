# The viewer service as a container image, for a Kubernetes deployment of it.
#
# Nothing but the binary.
# Its listen address and where groupd answers are arguments, so nothing is baked in here.

{
  dockerTools,
  screenshare-watchd,
  version,
}:

dockerTools.buildLayeredImage {
  name = "screenshare-watchd";
  tag = version;

  contents = [ screenshare-watchd ];

  config = {
    Entrypoint = [ "/bin/watchd" ];
    # Answers JSON on a port above 1024 and writes nothing.
    User = "65532:65532";
  };
}
