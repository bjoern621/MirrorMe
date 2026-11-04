package timeline

// Tunables for judging a run.
const (
	// Share of the target frame rate under which a reading counts as slow.
	DIP_SHARE = 0.8
	// Share a reading has to reach again to end a dip,
	// above DIP_SHARE so a rate hovering at the line does not open and close dips on every reading.
	RECOVER_SHARE = 0.9
	// Slow readings in a row that open a dip. The encoder reports about once a second.
	DIP_SAMPLES = 2
)
