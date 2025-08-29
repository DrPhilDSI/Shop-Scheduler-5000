package main

type Process string

const (
	ProcMill Process = "mill"
	ProcTurn Process = "turn"
)

type Shift string

const (
	Day   Shift = "day"
	Night Shift = "night"
)

type Machine struct {
	ID        int
	Name      string
	Processes map[Process]bool // capabilities
}

type Employee struct {
	ID          int
	Name        string
	Shift       Shift
	HoursPerDay int // usually 8
	Skills      map[Process]bool
}

type Job struct {
	ID        int
	Name      string
	Process   Process // primary process required (mill/turn). For 'both', make 2 jobs for now.
	Qty       int     // number of pieces
	CycleMins int     // minutes per piece
}

type Assignment struct {
	JobID      int
	MachineID  int
	DayIndex   int   // 0..6
	Shift      Shift // day/night
	Minutes    int   // minutes in this block (<= 8*60)
	Pieces     int   // how many pieces in this block
	CycleMins  int
	Process    Process
	EmployeeID int // 0 if unassigned
}

type ViewModel struct {
	WeekStart   TimeYMD
	Days        []TimeYMD
	Machines    []Machine
	Employees   []Employee
	Jobs        []Job
	Backlog     []Job
	Assignments []Assignment

	JobByID map[int]Job
	MacByID map[int]Machine
	EmpByID map[int]Employee

	// optional: per-machine/day shift utilization mins
	Util        map[int][]int // machineID -> [7]int minutes used on day shift only; expand if needed
	JobMinsDone map[int]int
	EmpColors   map[int]string
}
