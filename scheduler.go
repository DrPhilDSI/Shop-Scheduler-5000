package main

import (
	"fmt"
	"sort"
	"time"
)

// scheduleWeek: simple greedy baseline with 8h day/shift blocks.
func scheduleWeek(weekStart time.Time, machines []Machine, emps []Employee, jobs []Job) []Assignment {
	const days = 7
	shiftOrder := []Shift{Day, Night}

	// maps to track capacity used per (emp,day)
	empDayUsed := make(map[string][days]int)
	// machine usage map: (machine,day,shift) -> bool (one job per machine/shift block)
	machineUsed := make(map[string]bool)

	assignments := make([]Assignment, 0)
	mKey := func(mid string, day int, shift Shift) string {
		return fmt.Sprintf("%s:%d:%s", mid, day, shift)
	}

	// Split job hours across its processes evenly (remainder front-loaded)
	splitHours := func(total int, n int) []int {
		res := make([]int, n)
		for i := 0; i < n; i++ {
			res[i] = total / n
		}
		for i := 0; i < total%n; i++ {
			res[i]++
		}
		return res
	}

	for _, job := range jobs {
		perStep := splitHours(job.RunHours, len(job.Processes))
		for pi, proc := range job.Processes {
			remaining := perStep[pi]
			for day := 0; day < days && remaining > 0; day++ {
				for _, shift := range shiftOrder {
					if remaining <= 0 {
						break
					}
					// employees on this shift with skill
					candEmps := filterEmployees(emps, func(e Employee) bool {
						return hasSkill(e, proc) && e.Shift == shift
					})
					if len(candEmps) == 0 {
						continue
					}
					// machines supporting this process and free this shift
					candMacs := filterMachines(machines, func(m Machine) bool {
						return supports(m, proc) && !machineUsed[mKey(m.ID, day, shift)]
					})
					if len(candMacs) == 0 {
						continue
					}
					// prefer single-process machines first to keep hybrids free
					sort.Slice(candMacs, func(i, j int) bool {
						return len(candMacs[i].Processes) < len(candMacs[j].Processes)
					})

					var chosenEmp *Employee
					for idx := range candEmps {
						u := empDayUsed[candEmps[idx].ID]
						if u[day] < candEmps[idx].HoursPerDay {
							chosenEmp = &candEmps[idx]
							break
						}
					}
					if chosenEmp == nil {
						continue
					}
					chosenMac := candMacs[0]

					// schedule a block up to remaining or employee remaining or 8h shift
					used := empDayUsed[chosenEmp.ID]
					capLeft := chosenEmp.HoursPerDay - used[day]
					if capLeft <= 0 {
						continue
					}
					block := min(remaining, capLeft, 8)
					assignments = append(assignments, Assignment{
						DayIndex:   day,
						Shift:      shift,
						MachineID:  chosenMac.ID,
						EmployeeID: chosenEmp.ID,
						JobID:      job.ID,
						Hours:      block,
						Process:    proc,
					})
					used[day] += block
					empDayUsed[chosenEmp.ID] = used
					machineUsed[mKey(chosenMac.ID, day, shift)] = true
					remaining -= block
				}
			}
		}
	}
	return assignments
}

func hasSkill(e Employee, p Process) bool {
	for _, s := range e.Skills {
		if s == p || s == Both {
			return true
		}
	}
	return false
}

func supports(m Machine, p Process) bool {
	for _, s := range m.Processes {
		if s == p || s == Both {
			return true
		}
	}
	return false
}

func min(nums ...int) int {
	m := nums[0]
	for _, n := range nums[1:] {
		if n < m {
			m = n
		}
	}
	return m
}

func filterEmployees(list []Employee, pred func(Employee) bool) []Employee {
	out := make([]Employee, 0, len(list))
	for _, e := range list {
		if pred(e) {
			out = append(out, e)
		}
	}
	return out
}

func filterMachines(list []Machine, pred func(Machine) bool) []Machine {
	out := make([]Machine, 0, len(list))
	for _, m := range list {
		if pred(m) {
			out = append(out, m)
		}
	}
	return out
}
