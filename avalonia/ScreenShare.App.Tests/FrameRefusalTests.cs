using Grpc.Core;
using ScreenShare.Api.V1;
using ScreenShare.App.Backend;
using Xunit;

namespace ScreenShare.App.Tests;

/// <summary>
/// What a refused frame subscription turns into on a tile.
/// The backend answers the subscribe on the response stream, so the refusal surfaces on the first read.
/// </summary>
public sealed class FrameRefusalTests
{
    /// <summary>
    /// A tile prints the sentence the backend served, as every unary call's refusal does.
    /// The status wrapper around it names the transport and nothing the reader can act on.
    /// </summary>
    [Fact]
    public async Task ARefusalTheBackendServedKeepsItsOwnSentence()
    {
        const string served = "cannot draw 'Björn/test-1' over rtsp: nothing is decoding Björn/test-1 over rtsp";
        var client = new FrameService.FrameServiceClient(
            new RefusingInvoker(new Status(StatusCode.FailedPrecondition, served)));

        await using var channel = await FrameChannel.OpenAsync(client, "Björn/test-1", "rtsp", CancellationToken.None);

        var failure = await Assert.ThrowsAsync<BackendUnavailableException>(async () =>
        {
            await foreach (var _ in channel.ReadAsync(CancellationToken.None))
            {
            }
        });
        Assert.Equal(served, failure.Message);
    }

    /// <summary>Opens a frame call whose subscribe is taken and whose first read fails with one status.</summary>
    private sealed class RefusingInvoker(Status refusal) : CallInvoker
    {
        public override AsyncDuplexStreamingCall<TRequest, TResponse> AsyncDuplexStreamingCall<TRequest, TResponse>(
            Method<TRequest, TResponse> method, string? host, CallOptions options)
            => new(
                new DiscardingWriter<TRequest>(),
                new RefusingReader<TResponse>(refusal),
                Task.FromResult(new Metadata()),
                () => refusal,
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

    private sealed class RefusingReader<T>(Status refusal) : IAsyncStreamReader<T>
    {
        public T Current => throw new InvalidOperationException("a refused call carries no message");

        public Task<bool> MoveNext(CancellationToken cancellationToken) => Task.FromException<bool>(new RpcException(refusal));
    }
}
