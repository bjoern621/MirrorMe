namespace MirrorMe.App.Features.Setup.Model;

/// <summary>
/// Uplink field, the one key the setup flow names outside the encode step.
///
/// Naming a key is placement, the licence <see cref="QualityLayout"/> uses for its questions.
/// What the field is called, what it holds and whether it is reachable stays the form's answer.
/// The measurement rides beside that control, being an effect rather than a value
/// (<c>Features/Fields/Model/FieldAction.cs</c>).
/// </summary>
public static class UplinkLayout
{
    /// <summary>Upload capacity the prediction is measured against, Mb/s.</summary>
    public const string UplinkKey = "publish.uplink_mbps";
}
