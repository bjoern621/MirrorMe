package control

import "os"

// EnvAutostart leaves the shell's sign-in entry alone for a whole run, whatever the app setting holds.
// The shell reads it to decide whether it writes the entry (avalonia/README.md, "Starting at sign-in"),
// and this process reads it to grey the setting behind it, as EnvTray does for the icon.
const EnvAutostart = "MIRRORME_AUTOSTART"

// autostartForcedOff reports whether the environment holds the entry alone.
// "0" is off and every other value, absence included, leaves the setting deciding.
func autostartForcedOff() bool {
	return os.Getenv(EnvAutostart) == "0"
}
