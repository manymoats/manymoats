package orch

// splash is the last still of the cut: one ogre sitting with the workers,
// credit already faded in. No HUD, no title, no tower. Enter (or any key)
// begins the board.
func (m model) splash() string {
	return LastStill(m.w, m.h)
}
