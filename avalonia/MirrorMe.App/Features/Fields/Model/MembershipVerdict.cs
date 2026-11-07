using MirrorMe.Api.V1;
using MirrorMe.App.Contracts;
using MirrorMe.App.Controls;

namespace MirrorMe.App.Features.Fields.Model;

/// <summary>A line under one control saying what a service past the form made of its value.</summary>
public sealed record FieldVerdict(string Text, CheckState State);

/// <summary>
/// What the group service made of the key and the name, drawn under the box each one was typed in.
///
/// Placement, on the licence <see cref="ClipboardPlacement"/> takes: the backend states membership as one fact
/// about this machine (<c>MembersState</c>), and which box a reader fixes it in is this side's to say.
/// A presence the service took lists this machine's own row, the one proof that the key and the name both landed.
/// </summary>
public static class MembershipVerdict
{
    private const string KeyField = "relay.group_key";
    private const string NameField = "relay.display_name";

    /// <summary>The line under one control, null where membership says nothing about it.</summary>
    public static FieldVerdict? Of(string key, MembersState? members)
    {
        Assert.That(key.Length > 0, "a verdict is placed under the settings field it is about");

        if (members is null)
        {
            return null;
        }

        if (members.Refusal is { } refusal)
        {
            return FixedIn(refusal.Code) == key ? new FieldVerdict(Copy.Statements.Of(refusal), CheckState.Blocking) : null;
        }

        if (!members.Members.Any(member => member.Self))
        {
            return null;
        }

        return key switch
        {
            KeyField => new FieldVerdict(Copy.Fields.Joined(members.Members.Count - 1), CheckState.Passed),
            NameField => new FieldVerdict(Copy.Fields.NameClaimed, CheckState.Passed),
            _ => null,
        };
    }

    /// <summary>
    /// The box a refusal is fixed in, empty for one no box answers.
    /// A missing name stays out: the form anchors that line on the name itself.
    /// </summary>
    private static string FixedIn(TextCode code) => code switch
    {
        TextCode.GroupNameTaken => NameField,
        TextCode.GroupServiceRefused => KeyField,
        _ => "",
    };
}
