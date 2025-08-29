package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var currentAssignments []Assignment
var currentUtil map[int][]int

type TimeYMD struct{ time.Time }

func (t TimeYMD) Format(f string) string { return t.Time.Format(f) }

func startOfWeek(t time.Time) time.Time {
	// Monday as start
	w := int(t.Weekday())
	if w == 0 {
		w = 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-w+1, 0, 0, 0, 0, t.Location())
}

func pctMins(mins int) string {
	if mins <= 0 {
		return "0%"
	}
	if mins >= ShiftMinutes {
		return "100%"
	}
	p := float64(mins) / float64(ShiftMinutes) * 100.0
	return template.HTMLEscapeString(sprintFloat(p)) + "%"
}
func sprintFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 0, 64)
}
func title(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
func sub(a, b int) int { return a - b }

func mul(a, b int) int { return a * b }
func max0(x int) int {
	if x < 0 {
		return 0
	}
	return x
}

var tpl *template.Template

func hasProc(m Machine, p string) bool   { return m.Processes[Process(p)] }
func hasSkill(e Employee, p string) bool { return e.Skills[Process(p)] }
func minsToHM(mins int) string {
	if mins <= 0 {
		return "0m"
	}
	h := mins / 60
	m := mins % 60
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

// imports you may need on top:
// import "strings"

func procTitle(p Process) string {
	switch p {
	case ProcMill:
		return "Mill"
	case ProcTurn:
		return "Turn"
	default:
		// fallback if you add more later
		return strings.Title(string(p))
	}
}

var empPalette = []string{
	"bg-rose-900/40",
	"bg-amber-900/40",
	"bg-emerald-900/40",
	"bg-sky-900/40",
	"bg-fuchsia-900/40",
	"bg-teal-900/40",
	"bg-indigo-900/40",
	"bg-lime-900/40",
}

func empClass(eid int) string {
	if eid == 0 {
		// Unassigned
		return "bg-gray-700/50 ring-1 ring-red-500/40"
	}
	// Stable-ish hash to palette
	idx := eid % len(empPalette)
	if idx < 0 {
		idx = -idx
	}
	return empPalette[idx] + " ring-1 ring-white/10"
}

func main() {
	// Template funcs
	funcs := template.FuncMap{
		"title":     title,
		"sub":       sub,
		"mul":       mul,  // NEW
		"max0":      max0, // NEW
		"pctMins":   pctMins,
		"hasProc":   hasProc,
		"hasSkill":  hasSkill,
		"minsToHM":  minsToHM,
		"procTitle": procTitle,
		"empClass":  empClass,
	}

	// Parse templates
	var err error
	tpl, err = template.New("").Funcs(funcs).ParseFiles("templates/index.gohtml")
	if err != nil {
		log.Fatal(err)
	}

	// Static
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/auto", handleAuto)
	http.HandleFunc("/api/reset", handleReset)
	log.Printf("Machine Shop Scheduler running on http://localhost:8080 …")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	currentAssignments = nil
	currentUtil = nil
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleAuto(w http.ResponseWriter, r *http.Request) {
	machs := demoMachines()
	emps := demoEmployees()
	jobs := demoJobs()

	currentAssignments, currentUtil = scheduleWeek(machs, emps, jobs)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	machs := demoMachines()
	emps := demoEmployees()
	jobs := demoJobs()

	now := time.Now()
	ws := startOfWeek(now)
	days := make([]TimeYMD, 7)
	for i := 0; i < 7; i++ {
		days[i] = TimeYMD{ws.AddDate(0, 0, i)}
	}

	jobBy := map[int]Job{}
	macBy := map[int]Machine{}
	empBy := map[int]Employee{}
	for _, j := range jobs {
		jobBy[j.ID] = j
	}
	for _, m := range machs {
		macBy[m.ID] = m
	}
	for _, e := range emps {
		empBy[e.ID] = e
	}

	colors := []string{
		"bg-rose-400",
		"bg-emerald-400",
		"bg-indigo-400",
		"bg-amber-400",
		"bg-cyan-400",
		"bg-fuchsia-400",
		"bg-lime-400",
	}

	empColors := map[int]string{}
	i := 0
	for _, e := range emps {
		empColors[e.ID] = colors[i%len(colors)]
		i++
	}

	// minutes already scheduled per job
	jobMinsDone := map[int]int{}
	for _, a := range currentAssignments {
		jobMinsDone[a.JobID] += a.Minutes
	}

	// only show jobs with remaining work
	backlog := make([]Job, 0, len(jobs))
	for _, j := range jobs {
		total := j.Qty * j.CycleMins
		if jobMinsDone[j.ID] < total { // remaining > 0
			backlog = append(backlog, j)
		}
	}

	vm := ViewModel{
		WeekStart:   TimeYMD{ws},
		Days:        days,
		Machines:    machs,
		Employees:   emps,
		Jobs:        jobs,
		Backlog:     backlog,
		Assignments: currentAssignments, // <-- use saved plan (may be empty)
		JobByID:     jobBy,
		MacByID:     macBy,
		EmpByID:     empBy,
		Util:        currentUtil,
		JobMinsDone: jobMinsDone,
		EmpColors:   empColors,
	}

	if err := tpl.ExecuteTemplate(w, "index.gohtml", vm); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
}
