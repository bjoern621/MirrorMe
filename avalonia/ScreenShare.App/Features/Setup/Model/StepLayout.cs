using ScreenShare.App.Contracts;

namespace ScreenShare.App.Features.Setup.Model;

/// <summary>
/// Groups drawn inside another group's step rather than on a step of their own.
///
/// How the stream reaches the relay answers part of who can watch it, so the sending group stands under the relay
/// group on one step, and setup walks four steps.
/// Placement alone (<c>docs/ipc-api.md</c>, "The rule"): a hosted group keeps its own fields, order and verdicts,
/// and one whose host the form does not carry falls back to a step of its own.
/// </summary>
public static class StepLayout
{
    private static readonly IReadOnlyDictionary<string, string> Hosts = new Dictionary<string, string>
    {
        ["transport"] = RelayLayout.GroupKey,
    };

    /// <summary>Step key that draws this group: its host where it has one, the group itself otherwise.</summary>
    public static string HostOf(string groupKey)
    {
        Assert.That(groupKey.Length > 0, "placing a group names the group being placed");

        return Hosts.TryGetValue(groupKey, out var host) ? host : groupKey;
    }

    /// <summary>Group this step draws under its own, empty where it hosts none.</summary>
    public static string HostedOn(string stepKey)
    {
        Assert.That(stepKey.Length > 0, "a hosted group is looked up by the step drawing it");

        return Hosts.FirstOrDefault(entry => entry.Value == stepKey).Key ?? "";
    }
}
