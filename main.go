package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// --- In-memory store and view model ---

var store = struct {
	mu          sync.RWMutex
	machines    []Machine
	employees   []Employee
	jobs        []Job
	assignments []Assignment
	weekStart   time.Time
}{}

// Derived utilization: employee -> [7]int (hours booked that day)
type Utilization map[string][]int

// in main.go (viewModel)
type viewModel struct {
	WeekStart   time.Time
	Days        []time.Time
	Machines    []Machine
	Employees   []Employee
	Jobs        []Job
	Assignments []Assignment

	EmpByID map[string]Employee
	MacByID map[string]Machine
	JobByID map[string]Job

	Util        Utilization
	Backlog     []Job
	JobHours    map[string]int
	EmpWeekLeft map[string]int // <- NEW: remaining hours across week
}

//go:embed templates/*
var templatesFS embed.FS

var tmpl = template.Must(
	template.New("base").
		Option("missingkey=zero").
		Funcs(template.FuncMap{
			"title": func(v any) string {
				s := fmt.Sprint(v)
				if s == "" {
					return s
				}
				return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
			},
			"pctHours": func(h int) string {
				if h < 0 {
					h = 0
				}
				if h > 8 {
					h = 8
				}
				return fmt.Sprintf("%.5f%%", (float64(h)/8.0)*100.0)
			},
			"sub": func(a, b int) int { return a - b }, // <- add
		}).
		ParseFS(templatesFS, "templates/*.gohtml"),
)

func main() {
	seedData()

	// compute Monday week start
	now := time.Now()
	wd := int(now.Weekday())
	if wd == 0 {
		wd = 7
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(wd - 1))

	store.weekStart = start
	store.assignments = nil // <-- start empty for demo

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/override", handleOverride)
	http.HandleFunc("/api/reset", handleReset)

	log.Println("Machine Shop Scheduler running on http://localhost:8080 …")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	store.mu.RLock()
	vm := buildViewModel()
	store.mu.RUnlock()

	log.Printf("INDEX: employees=%d machines=%d jobs=%d assignments=%d",
		len(vm.Employees), len(vm.Machines), len(vm.Jobs), len(vm.Assignments))

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "index.gohtml", vm); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "template render error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

func buildViewModel() viewModel {
	days := make([]time.Time, 7)
	for i := 0; i < 7; i++ {
		days[i] = store.weekStart.AddDate(0, 0, i)
	}
	empBy := make(map[string]Employee, len(store.employees))
	for _, e := range store.employees {
		empBy[e.ID] = e
	}
	macBy := make(map[string]Machine, len(store.machines))
	for _, m := range store.machines {
		macBy[m.ID] = m
	}
	jobBy := make(map[string]Job, len(store.jobs))
	for _, j := range store.jobs {
		jobBy[j.ID] = j
	}

	// Utilization
	util := make(Utilization, len(store.employees))
	for _, e := range store.employees {
		util[e.ID] = make([]int, 7)
	}

	// Hours scheduled per job
	jobHours := make(map[string]int, len(store.jobs))
	for _, a := range store.assignments {
		if _, ok := util[a.EmployeeID]; ok && a.DayIndex >= 0 && a.DayIndex < 7 {
			util[a.EmployeeID][a.DayIndex] += a.Hours
			if util[a.EmployeeID][a.DayIndex] > 8 {
				util[a.EmployeeID][a.DayIndex] = 8
			}
		}
		jobHours[a.JobID] += a.Hours
	}

	// Backlog = jobs with remaining hours > 0
	backlog := make([]Job, 0, len(store.jobs))
	for _, j := range store.jobs {
		if jobHours[j.ID] < j.RunHours {
			backlog = append(backlog, j)
		}
	}

	emps := make([]Employee, len(store.employees))
	copy(emps, store.employees)
	sort.Slice(emps, func(i, j int) bool {
		// Day before Night
		if emps[i].Shift != emps[j].Shift {
			return emps[i].Shift == Day
		}
		// then by Name
		return emps[i].Name < emps[j].Name
	})

	// Weekly hours left per employee (7 * HoursPerDay - used)
	empWeekLeft := make(map[string]int, len(store.employees))
	for _, e := range store.employees {
		used := 0
		for _, h := range util[e.ID] {
			used += h
		}
		empWeekLeft[e.ID] = 7*e.HoursPerDay - used
	}

	return viewModel{
		WeekStart:   store.weekStart,
		Days:        days,
		Machines:    store.machines,
		Employees:   emps, // sorted list if you added sorting earlier
		Jobs:        store.jobs,
		Assignments: store.assignments,
		EmpByID:     empBy,
		MacByID:     macBy,
		JobByID:     jobBy,
		Util:        util,
		Backlog:     backlog,
		JobHours:    jobHours,
		EmpWeekLeft: empWeekLeft, // <- pass to template
	}
}

func handleOverride(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var dto struct {
		DayIndex   int     `json:"dayIndex"`
		Shift      Shift   `json:"shift"`
		MachineID  string  `json:"machineId"`
		EmployeeID string  `json:"employeeId"`
		JobID      string  `json:"jobId"`
		Hours      int     `json:"hours"`
		Process    Process `json:"process"`
	}
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	store.mu.Lock()
	store.assignments = append(store.assignments, Assignment{
		DayIndex:   dto.DayIndex,
		Shift:      dto.Shift,
		MachineID:  dto.MachineID,
		EmployeeID: dto.EmployeeID,
		JobID:      dto.JobID,
		Hours:      dto.Hours,
		Process:    dto.Process,
	})
	store.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	store.mu.Lock()
	store.assignments = scheduleWeek(store.weekStart, store.machines, store.employees, store.jobs)
	log.Printf("Auto-scheduled %d assignments", len(store.assignments))
	store.mu.Unlock()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
