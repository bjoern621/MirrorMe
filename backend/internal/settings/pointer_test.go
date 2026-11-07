package settings

import (
	"os"
	"path/filepath"
	"testing"

	"bjoernblessin.de/screenshare/internal/pointermode"
)

// A file written while the pointer mode sat under "cursor" names the reader's choice there,
// and the decode starts from the defaults, so the old key has to win over the default it finds.
func TestLoadMigratesTheCursorKeyOntoThePointerMode(t *testing.T) {
	isolateConfig(t)

	dir, err := configDir()
	if err != nil {
		t.Fatalf("resolving the config directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, configFileName),
		[]byte(`{"relay":{"host":"relay.example"},"publish":{"cursor":"hidden"}}`), 0o600); err != nil {
		t.Fatalf("writing a file naming the mode under the old key: %v", err)
	}

	got := mustLoad(t).Publish
	if got.Pointer != pointermode.Hidden {
		t.Errorf("pointer mode = %q, want %q off the old key", got.Pointer, pointermode.Hidden)
	}
	if got.FlatCursor != "" {
		t.Errorf("an upgraded publish carries no old pointer key, got %q", got.FlatCursor)
	}
}

// A preset decodes from nothing rather than from the defaults, so the same key fills an empty field.
func TestLoadPresetsMigratesTheCursorKeyOntoThePointerMode(t *testing.T) {
	isolateConfig(t)

	if err := os.WriteFile(mustPresetsPath(t),
		[]byte(`[{"name":"mine","settings":{"cursor":"metadata"}}]`), 0o644); err != nil {
		t.Fatalf("seeding a preset naming the mode under the old key: %v", err)
	}

	presets, err := LoadPresets()
	if err != nil {
		t.Fatalf("LoadPresets: %v", err)
	}
	if len(presets) != 1 || presets[0].Settings.Pointer != pointermode.Metadata {
		t.Errorf("presets = %+v, want one entry whose pointer mode is %q", presets, pointermode.Metadata)
	}
}
