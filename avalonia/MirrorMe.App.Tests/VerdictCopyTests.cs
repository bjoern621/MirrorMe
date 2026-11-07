using MirrorMe.Api.V1;
using MirrorMe.App.Copy;
using Xunit;

namespace MirrorMe.App.Tests;

public sealed class VerdictCopyTests
{
    [Fact]
    public void ADipNamesTheRateReachedAndTheOneAskedFor()
    {
        var dip = new Text { Code = TextCode.VerdictDip };
        dip.Args.Add(new TextArg { Name = TextArgName.Fps, Decimal = 41.6 });
        dip.Args.Add(new TextArg { Name = TextArgName.TargetFps, Decimal = 60 });

        Assert.Equal("Sending 42 of the 60 frames a second the stream asks for. Motion looks uneven to viewers.",
            Statements.Of(dip));
    }
}
