package watchsvc

import (
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"bjoernblessin.de/go-utils/util/assert"
)

// Lease is how long a statement stands without a restatement.
// A viewer restates on every receive sample, a second apart,
// so a lease of several seconds covers a few lost ones.
// A viewer that stopped drops off the publisher's table inside it.
const Lease = 10 * time.Second

// MaxViewings bounds one statement, at many more streams than one machine decodes at once.
const MaxViewings = 32

// Viewing is one stream a member watches and what that member's decode measures of it.
// Every figure is optional: absent means unmeasured, never zero.
type Viewing struct {
	Path      string `json:"path"`
	Transport string `json:"transport"`
	// ms, from the publisher's encoder to this viewer's decoder, off the clock stamped into each picture.
	PathMs *float64 `json:"pathMs,omitempty"`
	// ms, from the publisher's screen to this viewer's window.
	// A floor where PathMs is absent (docs/delay-measurement.md).
	TotalMs *float64 `json:"totalMs,omitempty"`
	// Frames a second reaching this viewer's window.
	ShownFPS *float64 `json:"shownFps,omitempty"`
	// Frames a second this viewer's decoder shed to keep up.
	DroppedFPS *float64 `json:"droppedFps,omitempty"`
}

// Viewer is a Viewing as the publisher of its path reads it, with the member it came from.
type Viewer struct {
	Member string `json:"member"`
	Viewing
}

// malformed names the first rule a viewing breaks, empty where it breaks none.
func (v Viewing) malformed() string {
	if v.Path == "" {
		return "a viewing names the path it watches"
	}
	if v.Transport == "" {
		return "a viewing names the leg it watches over"
	}
	for _, figure := range []*float64{v.PathMs, v.TotalMs, v.ShownFPS, v.DroppedFPS} {
		if figure != nil && (*figure < 0 || math.IsInf(*figure, 0) || math.IsNaN(*figure)) {
			return "a figure is a measurement of zero or more"
		}
	}
	return ""
}

// statement is everything one member stated it watches, and when that stops standing.
type statement struct {
	viewing []Viewing
	until   time.Time
}

// registry holds each member's latest statement, keyed by the token's subject.
// Can be called concurrently.
type registry struct {
	mu         sync.Mutex
	statements map[string]statement
}

func newRegistry() *registry {
	return &registry{statements: map[string]statement{}}
}

// State replaces what member watches with viewing, standing until now plus Lease.
// An empty viewing removes the member.
func (r *registry) State(member string, viewing []Viewing, now time.Time) {
	assert.Assert(member != "", "a statement belongs to a member")

	r.mu.Lock()
	defer r.mu.Unlock()

	r.pruneLocked(now)
	if len(viewing) == 0 {
		delete(r.statements, member)
		return
	}
	r.statements[member] = statement{viewing: slices.Clone(viewing), until: now.Add(Lease)}
}

// ViewersOf answers every standing viewing of path, in member order.
// The figures are copies, so a caller cannot reach into the registry.
func (r *registry) ViewersOf(path string, now time.Time) []Viewer {
	assert.Assert(path != "", "viewers are read for a path")

	r.mu.Lock()
	defer r.mu.Unlock()

	r.pruneLocked(now)
	viewers := []Viewer{}
	for member, s := range r.statements {
		for _, v := range s.viewing {
			if v.Path == path {
				viewers = append(viewers, Viewer{Member: member, Viewing: v.copied()})
			}
		}
	}
	slices.SortFunc(viewers, func(a, b Viewer) int {
		if order := strings.Compare(a.Member, b.Member); order != 0 {
			return order
		}
		return strings.Compare(a.Transport, b.Transport)
	})
	return viewers
}

// pruneLocked deletes every statement past its lease.
// Assumes the caller holds mu.
func (r *registry) pruneLocked(now time.Time) {
	for member, s := range r.statements {
		if !now.Before(s.until) {
			delete(r.statements, member)
		}
	}
}

func (v Viewing) copied() Viewing {
	v.PathMs = copyOf(v.PathMs)
	v.TotalMs = copyOf(v.TotalMs)
	v.ShownFPS = copyOf(v.ShownFPS)
	v.DroppedFPS = copyOf(v.DroppedFPS)
	return v
}

func copyOf(figure *float64) *float64 {
	if figure == nil {
		return nil
	}
	value := *figure
	return &value
}
