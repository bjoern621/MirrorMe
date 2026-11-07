package form

import (
	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"

	"bjoernblessin.de/mirrorme/internal/gpupath"
	"bjoernblessin.de/mirrorme/internal/wire"
)

// staysOnDevice reports whether this run hands the encoder the frames the capture produced,
// without a trip through system memory.
//
// It reads the row and the value together, the two facts gpupath.Resolve reads.
// Auto reads the colour verdict as well: it takes a device path only where that path costs nothing,
// so a pair whose only path leaves the conversion to the encoder downloads under auto.
func (av availability) staysOnDevice() bool {
	if !av.onDevicePath || av.s.Publish.CaptureMemory == gpupath.MemorySystem {
		return false
	}
	return !(av.s.Publish.CaptureMemory == gpupath.MemoryAuto && av.path.Colour.TradesColour())
}

// frameMemoryNote states which import carries the frames on the direct path,
// and on the path that converts nothing, what that costs.
// The field that chose the trade states the cost beside the import,
// rather than leaving it to the two colour fields it overrides.
func (av availability) frameMemoryNote() *mirrormev1.Text {
	if !av.staysOnDevice() {
		return nil
	}
	return say(av.path.Import, argCost(wire.GpuPathCost(av.path)))
}

// frameMemoryReason states why this capture backend
// and codec cannot publish through a frame memory.
//
// Both deciding facts come off the pair table: whether the pair has a device path at all,
// and what that path does to the colour (docs/field-availability.md).
// A greyed device value carries the cost it would have paid,
// so the value that accepts that cost stays live.
//
// Auto and the system copy are never greyed: auto answers with whichever path costs the user
// nothing and system memory is the path every pair has.
//
// Every statement names both ends, since neither decides on its own and switching either side
// is a way to reach the path.
func (av availability) frameMemoryReason(memory string) *mirrormev1.Text {
	if av.engine == "" {
		return nil
	}
	if !av.onDevicePath {
		if memory != gpupath.MemoryGpu && memory != gpupath.MemoryGpuEncoderColor {
			return nil
		}
		return say(pairHasNoDeviceMemory,
			argCapture(av.s.Publish.Capture), argCodec(av.s.Publish.Codec()), argEngine(av.engine))
	}
	if !av.path.Colour.TradesColour() {
		if memory != gpupath.MemoryGpuEncoderColor {
			return nil
		}
		return say(pairConvertsOnDevice,
			argCapture(av.s.Publish.Capture), argCodec(av.s.Publish.Codec()), argMemory(gpupath.MemoryGpu))
	}
	if memory != gpupath.MemoryGpu {
		return nil
	}
	return say(pairTradesColour,
		argCapture(av.s.Publish.Capture),
		argCodec(av.s.Publish.Codec()),
		argEngine(av.engine),
		argCost(wire.GpuPathCost(av.path)),
		argReach(availabilityExactColourReach(av.path.Family)))
}

// outputResolutionReason states why this pair cannot be asked for a scaled picture.
//
// One case has it, and it is about where the frames are rather than about the size.
// On a device path the only thing that can resize them is the filter already on that path,
// and a colour-trading row has none: the encoder reads the captured surfaces directly.
// The ffmpeg builder refuses such a run, so the entry greys with the same fact rather
// than being offered into a refusal.
//
// Having such a row is not enough, the run has to be taking it, which staysOnDevice answers.
// The statement offers the system copy as the way across, so the same fact on a run that already
// downloads every frame would name a fix already applied
// and grey a scale that path's CPU filter can perform.
//
// The source size is never greyed: it is what every pair does,
// and a control with every entry out teaches nothing.
func (av availability) outputResolutionReason(value string) *mirrormev1.Text {
	if value == "" || !av.staysOnDevice() || !av.path.Colour.TradesColour() {
		return nil
	}
	return say(devicePathHasNoScaler,
		argCapture(av.s.Publish.Capture), argCodec(av.s.Publish.Codec()), argMemory(gpupath.MemorySystem))
}

// availabilityExactColourReach states how an encoder family is reached on the device
// with the colour the form shows, nil where the pair table declares no such row.
//
// It is the way out the greyed direct value carries, and the one the dropdown cannot show
// by itself: the fix is another capture backend rather than another value of this field.
// Assembled from the row, so it names whichever pair
// the table declares rather than a platform written in here.
func availabilityExactColourReach(family string) *mirrormev1.Text {
	for _, p := range gpupath.Paths {
		if p.Family != family || p.Colour.TradesColour() {
			continue
		}
		return say(exactColourReach, argCapture(p.Capture), argEngine(p.Engine), argImport(say(p.Import)))
	}
	return nil
}
