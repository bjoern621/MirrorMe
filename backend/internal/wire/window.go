package wire

import (
	"strconv"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/window"
)

// ShareWindows carries the window enumeration out.
//
// The handle crosses as decimal text, which is what PublishSettings.share_window stores
// and what the form offers as an option value:
// the platform handle is 64 bits wide, and one spelling on both sides is what lets a shell
// match a picked value against the row describing it.
func ShareWindows(windows []window.Window) []*mirrormev1.ShareWindow {
	out := make([]*mirrormev1.ShareWindow, 0, len(windows))
	for _, w := range windows {
		out = append(out, &mirrormev1.ShareWindow{
			Handle: strconv.FormatUint(w.Handle, 10),
			Title:  w.Title,
			App:    w.App,
			Width:  int32(w.Width),
			Height: int32(w.Height),
		})
	}
	return out
}
