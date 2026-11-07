//go:build !windows && !linux

package receive

import (
	"runtime"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
	"bjoernblessin.de/mirrorme/internal/text"

	"github.com/go-gst/go-gst/pkg/gst"
)

// The export leg of a platform whose handle type is not built.
//
// Refuses rather than falling back to a download.
// A tile fed from system memory would work and cost gigabytes a second,
// the copy the frame channel exists to avoid (docs/viewer-architecture.md, "The frame channel").
// The refusal names the platform,
// and the viewer it points at is the native player, which needs no frame channel at all.
//
// macOS is the leg that is left, and it is IOSurface out of VideoToolbox.

type unbuiltSharer struct{}

func newSharer() sharer { return unbuiltSharer{} }

func (unbuiltSharer) open(*gst.Sample, int) (Pool, error) {
	return Pool{}, text.Fail(mirrormev1.TextCode_TEXT_CODE_FRAMES_NEED_PLAYER, text.Os(runtime.GOOS))
}

func (unbuiltSharer) write(int, *gst.Sample) error {
	return text.Fail(mirrormev1.TextCode_TEXT_CODE_FRAMES_UNSHARED)
}

func (unbuiltSharer) close() {}
