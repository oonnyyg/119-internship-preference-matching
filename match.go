package main

import "fmt"

// ProposalRecord logs the outcome of a single proposal.
type ProposalRecord struct {
	Seq            int    `json:"seq"`
	Student        string `json:"student"`
	Project        string `json:"project"`
	Decision       string `json:"decision"` // "held", "rejected", "accepted-replacing"
	Reason         string `json:"reason"`
	Replaced       string `json:"replaced,omitempty"`
	ReplacedRecord int    `json:"replacedRecord,omitempty"`
}

// Assignment pairs one student with one project.
type Assignment struct {
	Student string `json:"student"`
	Project string `json:"project"`
}

// RemainingSlot reports spare capacity of a project.
type RemainingSlot struct {
	Project   string `json:"project"`
	Remaining int    `json:"remaining"`
}

// MatchReport is the JSON output of the match command.
type MatchReport struct {
	Assignments       []Assignment     `json:"assignments"`
	UnmatchedStudents []string         `json:"unmatchedStudents"`
	Remaining         []RemainingSlot  `json:"remainingCapacity"`
	Proposals         []ProposalRecord `json:"proposals"`
}

// Match runs the student-proposing deferred acceptance algorithm.
func Match(in *Input) *MatchReport {
	held := map[string][]string{}  // project -> held student IDs
	heldRecord := map[string]int{} // student -> seq of record where it was held
	nextChoice := map[string]int{} // student -> next preference index
	var records []ProposalRecord

	addRecord := func(student, project, decision, reason, replaced string, replacedRecord int) {
		rec := ProposalRecord{
			Seq:            len(records) + 1,
			Student:        student,
			Project:        project,
			Decision:       decision,
			Reason:         reason,
			Replaced:       replaced,
			ReplacedRecord: replacedRecord,
		}
		records = append(records, rec)
		if decision == "held" || decision == "accepted-replacing" {
			heldRecord[student] = rec.Seq
		}
	}

	students := map[string]Student{}
	for _, s := range in.Students {
		students[s.ID] = s
	}

	queue := in.sortedStudentIDs()
	for len(queue) > 0 {
		sid := queue[0]
		queue = queue[1:]
		s := students[sid]
		idx := nextChoice[sid]
		if idx >= len(s.Preferences) {
			continue // exhausted all preferences; stays unmatched
		}
		nextChoice[sid] = idx + 1
		pid := s.Preferences[idx]
		p := in.project(pid)

		if in.projectRank(pid, sid) < 0 {
			addRecord(sid, pid, "rejected",
				fmt.Sprintf("project %s does not list student %s as acceptable", pid, sid), "", 0)
			queue = append(queue, sid)
			continue
		}

		if len(held[pid]) < p.Capacity {
			addRecord(sid, pid, "held",
				fmt.Sprintf("project %s has a free slot", pid), "", 0)
			held[pid] = append(held[pid], sid)
			continue
		}

		// Project is full: find its worst held student.
		worst, worstRank := "", -1
		for _, h := range held[pid] {
			if r := in.projectRank(pid, h); r > worstRank {
				worstRank, worst = r, h
			}
		}
		if in.projectRank(pid, sid) < worstRank {
			replacedRecord := heldRecord[worst]
			addRecord(sid, pid, "accepted-replacing",
				fmt.Sprintf("project %s prefers %s over held student %s", pid, sid, worst),
				worst, replacedRecord)
			newHeld := []string{}
			for _, h := range held[pid] {
				if h != worst {
					newHeld = append(newHeld, h)
				}
			}
			held[pid] = append(newHeld, sid)
			delete(heldRecord, worst)
			queue = append(queue, worst)
		} else {
			addRecord(sid, pid, "rejected",
				fmt.Sprintf("project %s is full and prefers all held students over %s", pid, sid), "", 0)
			queue = append(queue, sid)
		}
	}

	report := &MatchReport{Proposals: records}
	if report.Proposals == nil {
		report.Proposals = []ProposalRecord{}
	}
	for _, pid := range sortedKeys(held) {
		for _, sid := range held[pid] {
			report.Assignments = append(report.Assignments, Assignment{Student: sid, Project: pid})
		}
	}
	if report.Assignments == nil {
		report.Assignments = []Assignment{}
	}
	assigned := map[string]bool{}
	for _, a := range report.Assignments {
		assigned[a.Student] = true
	}
	for _, sid := range in.sortedStudentIDs() {
		if !assigned[sid] {
			report.UnmatchedStudents = append(report.UnmatchedStudents, sid)
		}
	}
	if report.UnmatchedStudents == nil {
		report.UnmatchedStudents = []string{}
	}
	for _, p := range in.Projects {
		report.Remaining = append(report.Remaining, RemainingSlot{
			Project:   p.ID,
			Remaining: p.Capacity - len(held[p.ID]),
		})
	}
	return report
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
