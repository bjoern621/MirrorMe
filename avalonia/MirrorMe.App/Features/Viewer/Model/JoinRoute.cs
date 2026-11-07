namespace MirrorMe.App.Features.Viewer.Model;

/// <summary>A way into a group the join card offers. The card leads to the control that sets it.</summary>
public enum JoinRoute
{
    /// <summary>The group follows the voice channel; Settings links the account first.</summary>
    Discord,

    /// <summary>A group key, set on the relay step.</summary>
    Key,
}
