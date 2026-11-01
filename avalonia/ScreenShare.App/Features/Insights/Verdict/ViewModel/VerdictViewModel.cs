using ScreenShare.Api.V1;
using ScreenShare.App.Contracts;
using ScreenShare.App.Copy;
using ScreenShare.App.Features.Insights.Model;
using ScreenShare.App.Mvvm;
using TablerIcons;

namespace ScreenShare.App.Features.Insights.Verdict.ViewModel;

public enum VerdictTone
{
    Quiet,
    Caution,
    Stopped,
}

/// <summary>
/// The verdict line: how the stream is doing, in a sentence, as the backend judged it (<c>docs/insights.md</c>).
/// Grey while clean or idle, amber while running and disappointing, red once stopped.
/// A stopped run carries the pipeline's own words and its log, the one place this screen reaches a log from.
/// </summary>
public sealed class VerdictViewModel : Observable
{
    private readonly Func<string, Task> _openLog;

    public VerdictViewModel(Func<string, Task> openLog, Action<Action> dispatch)
    {
        Assert.NotNull(openLog, "a stopped verdict opens the log it names");
        Assert.NotNull(dispatch, "a verdict's command marshals onto the UI loop");

        _openLog = openLog;
        OpenLogCommand = new PendingCommand(() => _openLog(_logPath), dispatch, () => HasLog);
    }

    private VerdictTone _tone;
    private Icons _glyph = Icons.IconCircleCheck;
    private string _headline = InsightsCopy.NotSharing;
    private string _detail = InsightsCopy.NotSharingDetail;
    private string _message = "";
    private string _logPath = "";
    private bool _hasLog;

    public VerdictTone Tone
    {
        get => _tone;
        private set
        {
            if (Set(ref _tone, value))
            {
                OnPropertyChanged(nameof(IsCaution));
                OnPropertyChanged(nameof(IsStopped));
            }
        }
    }

    public bool IsCaution => Tone == VerdictTone.Caution;

    public bool IsStopped => Tone == VerdictTone.Stopped;

    public Icons Glyph { get => _glyph; private set => Set(ref _glyph, value); }

    public string Headline { get => _headline; private set => Set(ref _headline, value); }

    public string Detail { get => _detail; private set => Set(ref _detail, value); }

    /// <summary>The pipeline's own last words, empty unless a run stopped or restarts.</summary>
    public string Message { get => _message; private set => Set(ref _message, value); }

    public bool HasMessage => Message.Length > 0;

    public bool HasLog { get => _hasLog; private set => Set(ref _hasLog, value); }

    public static string OpenLogLabel => InsightsCopy.OpenFullLog;

    public PendingCommand OpenLogCommand { get; }

    /// <summary>
    /// The one render function.
    /// timeline: the backend's judgment, null before the first read.
    /// reading: the publish state, for the attempt counter a restart names.
    /// </summary>
    public void Apply(StreamTimeline? timeline, InsightsSnapshot reading)
    {
        Assert.NotNull(reading, "a verdict reads the publish state beside the timeline");

        var verdict = timeline?.Verdict;
        var level = verdict?.Level ?? StreamVerdict.Types.Level.Clean;
        var cause = Statements.Of(verdict?.Cause);

        switch (level)
        {
            case StreamVerdict.Types.Level.Stopped:
                Write(VerdictTone.Stopped, Icons.IconCircleX, InsightsCopy.Stopped, cause, verdict!.Message, verdict.LogPath);
                break;
            case StreamVerdict.Types.Level.Disappoints when reading.IsRetrying:
                Write(VerdictTone.Caution, Icons.IconAlertTriangle, InsightsCopy.Restarting(reading.Attempt, reading.Budget),
                    cause, reading.RetryMessage, "");
                break;
            case StreamVerdict.Types.Level.Disappoints:
                Write(VerdictTone.Caution, Icons.IconAlertTriangle, InsightsCopy.UnderTarget, cause, "", "");
                break;
            default:
                Write(VerdictTone.Quiet, reading.IsLive ? Icons.IconCircleCheck : Icons.IconPlayerStop,
                    reading.IsLive ? InsightsCopy.RunningClean : InsightsCopy.NotSharing,
                    reading.IsLive ? InsightsCopy.RunningCleanDetail : InsightsCopy.NotSharingDetail, "", "");
                break;
        }

        OpenLogCommand.Refresh();

        Assert.That(Headline.Length > 0, "a verdict always says how the stream is doing", Tone);
        Assert.That(!HasLog || IsStopped, "a log is offered on a stopped run alone", Tone, _logPath);
    }

    private void Write(VerdictTone tone, Icons glyph, string headline, string detail, string message, string logPath)
    {
        Tone = tone;
        Glyph = glyph;
        Headline = headline;
        Detail = detail;
        Message = message;
        OnPropertyChanged(nameof(HasMessage));
        _logPath = logPath;
        HasLog = logPath.Length > 0;
    }
}
