package app

import (
	"golang.org/x/sys/windows"
)

// openWithDefaultHandler hands path to ShellExecute whole.
//
// `cmd /c start` would cut a URL at its first `&` and run the rest as a command of its own,
// dropping every query parameter after it, a link start's nonce among them.
func openWithDefaultHandler(path string) error {
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, nil, file, nil, nil, windows.SW_SHOWNORMAL)
}
