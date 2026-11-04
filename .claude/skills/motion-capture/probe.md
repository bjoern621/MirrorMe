# Headless probe

Runs the real `App` with every `Design/` style and samples a control's animated values, on a copy of the tree.
A copy holds while another session edits the checkout.
Test project holds no headless host, so a probe stays throwaway.

1. Run `.claude/skills/motion-capture/scripts/probe.sh <scratch>` from the repository root.
   It archives `HEAD` to `<scratch>/tree` and writes `<scratch>/probe`.
2. Copy uncommitted files of your own over `<scratch>/tree`.
3. Edit the scenario in `<scratch>/probe/Program.cs` to the sequence the recording showed.
   Mount a screen's real view, since a bare `TileCard` hid a double fade the real tree showed.
4. Run `dotnet run -c Debug` in `<scratch>/probe`.

The first mount runs during JIT, so read a second mount for timing.

## Reading a replay

A value reaching its end and starting again, with no class, layout or view model event at that moment, means two `Style.Animations` of one property on one control.
Avalonia plays them back to back.
Move the second to another element or another property.
A `:not(.class)` on the first replays it when the class goes.

An exception reading "No animator registered for the property RenderTransform" means `Controls/Motion/TransformKeyFrames.cs` did not run.

`UseHeadlessDrawing = false` plus `UseSkia()` make hit testing and `RenderTargetBitmap` work.
