package main

func demoMachines() []Machine {
	return []Machine{
		{ID: 1, Name: "Haas VF-2SS", Processes: map[Process]bool{ProcMill: true}},
		{ID: 2, Name: "Okuma LB3000", Processes: map[Process]bool{ProcMill: true}},
		{ID: 3, Name: "Hermle C22U", Processes: map[Process]bool{ProcTurn: true}},
		{ID: 4, Name: "DMG CLX", Processes: map[Process]bool{ProcTurn: true}},
	}
}
func demoEmployees() []Employee {
	return []Employee{
		{ID: 1, Name: "Alex", Shift: Day, HoursPerDay: 8, Skills: map[Process]bool{ProcMill: true}},
		{ID: 2, Name: "Blair", Shift: Day, HoursPerDay: 8, Skills: map[Process]bool{ProcTurn: true}},
		{ID: 3, Name: "Casey", Shift: Night, HoursPerDay: 8, Skills: map[Process]bool{ProcMill: true, ProcTurn: true}},
		{ID: 4, Name: "Evan", Shift: Day, HoursPerDay: 8, Skills: map[Process]bool{ProcTurn: true}},
	}
}

// feel free to tweak hours to see more bars fill the lanes
func demoJobs() []Job {
	return []Job{
		{ID: 1, Name: "Rocket Bracket", Process: ProcMill, Qty: 60, CycleMins: 45},
		{ID: 2, Name: "Watch Crown", Process: ProcTurn, Qty: 1200, CycleMins: 3},
		{ID: 3, Name: "Valve Body", Process: ProcMill, Qty: 40, CycleMins: 8},
		{ID: 4, Name: "Rotor Cap", Process: ProcTurn, Qty: 30, CycleMins: 60},
		{ID: 5, Name: "Manifold", Process: ProcMill, Qty: 24, CycleMins: 120},
		{ID: 6, Name: "Spindle Nut", Process: ProcTurn, Qty: 80, CycleMins: 5},
	}
}
