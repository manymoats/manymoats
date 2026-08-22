package agent

import (
	"sync"
	"time"
)

// A noisy instantaneous signal (Cursor's CPU share) crossing a hard threshold
// makes rows appear and disappear between refreshes. A board that blinks is a
// board you stop believing. Steady remembers when something was last genuinely
// busy and holds it there through the dips.
const (
	holdWorking = 25 * time.Second
	enterBusy   = 12.0 // %cpu across an app's processes — becoming busy
	leaveBusy   = 4.0  // %cpu — sustained quiet before we call it idle
	firstSight  = 8.0  // %cpu — an agent we have never observed before
)

type steady struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

var busy = &steady{seen: map[string]time.Time{}}

// Settle applies hysteresis to CPU-derived states. Token-derived states are
// already smooth — they come from a file that only changes when work happens.
func Settle(as []Agent, now time.Time) []Agent {
	busy.mu.Lock()
	defer busy.mu.Unlock()
	for i := range as {
		a := &as[i]
		if a.CPUPct <= 0 {
			continue
		}
		// never resurrect something its own source says is waiting on the founder
		if a.State == Asks {
			continue
		}
		key := string(a.Source) + "/" + a.Project
		// First sight gets the benefit of the doubt for one cycle: launching the
		// board while an agent happens to be mid-dip must not hide it. After that
		// the normal thresholds govern.
		if _, known := busy.seen[key]; !known && a.CPUPct >= firstSight {
			busy.seen[key] = now
			a.State = Working
			continue
		}
		switch {
		case a.CPUPct >= enterBusy:
			busy.seen[key] = now
			a.State = Working
		case a.CPUPct > leaveBusy:
			// in the grey band: keep whatever it already was
			if last, ok := busy.seen[key]; ok && now.Sub(last) < holdWorking {
				a.State = Working
			}
		default:
			if last, ok := busy.seen[key]; ok && now.Sub(last) < holdWorking {
				a.State = Working
			} else {
				a.State = Idle
			}
		}
	}
	return as
}

func resetSteady() {
	busy.mu.Lock()
	busy.seen = map[string]time.Time{}
	busy.mu.Unlock()
}
