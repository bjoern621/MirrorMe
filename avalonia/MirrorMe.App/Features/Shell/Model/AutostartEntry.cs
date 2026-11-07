using System.Runtime.Versioning;
using System.Text;
using Avalonia.Logging;
using Microsoft.Win32;

namespace MirrorMe.App.Features.Shell.Model;

/// <summary>
/// The entry the desktop starts this shell from when the user signs in, written and removed with the app setting
/// (<c>avalonia/README.md</c>, "Starting at sign-in").
///
/// Linux reads an XDG autostart file, Windows the current user's <c>Run</c> key.
/// Every other platform carries no entry, and the setting reaches nothing there.
///
/// Converging rewrites the entry on every start that asks for one, so an install that moved or updated points
/// the next sign-in at the executable running now.
/// </summary>
public static class AutostartEntry
{
    /// <summary>
    /// <c>0</c> leaves the entry alone for a whole run, neither written nor removed, whatever the setting holds.
    ///
    /// For a checkout run beside an installed app, whose entry a build under development would otherwise
    /// point at itself.
    /// The backend reads it too and greys the app setting behind it
    /// (<c>internal/control</c>, <c>EnvAutostart</c>).
    /// </summary>
    internal const string EnvAutostart = "MIRRORME_AUTOSTART";

    /// <summary>Name of the <c>Run</c> value on Windows and of the autostart file on Linux.</summary>
    private const string Name = "MirrorMe";

    private const string RunKey = @"Software\Microsoft\Windows\CurrentVersion\Run";

    /// <summary>
    /// Writes the entry where <paramref name="wanted"/> asks for one and removes it where it does not.
    /// Null, settings not read yet, touches nothing: removing an entry before the read would undo a choice
    /// the stored settings still hold.
    /// A second call with the same value changes nothing.
    /// </summary>
    public static void Converge(bool? wanted)
    {
        if (wanted is not { } want || !IsManaged(Environment.GetEnvironmentVariable(EnvAutostart)))
        {
            return;
        }

        // A home directory or hive refusing the write costs the sign-in start and nothing else,
        // so it is reported rather than thrown.
        try
        {
            if (OperatingSystem.IsWindows())
            {
                ConvergeRunKey(want);
            }
            else if (OperatingSystem.IsLinux())
            {
                ConvergeDesktopFile(want);
            }
        }
        catch (Exception failed) when (failed is IOException or UnauthorizedAccessException)
        {
            Logger.TryGet(LogEventLevel.Warning, "Autostart")?.Log(
                null, "the sign-in entry stays as it was: {Error}", failed.Message);
        }
    }

    /// <summary>
    /// Whether a run writes and removes the entry.
    /// <see cref="EnvAutostart"/> holds it alone at <c>0</c>, and the setting decides everywhere else.
    /// </summary>
    internal static bool IsManaged(string? environment) => environment != "0";

    [SupportedOSPlatform("windows")]
    private static void ConvergeRunKey(bool want)
    {
        using var run = Registry.CurrentUser.CreateSubKey(RunKey);
        if (want)
        {
            run.SetValue(Name, $"\"{Environment.ProcessPath}\"");
        }
        else
        {
            run.DeleteValue(Name, throwOnMissingValue: false);
        }
    }

    private static void ConvergeDesktopFile(bool want)
    {
        var path = Path.Combine(
            AutostartDirectory(
                Environment.GetEnvironmentVariable("FLATPAK_ID"),
                Environment.GetEnvironmentVariable("HOST_XDG_CONFIG_HOME"),
                Environment.GetEnvironmentVariable("XDG_CONFIG_HOME"),
                Environment.GetFolderPath(Environment.SpecialFolder.UserProfile)),
            Name.ToLowerInvariant() + ".desktop");

        if (!want)
        {
            File.Delete(path);
            return;
        }

        var (exec, tryExec) = LinuxCommand(
            Environment.ProcessPath ?? "", Environment.GetEnvironmentVariable("FLATPAK_ID"));
        var entry = DesktopEntry(exec, tryExec);
        if (File.Exists(path) && File.ReadAllText(path) == entry)
        {
            return;
        }

        Directory.CreateDirectory(Path.GetDirectoryName(path)!);
        File.WriteAllText(path, entry);
    }

    /// <summary>
    /// The directory a session's autostart reads, which inside a Flatpak sandbox is the host's:
    /// the sandbox points <c>XDG_CONFIG_HOME</c> at a directory of the app's own that no session reads.
    /// A relative <c>XDG_CONFIG_HOME</c> counts as unset, as the base directory spec asks.
    /// </summary>
    internal static string AutostartDirectory(string? flatpakId, string? hostConfig, string? config, string home)
    {
        var configHome = string.IsNullOrEmpty(flatpakId) ? config : hostConfig;
        if (string.IsNullOrEmpty(configHome) || !Path.IsPathRooted(configHome))
        {
            configHome = PosixJoin(home, ".config");
        }

        return PosixJoin(configHome, "autostart");
    }

    /// <summary>
    /// Joins with <c>/</c> on every host, where <see cref="Path.Combine(string, string)"/> takes <c>\</c> on Windows
    /// and breaks the Linux path that the Windows test run checks.
    /// </summary>
    private static string PosixJoin(string directory, string name) => directory.TrimEnd('/') + "/" + name;

    /// <summary>
    /// What the entry starts, and the program whose absence makes the session skip the entry.
    ///
    /// A Flatpak starts through <c>flatpak run</c>, the executable inside the sandbox being no path on the host.
    /// A Nix store path changes with every update and skips the wrapper that puts the backend on <c>PATH</c>,
    /// so a store install starts the <c>mirrorme</c> its profile puts there.
    /// Everything else starts the executable running now.
    /// </summary>
    internal static (string Exec, string? TryExec) LinuxCommand(string processPath, string? flatpakId)
    {
        if (!string.IsNullOrEmpty(flatpakId))
        {
            return ($"flatpak run {flatpakId}", null);
        }

        if (processPath.StartsWith("/nix/store/", StringComparison.Ordinal))
        {
            return ("mirrorme", "mirrorme");
        }

        return (ExecArgument(processPath), processPath);
    }

    /// <summary>The XDG autostart file, read by the session at sign-in.</summary>
    internal static string DesktopEntry(string exec, string? tryExec)
    {
        var entry = new StringBuilder()
            .Append("[Desktop Entry]\n")
            .Append("Type=Application\n")
            .Append("Name=MirrorMe\n")
            .Append("Exec=").Append(exec).Append('\n');
        if (tryExec is not null)
        {
            entry.Append("TryExec=").Append(EscapeString(tryExec)).Append('\n');
        }

        return entry.Append("Terminal=false\n").ToString();
    }

    /// <summary>
    /// One argument of an <c>Exec</c> key, quoted where it holds a character the key reserves.
    /// Escaped twice: once for the quoting, then once more for the string value carrying it,
    /// as the desktop entry spec asks.
    /// </summary>
    internal static string ExecArgument(string argument)
    {
        var escaped = argument.Replace("%", "%%");
        if (escaped.IndexOfAny(ExecReserved) < 0)
        {
            return escaped;
        }

        var quoted = new StringBuilder("\"");
        foreach (var c in escaped)
        {
            if (c is '"' or '`' or '$' or '\\')
            {
                quoted.Append('\\');
            }

            quoted.Append(c);
        }

        return EscapeString(quoted.Append('"').ToString());
    }

    private static readonly char[] ExecReserved =
        [' ', '\t', '\n', '"', '\'', '\\', '>', '<', '~', '|', '&', ';', '$', '*', '?', '#', '(', ')', '`'];

    private static string EscapeString(string value) => value.Replace("\\", "\\\\");
}
