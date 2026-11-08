// Package gstelements walks a GStreamer iterator and leaves no reference behind.
//
// The binding's Iterator.Next never unsets the GValue that it fills, so each element it yields
// stays referenced for the life of the process.
// One walk over a pipeline then keeps every element alive, and with them the Direct3D 11 device
// that a decoder holds: dozens of driver threads per pipeline, none returned by a stop.
package gstelements

// #cgo pkg-config: gstreamer-1.0
// #include <gst/gst.h>
import "C"

import (
	"unsafe"

	"github.com/go-gst/go-gst/pkg/gst"

	"bjoernblessin.de/go-utils/util/assert"
)

// Of drains an iterator of elements and frees it.
// The caller must not use the iterator afterwards.
func Of(iterator *gst.Iterator) []gst.Element {
	assert.IsNotNil(iterator, "a walk reads the iterator it was handed")
	defer gst.UnsafeIteratorFree(iterator)

	native := (*C.GstIterator)(gst.UnsafeIteratorToGlibNone(iterator))
	var elements []gst.Element
	for {
		var value C.GValue
		switch C.gst_iterator_next(native, &value) {
		case C.GST_ITERATOR_OK:
			object := C.g_value_get_object(&value)
			elements = append(elements, gst.UnsafeElementFromGlibNone(unsafe.Pointer(object)))
			C.g_value_unset(&value)
		case C.GST_ITERATOR_RESYNC:
			// The bin changed under the walk, and what was collected may name removed children.
			C.gst_iterator_resync(native)
			elements = nil
		case C.GST_ITERATOR_DONE:
			return elements
		default:
			assert.Never("an iterator over a bin's children yields no error")
			return elements
		}
	}
}
