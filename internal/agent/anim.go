package agent

import "math"

// Silk — the house seed. Exponential, no overshoot. Nothing bounces.
func Silk(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return 1 - math.Pow(2, -10*t)
}

// BeamAt returns the leading-edge column of the sweep at progress t across width w.
func BeamAt(t float64, w int) int { return int(SilkInOut(t) * float64(w)) }

// BeamWedge returns the intensity 0..1 of the beam at column x — a 6-cell wedge
// trailing the leading edge, so the light has a body and not just a point.
func BeamWedge(x, lead int) float64 {
	const tail = 6
	d := lead - x
	if d < 0 || d > tail {
		return 0
	}
	return 1 - float64(d)/float64(tail)
}

// Woken reports whether the beam has already passed a mark sitting at column x,
// and how far into its 180ms fade it is.
func Woken(x, lead int, fadeCells int) float64 {
	if fadeCells <= 0 {
		fadeCells = 3
	}
	if lead < x {
		return 0
	}
	p := float64(lead-x) / float64(fadeCells)
	if p > 1 {
		return 1
	}
	return Silk(p)
}

// Ramp picks a shade from the silver ramp by intensity.
func Ramp(intensity float64) string {
	if intensity <= 0 {
		return "#232a33"
	}
	i := int(intensity * float64(len(Silver)-1))
	if i >= len(Silver) {
		i = len(Silver) - 1
	}
	return Silver[i]
}

// SilkInOut — symmetric, cubic. A sweep is a head turning: slow at the edges,
// quick through the middle, but MOVING the whole time. Plain Silk is an ease-OUT
// and lurches to 95% by t=0.45. An exponential in-out is the opposite failure —
// it sits still for 30%, blurs across, and sits still again, which reads as a
// strobe rather than a turn. Cubic spreads the movement across the whole beat.
func SilkInOut(t float64) float64 {
	switch {
	case t <= 0:
		return 0
	case t >= 1:
		return 1
	case t < 0.5:
		return 4 * t * t * t
	default:
		return 1 - math.Pow(-2*t+2, 3)/2
	}
}
