using ScreenShare.App.Features.Fields.ViewModel;

namespace ScreenShare.App.Features.Setup.QualityStep.ViewModel;

/// <summary>
/// One question of the encode step: the control it asks about, under the sentence asking it.
/// A record over the shared field view model, so a pass over an unchanged group builds rows equal to the last.
/// </summary>
public sealed record QuestionRow
{
    public required string Heading { get; init; }

    public required FieldViewModel Field { get; init; }
}
