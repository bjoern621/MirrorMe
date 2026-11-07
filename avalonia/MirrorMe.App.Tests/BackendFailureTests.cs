using Grpc.Core;
using MirrorMe.Api.V1;
using MirrorMe.App.Backend;
using MirrorMe.App.Copy;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// What a failed call turns into on screen.
///
/// <b>The division is who wrote the status, not which code it carries.</b>
/// The contract's table gives the backend <c>UNAVAILABLE</c> for a relay it could not reach
/// and for a child process that would not start (<c>docs/ipc-api.md</c>, "Errors"),
/// so a shell reading that code as "nothing is listening" answers a press of Start sharing
/// with a sentence about the connection to a backend it just resolved a form through.
/// A served refusal reads as this shell's sentence for its code,
/// and only a failure the client made itself names the endpoint.
/// </summary>
public sealed class BackendFailureTests
{
    /// <summary>A third party's words follow the sentence on a line of their own, verbatim.</summary>
    [Fact]
    public void ARefusalTheBackendServedReadsAsThisShellsSentence()
    {
        var refusal = new Refusal { Text = new Text { Code = TextCode.SettingsUnsaved }, Message = "no space left on device" };

        var failure = ControlBackend.Translate(Served.Refusal(StatusCode.Unavailable, refusal), CancellationToken.None);

        var unavailable = Assert.IsType<BackendUnavailableException>(failure);
        Assert.Equal("Could not save the settings.\nno space left on device", unavailable.Message);
    }

    /// <summary>A coded reason behind the refusal follows it as its own sentence.</summary>
    [Fact]
    public void ACodedCauseFollowsTheRefusal()
    {
        var cause = new Text { Code = TextCode.TrayIconOff };
        var text = new Text { Code = TextCode.PublishUnstarted };
        text.Args.Add(new TextArg { Name = TextArgName.Cause, Text = cause });

        var failure = ControlBackend.Translate(
            Served.Refusal(StatusCode.Unavailable, new Refusal { Text = text }), CancellationToken.None);

        Assert.Equal($"Could not start sharing. {Statements.Of(cause)}", failure.Message);
    }

    /// <summary>
    /// A served status carrying no refusal comes from a backend of another build.
    /// Its message is no sentence of this shell's, so the code stands in for it.
    /// </summary>
    [Fact]
    public void AServedStatusWithNoRefusalNamesItsCode()
    {
        var failure = ControlBackend.Translate(
            new RpcException(new Status(StatusCode.Unavailable, "cannot start publishing")), CancellationToken.None);

        var unavailable = Assert.IsType<BackendUnavailableException>(failure);
        Assert.Contains("Unavailable", unavailable.Message);
        Assert.DoesNotContain("cannot start publishing", unavailable.Message);
    }

    /// <summary>
    /// A failure the client library built from a local exception means the backend is not running,
    /// whatever code it wears: an absent named pipe arrives as <c>INTERNAL</c> on Windows,
    /// an unbound socket as <c>UNAVAILABLE</c>.
    /// Sentence names the address that was tried, the part a reader can act on.
    /// </summary>
    [Theory]
    [InlineData(StatusCode.Internal)]
    [InlineData(StatusCode.Unavailable)]
    public void AConnectionThatWasNeverMadeNamesTheEndpoint(StatusCode code)
    {
        var local = new Status(code, "Error starting gRPC call.", new IOException("the pipe is not there"));

        var failure = ControlBackend.Translate(new RpcException(local), CancellationToken.None);

        var unavailable = Assert.IsType<BackendUnavailableException>(failure);
        Assert.Contains("The backend is not running", unavailable.Message);
        Assert.Contains(ControlEndpoint.Describe(), unavailable.Message);
    }

    /// <summary>
    /// Exception promises a sentence and the screen asserts on it,
    /// so a status carrying no prose is named by its code rather than handed upwards blank.
    /// </summary>
    [Fact]
    public void AServedStatusWithNothingSaidNamesTheCode()
    {
        var failure = ControlBackend.Translate(
            new RpcException(new Status(StatusCode.Unknown, "")), CancellationToken.None);

        var unavailable = Assert.IsType<BackendUnavailableException>(failure);
        Assert.Contains("Unknown", unavailable.Message);
        Assert.Contains(ControlEndpoint.Describe(), unavailable.Message);
    }

    /// <summary>Flow cancels a resolve on every keystroke, and a superseded one is no failure the reader is told about.</summary>
    [Fact]
    public void AReadThisShellCancelledIsNotASentence()
    {
        using var cancelled = new CancellationTokenSource();
        cancelled.Cancel();

        var failure = ControlBackend.Translate(
            new RpcException(new Status(StatusCode.Cancelled, "Call canceled by the client.")), cancelled.Token);

        Assert.IsType<OperationCanceledException>(failure);
    }

    /// <summary>
    /// A <c>CANCELLED</c> the backend produced on its own is a failure like any other, nothing here asked for it.
    /// Token tells the two apart, and is checked alongside the code rather than instead of it.
    /// </summary>
    [Fact]
    public void ACancellationNobodyAskedForIsStillASentence()
    {
        var failure = ControlBackend.Translate(
            new RpcException(new Status(StatusCode.Cancelled, "")), CancellationToken.None);

        var unavailable = Assert.IsType<BackendUnavailableException>(failure);
        Assert.Contains("Cancelled", unavailable.Message);
    }
}
