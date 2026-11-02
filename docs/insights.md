# Insights

Backend judges this machine's publish run the way a viewer would see it.
Every shell draws that judgment and decides none of it.

| Verdict | When |
| --- | --- |
| clean | no run in force, or the encoder holds its frame rate |
| disappoints | output under 80 % of the target for two readings in a row, or a pipeline ended with a retry relaunching it |
| stopped | the run ended on a failure no retry recovers |

A dip ends once output climbs back to 90 % of the target.
Between the two thresholds the verdict holds, so a rate hovering at one line does not flicker.
A restart ends with the first reading of the relaunched pipeline.

Rate read: frames the encoder emitted.
Capture rate stays out, a still screen producing no new pictures while the viewer sees nothing wrong.

A stopped run carries the pipeline's own words and its log path, and keeps them until the next start.
A stop the user asked for clears the run.

## Marks

Each run keeps the marks that moved it, oldest first, the most recent 200.

| Mark | Carries |
| --- | --- |
| dip | when it began, the lowest rate it reached and the target |
| restart | when the pipeline ended |
| join, leave | the member, against the group's first reading of the run |

`Read stream_timeline` reads the run and `stream_timeline` pushes it whole on every change.

## Screen

Header: what is shared and to which group, the way it travels, the running time and `Edit in setup`.
Under it the verdict, then two tabs.

Stream tab: the preview, a card per figure and a chart of the picked card over one minute, five minutes or the whole run.
Each card holds its newest value against its limit, and the chart marks every moment the run kept.
The chart's axis spans two round steps around the readings and the limit, and a stretch past the limit draws in amber.

| Figure | Limit |
| --- | --- |
| frame rate | the target frame rate |
| encode time | one frame's duration at the target rate |
| bitrate | the encode's rate ceiling, none where nothing bounds it |

Viewers tab: a row per viewer the relay names, with round trip, loss and dropped wherever its leg measures them (`field-availability.md`).
