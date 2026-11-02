using Google.Protobuf;
using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using ScreenShare.Api.V1;

namespace ScreenShare.App.Tests;

/// <summary>A status as the backend serves one: the code's name for a message, the Refusal as its one detail.</summary>
internal static class Served
{
    public static RpcException Refusal(StatusCode code, Refusal refusal)
    {
        var status = new Google.Rpc.Status { Code = (int)code, Message = refusal.Text.Code.ToString() };
        status.Details.Add(Any.Pack(refusal));
        var trailers = new Metadata { { "grpc-status-details-bin", status.ToByteArray() } };
        return new RpcException(new Status(code, status.Message), trailers);
    }
}
