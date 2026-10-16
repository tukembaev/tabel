package domain

import "time"

// LunchOverlapSeconds is how much of [opened, closed] falls inside [lunchStart, lunchEnd].
func LunchOverlapSeconds(opened, closed, lunchStart, lunchEnd time.Time) int {
	if !closed.After(opened) || !lunchEnd.After(lunchStart) {
		return 0
	}
	start := opened
	if lunchStart.After(start) {
		start = lunchStart
	}
	end := closed
	if lunchEnd.Before(end) {
		end = lunchEnd
	}
	if !end.After(start) {
		return 0
	}
	return int(end.Sub(start) / time.Second)
}
