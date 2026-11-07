package receive

/*
#cgo pkg-config: gstreamer-1.0
#include <stdlib.h>
#include <gst/gst.h>

static guint64 decoderFactoryType(void) { return GST_ELEMENT_FACTORY_TYPE_DECODER; }

// selectStreams hands the element a select-streams event naming ids, which it takes ownership of.
static gboolean selectStreams(GstElement *element, char **ids, int n) {
	GList *list = NULL;
	for (int i = 0; i < n; i++)
		list = g_list_append(list, ids[i]);
	GstEvent *event = gst_event_new_select_streams(list);
	g_list_free(list);
	return gst_element_send_event(element, event);
}
*/
import "C"

import (
	"runtime"
	"slices"
	"unsafe"

	"github.com/go-gst/go-glib/pkg/gobject/v2"
	"github.com/go-gst/go-gst/pkg/gst"

	"bjoernblessin.de/go-utils/util/assert"
	"bjoernblessin.de/go-utils/util/logger"
)

// decodebin3 races its own caps update when a stream starts (GStreamer 1.28.7).
// The multiqueue thread sets up the stream's decoder from the caps tsdemux opened it with,
// "video/x-h264, stream-format=byte-stream" on SRT, while the input thread swaps in the parsed caps.
// No decoder takes caps without an alignment, so decodebin3 drops the stream from its selection and
// never takes it back: the decode plays sound and shows no picture.
// A loaded machine loses the race on about one SRT join in four.
//
// The selection decodebin3 posts after the drop lets a dropped stream back in, once its caps
// name a decoder.
// Those caps can land after the selection does, so a dropped stream short of them is watched until
// they arrive.

// onStreamsSelected lets a stream that decodebin3 dropped back into its selection,
// now or once its caps name a decoder.
func (r *Receiver) onStreamsSelected(msg *gst.Message) {
	src := msg.Source()
	if src == nil || src.GetName() != decodeName {
		return
	}
	collection := msg.ParseStreamsSelected()
	if collection == nil {
		return
	}

	var selected []string
	for i := range msg.StreamsSelectedGetSize() {
		selected = append(selected, msg.StreamsSelectedGetStream(i).GetStreamID())
	}
	byID := map[string]gst.Stream{}
	for i := range collection.GetSize() {
		s := collection.GetStream(i)
		byID[s.GetStreamID()] = s
	}
	offers := func() []offeredStream {
		var streams []offeredStream
		for i := range collection.GetSize() {
			streams = append(streams, offered(collection.GetStream(i)))
		}
		return streams
	}
	streams := offers()

	r.reselectMu.Lock()
	waiting := dropped(streams, selected, r.reselected)
	r.reselectMu.Unlock()
	for _, s := range waiting {
		if s.decodable {
			continue
		}
		byID[s.id].Connect("notify::caps", func(changed gst.Stream, _ *gobject.ParamSpec) {
			// The input thread emits it inside decodebin3's locks, which the event takes again.
			go r.reselect(withOffer(streams, offered(changed)), selected)
		})
	}
	// Read again once the handlers sit, so caps landing while they were connected still count.
	r.reselect(offers(), selected)
}

// reselect sends decodebin3 the selection that lets the decodable dropped streams back in,
// at most once per stream ID.
func (r *Receiver) reselect(streams []offeredStream, selected []string) {
	r.reselectMu.Lock()
	ids := reselection(streams, selected, r.reselected)
	for _, id := range ids {
		if !slices.Contains(selected, id) {
			if r.reselected == nil {
				r.reselected = map[string]bool{}
			}
			r.reselected[id] = true
		}
	}
	r.reselectMu.Unlock()
	if ids == nil {
		return
	}

	pipeline := r.heldPipeline()
	if pipeline == nil {
		return
	}
	dec := pipeline.GetByName(decodeName)
	assert.IsNotNil(dec, "the decodebin3 that posted a selection sits in the pipeline")
	logger.Infof("stream %q: %s dropped a stream with a decoder for it, selecting %v", r.name, decodeName, ids)
	if !selectStreams(dec, ids) {
		logger.Warnf("stream %q: %s refused the selection %v", r.name, decodeName, ids)
	}
}

func offered(s gst.Stream) offeredStream {
	return offeredStream{
		id:        s.GetStreamID(),
		kind:      gst.StreamTypeGetName(s.GetStreamType()),
		decodable: decodable(s.GetCaps()),
	}
}

// decodable answers whether a decoder of marginal rank or above takes caps whole,
// the test decodebin3 picks its decoders by.
func decodable(caps *gst.Caps) bool {
	if caps == nil {
		return false
	}
	for _, f := range gst.ElementFactoryListGetElements(gst.ElementFactoryListType(C.decoderFactoryType()), gst.RankMarginal) {
		if f.CanSinkAllCaps(caps) {
			return true
		}
	}
	return false
}

func selectStreams(element gst.Element, ids []string) bool {
	assert.Assert(len(ids) > 0, "a selection names a stream")

	cIDs := make([]*C.char, len(ids))
	for i, id := range ids {
		cIDs[i] = C.CString(id)
		defer C.free(unsafe.Pointer(cIDs[i]))
	}
	ok := C.selectStreams((*C.GstElement)(gst.UnsafeElementToGlibNone(element)), &cIDs[0], C.int(len(ids)))
	runtime.KeepAlive(element)
	return ok != 0
}
