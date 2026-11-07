using Grpc.Core;
using MirrorMe.Api.V1;
using MirrorMe.App.Backend;
using Xunit;

namespace MirrorMe.App.Tests;

/// <summary>
/// What a refused frame subscription turns into on a tile.
/// The backend answers the subscribe on the response stream, so the refusal surfaces on the first read.
/// </summary>
public sealed class FrameRefusalTests
{
    /// <summary>A tile prints the refusal's sentence, as every unary call's refusal reads.</summary>
    [Fact]
    public async Task ARefusalTheBackendServedReadsAsItsSentence()
    {
        var text = new Text { Code = TextCode.FramesUndrawable };
        text.Args.Add(new TextArg { Name = TextArgName.Stream, Id = "Björn/test-1" });
        text.Args.Add(new TextArg { Name = TextArgName.Transport, Id = "rtsp" });
        var refusal = new Refusal { Text = text };
        var client = new FrameService.FrameServiceClient(
            new RefusingInvoker(Served.Refusal(StatusCode.FailedPrecondition, refusal)));

        await using var channel = await FrameChannel.OpenAsync(client, "Björn/test-1", "rtsp", CancellationToken.None);

        var failure = await Assert.ThrowsAsync<BackendUnavailableException>(async () =>
        {
            await foreach (var _ in channel.ReadAsync(CancellationToken.None))
            {
            }
        });
        Assert.Equal(Copy.Refusals.Of(refusal), failure.Message);
    }

    /// <summary>Opens a frame call whose subscribe is taken and whose first read fails with one status.</summary>
    private sealed class RefusingInvoker(RpcException refusal) : CallInvoker
    {
        public override AsyncDuplexStreamingCall<TRequest, TResponse> AsyncDuplexStreamingCall<TRequest, TResponse>(
            Method<TRequest, TResponse> method, string? host, CallOptions options)
            => new(
                new DiscardingWriter<TRequest>(),
                new RefusingReader<TResponse>(refusal),
                Task.FromResult(new Metadata()),
                () => refusal.Status,
                () => new Metadata(),
                () => { });

        public override TResponse BlockingUnaryCall<TRequest, TResponse>(
            Method<TRequest, TResponse> method, string? host, CallOptions options, TRequest request)
            => throw new NotSupportedException();

        public override AsyncUnaryCall<TResponse> AsyncUnaryCall<TRequest, TResponse>(
            Method<TRequest, TResponse> method, string? host, CallOptions options, TRequest request)
            => throw new NotSupportedException();

        public override AsyncServerStreamingCall<TResponse> AsyncServerStreamingCall<TRequest, TResponse>(
            Method<TRequest, TResponse> method, string? host, CallOptions options, TRequest request)
            => throw new NotSupportedException();

        public override AsyncClientStreamingCall<TRequest, TResponse> AsyncClientStreamingCall<TRequest, TResponse>(
            Method<TRequest, TResponse> method, string? host, CallOptions options)
            => throw new NotSupportedException();
    }

    private sealed class DiscardingWriter<T> : IClientStreamWriter<T>
    {
        public WriteOptions? WriteOptions { get; set; }

        public Task WriteAsync(T message) => Task.CompletedTask;

        public Task CompleteAsync() => Task.CompletedTask;
    }

    private sealed class RefusingReader<T>(RpcException refusal) : IAsyncStreamReader<T>
    {
        public T Current => throw new InvalidOperationException("a refused call carries no message");

        public Task<bool> MoveNext(CancellationToken cancellationToken) => Task.FromException<bool>(refusal);
    }
}
