package main

import (
	"cmp"
	"slices"
)

const ShiftMinutes = 8 * 60
const WalkBuffer = 2 // minutes

// --- tiny helpers ---

func insertSorted(xs []int, v int) []int {
	i := slices.IndexFunc(xs, func(x int) bool { return x >= v })
	if i == -1 {
		return append(xs, v)
	}
	xs = append(xs, 0)
	copy(xs[i+1:], xs[i:])
	xs[i] = v
	return xs
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Try to place events with a phase shift in [0..WalkBuffer] minutes.
// Returns (ok, chosenStart).
func tryAssign(events []int, startOffset, cycle, pieces int) (bool, int) {
	for phase := 0; phase <= WalkBuffer; phase++ {
		ok := true
		for n := 0; n < pieces; n++ {
			t := startOffset + phase + n*cycle
			if t >= ShiftMinutes {
				ok = false
				break
			}
			// conflict to any existing event?
			for _, e := range events {
				if abs(t-e) < WalkBuffer {
					ok = false
					break
				}
			}
			if !ok {
				break
			}
		}
		if ok {
			return true, startOffset + phase
		}
	}
	return false, 0
}

// Commit events (at chosenStart + n*cycle) into a sorted list.
func commit(events []int, chosenStart, cycle, pieces int) []int {
	for n := 0; n < pieces; n++ {
		t := chosenStart + n*cycle
		if t >= ShiftMinutes {
			break
		}
		events = insertSorted(events, t)
	}
	return events
}

// --- the main scheduler ---

func scheduleWeek(machs []Machine, emps []Employee, jobs []Job) ([]Assignment, map[int][]int) {
	// Sort machines by name; employees: Day first, then name
	slices.SortFunc(machs, func(a, b Machine) int { return cmp.Compare(a.Name, b.Name) })
	slices.SortFunc(emps, func(a, b Employee) int {
		if a.Shift != b.Shift {
			if a.Shift == Day {
				return -1
			}
			return 1
		}
		return cmp.Compare(a.Name, b.Name)
	})

	// Machine capability pools
	mills, lathes := []Machine{}, []Machine{}
	for _, m := range machs {
		if m.Processes[ProcMill] {
			mills = append(mills, m)
		}
		if m.Processes[ProcTurn] {
			lathes = append(lathes, m)
		}
	}

	// Greedy pick: least total minutes already packed across week/shifts
	type key struct {
		mid, day int
		sh       Shift
	}
	mCursor := map[key]int{} // minutes used per machine/day/shift
	sumLoad := func(mid int) int {
		total := 0
		for d := 0; d < 7; d++ {
			total += mCursor[key{mid, d, Day}]
			total += mCursor[key{mid, d, Night}]
		}
		return total
	}
	pickMachine := func(proc Process) int {
		pool := mills
		if proc == ProcTurn {
			pool = lathes
		}
		if len(pool) == 0 {
			return 0
		}
		best := pool[0].ID
		bestLoad := sumLoad(best)
		for _, m := range pool[1:] {
			if l := sumLoad(m.ID); l < bestLoad {
				best, bestLoad = m.ID, l
			}
		}
		return best
	}

	// Per-employee load “event” times (mins from shift start)
	type esKey struct {
		id, day int
		sh      Shift
	}
	eventTimes := map[esKey][]int{}

	// Employees grouped by shift and filtered for skill at call-site
	empsByShift := map[Shift][]Employee{Day: {}, Night: {}}
	for _, e := range emps {
		empsByShift[e.Shift] = append(empsByShift[e.Shift], e)
	}

	// Utilization (minutes) per machine per day (day shift only, as before)
	mUtil := map[int][]int{}
	for _, m := range machs {
		mUtil[m.ID] = make([]int, 7)
	}

	assignments := []Assignment{}

	// For each job, place pieces on machines; assign employee immediately.
	for _, job := range jobs {
		minsLeft := job.Qty * job.CycleMins
		if minsLeft <= 0 {
			continue
		}

		mid := pickMachine(job.Process)
		if mid == 0 {
			continue
		} // no capable machine

		day, sh := 0, Day

		for minsLeft > 0 && day < 7 {
			k := key{mid, day, sh}
			used := mCursor[k]
			if used >= ShiftMinutes {
				// advance shift/day
				if sh == Day {
					sh = Night
				} else {
					sh = Day
					day++
				}
				continue
			}

			space := ShiftMinutes - used
			if space < job.CycleMins {
				// not enough time for even one piece -> next shift/day
				if sh == Day {
					sh = Night
				} else {
					sh = Day
					day++
				}
				continue
			}

			// How many pieces fit here (cap by what's left)
			maxPiecesHere := space / job.CycleMins
			wantPieces := minsLeft / job.CycleMins
			if wantPieces == 0 {
				wantPieces = 1
			}
			if maxPiecesHere < wantPieces {
				wantPieces = maxPiecesHere
			}

			// Find an operator; if none, bump the block to next day/shift and retry.
			found := false
			finalDay, finalShift := day, sh
			finalEmp := 0

			// Try current slot first, then roll ahead up to end of week
			tryDay, tryShift := day, sh
			for tries := 0; tries < 14 && tryDay < 7; tries++ {
				kk := key{mid, tryDay, tryShift}
				cur := mCursor[kk]
				if cur >= ShiftMinutes {
					if tryShift == Day {
						tryShift = Night
					} else {
						tryShift = Day
						tryDay++
					}
					continue
				}
				avail := ShiftMinutes - cur
				if avail < job.CycleMins {
					if tryShift == Day {
						tryShift = Night
					} else {
						tryShift = Day
						tryDay++
					}
					continue
				}

				herePieces := avail / job.CycleMins
				pieces := wantPieces
				if pieces > herePieces {
					pieces = herePieces
				}
				if pieces == 0 {
					if tryShift == Day {
						tryShift = Night
					} else {
						tryShift = Day
						tryDay++
					}
					continue
				}

				// Try employees for this shift with the right skill
				candidate := 0
				chosenStart := 0
				for _, e := range empsByShift[tryShift] {
					if !e.Skills[job.Process] {
						continue
					}
					esK := esKey{e.ID, tryDay, tryShift}
					ok, st := tryAssign(eventTimes[esK], cur, job.CycleMins, pieces)
					if ok {
						candidate = e.ID
						chosenStart = st
						break
					}
				}

				if candidate != 0 {
					// success: place block here
					finalDay, finalShift = tryDay, tryShift
					finalEmp = candidate

					mCursor[kk] += pieces * job.CycleMins
					if finalShift == Day {
						mUtil[mid][finalDay] += pieces * job.CycleMins
					}
					esK := esKey{finalEmp, finalDay, finalShift}
					eventTimes[esK] = commit(eventTimes[esK], chosenStart, job.CycleMins, pieces)

					assignments = append(assignments, Assignment{
						JobID:      job.ID,
						MachineID:  mid,
						DayIndex:   finalDay,
						Shift:      finalShift,
						Minutes:    pieces * job.CycleMins,
						Pieces:     pieces,
						CycleMins:  job.CycleMins,
						Process:    job.Process,
						EmployeeID: finalEmp,
					})

					minsLeft -= pieces * job.CycleMins
					found = true
					break
				}

				if tryShift == Day {
					tryShift = Night
				} else {
					tryShift = Day
					tryDay++
				}
			}

			if !found {
				// Could not find any operator through the week.
				// Place the block at the original day/shift (or the last capacity-checked one) as UNASSIGNED,
				// respecting machine capacity there.
				kk := key{mid, day, sh}
				cur := mCursor[kk]
				avail := ShiftMinutes - cur
				if avail < job.CycleMins {
					// advance day/shift until some capacity exists; if none anywhere, break
					dd, ss := day, sh
					ok := false
					for dd < 7 && !ok {
						if ss == Day {
							ss = Night
						} else {
							ss = Day
							dd++
						}
						if dd >= 7 {
							break
						}
						if mCursor[key{mid, dd, ss}] <= ShiftMinutes-job.CycleMins {
							ok = true
							day, sh = dd, ss
						}
					}
					if !ok {
						break
					} // nowhere to place
					kk = key{mid, day, sh}
					cur = mCursor[kk]
					avail = ShiftMinutes - cur
				}
				pieces := avail / job.CycleMins
				if pieces > minsLeft/job.CycleMins {
					pieces = minsLeft / job.CycleMins
				}
				if pieces == 0 { // safety
					if sh == Day {
						sh = Night
					} else {
						sh = Day
						day++
					}
					continue
				}

				mCursor[kk] += pieces * job.CycleMins
				if sh == Day {
					mUtil[mid][day] += pieces * job.CycleMins
				}

				assignments = append(assignments, Assignment{
					JobID:     job.ID,
					MachineID: mid,
					DayIndex:  day,
					Shift:     sh,
					Minutes:   pieces * job.CycleMins,
					Pieces:    pieces,
					CycleMins: job.CycleMins,
					Process:   job.Process,
					// EmployeeID: 0 (unassigned)
				})
				minsLeft -= pieces * job.CycleMins
			}

			// Continue filling from the *current* day/shift (they may still have capacity)
			// Loop will try here again first; if full it advances.
		}
	}

	return assignments, mUtil
}
