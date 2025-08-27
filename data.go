package main

type Process string

const (
	Mill Process = "mill"
	Turn Process = "turn"
	Both Process = "both"
)

type Shift string

const (
	Day   Shift = "day"
	Night Shift = "night"
)

type Machine struct {
	ID        string
	Name      string
	Processes []Process // what this machine can run
}

type Employee struct {
	ID          string
	Name        string
	Skills      []Process // mill/turn/both
	Shift       Shift
	HoursPerDay int // assume 8
}

type Job struct {
	ID        string
	Name      string
	Processes []Process // sequential processes
	RunHours  int       // total runtime hours
}

type Assignment struct {
	DayIndex   int
	Shift      Shift
	MachineID  string
	EmployeeID string
	JobID      string
	Hours      int
	Process    Process
}

func seedData() {
	store.machines = []Machine{
		{ID: "M1", Name: "Haas VF-2SS", Processes: []Process{Mill}},
		{ID: "M2", Name: "Okuma LB3000", Processes: []Process{Turn}},
		{ID: "M3", Name: "Hermle C22U", Processes: []Process{Mill}},
		{ID: "M4", Name: "DMG CLX", Processes: []Process{Mill, Turn}},
	}
	store.employees = []Employee{
		{ID: "E1", Name: "Incoherent Rambler", Skills: []Process{Mill}, Shift: Day, HoursPerDay: 8},
		{ID: "E2", Name: "John & John", Skills: []Process{Turn}, Shift: Night, HoursPerDay: 8},
		{ID: "E3", Name: "Rob", Skills: []Process{Both}, Shift: Day, HoursPerDay: 8},
		{ID: "E4", Name: "Brown guy", Skills: []Process{Both}, Shift: Night, HoursPerDay: 3},
		{ID: "E5", Name: "1186", Skills: []Process{Both}, Shift: Night, HoursPerDay: 8},
		{ID: "E5", Name: "Probably Ai", Skills: []Process{Both}, Shift: Night, HoursPerDay: 5},
	}
	// feel free to tweak hours to see more bars fill the lanes
	store.jobs = []Job{
		{ID: "J1001", Name: "Rocket Bracket", Processes: []Process{Mill}, RunHours: 24},
		{ID: "J1002", Name: "Valve Body", Processes: []Process{Turn, Mill}, RunHours: 20},
		{ID: "J1003", Name: "Watch Crown", Processes: []Process{Turn}, RunHours: 16},
		{ID: "J1004", Name: "Camera Plate", Processes: []Process{Mill}, RunHours: 12},
		{ID: "J1005", Name: "Knife", Processes: []Process{Mill}, RunHours: 12},
		{ID: "J1006", Name: "Some Artsy Thing", Processes: []Process{Turn, Mill}, RunHours: 12},
	}
}
