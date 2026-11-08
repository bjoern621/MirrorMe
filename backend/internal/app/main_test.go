package app

import (
	"fmt"
	"os"
	"testing"
)

// TestMain points os.UserConfigDir at a temp directory for the whole test binary.
// A test reaching settings.Save without isolateConfig would overwrite the developer's real settings.
// DISPLAY stands in for the X11 capture rows, which build their arguments without connecting.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "mirrorme-app-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "isolating the config directory:", err)
		os.Exit(1)
	}
	for _, name := range []string{"XDG_CONFIG_HOME", "AppData", "HOME"} {
		if err := os.Setenv(name, dir); err != nil {
			fmt.Fprintln(os.Stderr, "isolating the config directory:", err)
			os.Exit(1)
		}
	}

	if os.Getenv("DISPLAY") == "" {
		os.Setenv("DISPLAY", ":0")
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
