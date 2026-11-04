package receive

/*
#cgo pkg-config: gstreamer-video-1.0
#include <stdlib.h>
#include <gst/video/video.h>

// pixel_shape takes a raw video format name, e.g. "Y444_10LE", and fills in bit depth per
// component and the two chroma subsampling factors.
// Factor 1 is full chroma resolution on that axis.
// 0 where this GStreamer build knows no such name.
static int pixel_shape(const char *name, int *depth, int *h_sub, int *v_sub) {
	GstVideoFormat fmt = gst_video_format_from_string(name);
	if (fmt == GST_VIDEO_FORMAT_UNKNOWN) {
		return 0;
	}
	const GstVideoFormatInfo *info = gst_video_format_get_info(fmt);
	*depth = GST_VIDEO_FORMAT_INFO_DEPTH(info, 0);
	*h_sub = 1 << GST_VIDEO_FORMAT_INFO_W_SUB(info, 1);
	*v_sub = 1 << GST_VIDEO_FORMAT_INFO_H_SUB(info, 1);
	return 1;
}

// dma_drm_format takes a drm-format caps field, e.g. "NV12:0x0200000010401b04", and returns
// the raw video format with the same planes and depth.
// The modifier names a tiling, which gst_video_dma_drm_format_to_gst_format maps to no format,
// so the fourcc alone decides.
// NULL where this GStreamer build maps the fourcc to none.
static const char *dma_drm_format(const char *drm) {
	guint64 modifier = 0;
	guint32 fourcc = gst_video_dma_drm_fourcc_from_string(drm, &modifier);
	GstVideoFormat fmt = gst_video_dma_drm_fourcc_to_format(fourcc);
	if (fmt == GST_VIDEO_FORMAT_UNKNOWN) {
		return NULL;
	}
	return gst_video_format_to_string(fmt);
}
*/
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/go-gst/go-gst/pkg/gst"
)

// padCaps is what one of an element's static pads negotiated, nil while either the pad or
// the negotiation is absent.
func padCaps(e gst.Element, name string) *gst.Caps {
	pad := e.GetStaticPad(name)
	if pad == nil {
		return nil
	}
	caps := pad.GetCurrentCaps()
	if caps == nil || caps.GetSize() == 0 {
		return nil
	}
	return caps
}

// codecDescription is GStreamer's own reading of caps, e.g. "H.265 (Main 4:4:4 profile)".
// The media type stands in where pbutils describes nothing, leaving such a codec to name itself.
func codecDescription(caps *gst.Caps) string {
	if d := pbUtilsCodecDescription(caps); d != "" {
		return d
	}
	return caps.GetStructure(0).GetName()
}

// pixelShape is a raw video format on the axes a viewer cares about: bit depth per component, and
// chroma subsampling.
// GStreamer's own format table supplies both, so a format this build knows
// needs no entry here.
func pixelShape(format string) (depth int, subsampling string) {
	if format == "" {
		return 0, ""
	}
	name := C.CString(format)
	defer C.free(unsafe.Pointer(name))
	var d, h, v C.int
	if C.pixel_shape(name, &d, &h, &v) == 0 {
		return 0, ""
	}
	return int(d), subsamplingLabel(int(h), int(v))
}

// rawFormat reads the raw video format a caps structure carries.
// dmabuf caps carry DMA_DRM there and their layout in drm-format, which this reads instead.
// "" where the structure names neither.
func rawFormat(st *gst.Structure) string {
	format := st.GetString("format")
	if format != dmaDrmFormat {
		return format
	}
	drm := st.GetString("drm-format")
	if drm == "" {
		return ""
	}
	name := C.CString(drm)
	defer C.free(unsafe.Pointer(name))
	raw := C.dma_drm_format(name)
	if raw == nil {
		return ""
	}
	return C.GoString(raw)
}

// dmaDrmFormat stands in the format field of dmabuf caps, which name their layout in drm-format.
const dmaDrmFormat = "DMA_DRM"

// subsamplingLabel writes chroma subsampling factors as J:a:b: a region four luma samples wide,
// a chroma samples across it, and b for the second row, which vertical subsampling takes to zero.
func subsamplingLabel(h, v int) string {
	if h <= 0 || v <= 0 || 4%h != 0 {
		return ""
	}
	a := 4 / h
	b := a
	if v > 1 {
		b = 0
	}
	return fmt.Sprintf("4:%d:%d", a, b)
}
