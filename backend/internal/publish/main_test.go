package publish

import (
	"os"
	"testing"
)

// TestMain gives the X11 capture rows a DISPLAY to name.
// They build their arguments without connecting, and Windows or a headless session sets none.
func TestMain(m *testing.M) {
	if os.Getenv("DISPLAY") == "" {
		os.Setenv("DISPLAY", ":0")
	}
	os.Exit(m.Run())
}
