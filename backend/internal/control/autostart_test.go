package control

import "testing"

// The shell's gate reads the same variable (AutostartEntry.IsWanted), so the two agree per value.
func TestOnlyZeroLeavesTheSignInEntryAlone(t *testing.T) {
	for _, tc := range []struct {
		set  string
		want bool
	}{
		{"0", true},
		{"", false},
		{"1", false},
		{"false", false},
	} {
		t.Setenv(EnvAutostart, tc.set)

		if got := autostartForcedOff(); got != tc.want {
			t.Errorf("autostartForcedOff() = %v with %s=%q, want %v", got, EnvAutostart, tc.set, tc.want)
		}
	}
}
