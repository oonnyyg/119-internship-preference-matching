package main

import "sort"

// Record is one proposal event in the deferred-acceptance log.
type Record struct {
	Seq            int    `json:"seq"`
	Student        string `json:"student"`
	Project        string `json:"project"`
	Action         string `json:"action"` // "held", "rejected", "replaced"
	Reason         string `json:"reason"`
	ReplacedRecord int    `json:"replaced_record,omitempty"` // seq of the displaced held record
}

type Assignment struct {
	Student string `json:"student"`
	Project string `json:"project"`
}

type MatchReport struct {
	Assignments       []Assignment   `json:"assignments"`
	UnmatchedStudents []string       `json:"unmatched_students"`
	RemainingCapacity map[string]int `json:"remaining_capacity"`
	Records           []Record       `json:"records"`
}

// match runs student-proposing deferred acceptance. Students enter the
// proposal queue in lexicographic id order; displaced students rejoin the
// back of the queue, so input object ordering never affects the result.
func match(v *Validated) *MatchReport {
	ids := make([]string, 0, len(v.Students))
	for _, s := range v.Students {
		ids = append(ids, s.ID)
	}
	sort.Strings(ids)

	nextIdx := map[string]int{}            // student -> next preference index
	held := map[string][]string{}          // project -> held student ids
	heldSeq := map[string]map[string]int{} // project -> student -> record seq
	matched := map[string]string{}
	var records []Record
	seq := 0

	queue := append([]string(nil), ids...)
	for len(queue) > 0 {
		sid := queue[0]
		queue = queue[1:]
		sprefs := studentPrefs(v, sid)
		for nextIdx[sid] < len(sprefs) {
			pid := sprefs[nextIdx[sid]]
			nextIdx[sid]++
			if _, ok := v.projectRank[pid][sid]; !ok {
				continue // project does not list this student
			}
			seq++
			capOf := capacityOf(v, pid)
			current := held[pid]
			if len(current) < capOf {
				records = append(records, Record{Seq: seq, Student: sid, Project: pid,
					Action: "held", Reason: "project has free capacity"})
				held[pid] = append(current, sid)
				if heldSeq[pid] == nil {
					heldSeq[pid] = map[string]int{}
				}
				heldSeq[pid][sid] = seq
				matched[sid] = pid
				break
			}
			worst, worstRank := "", -1
			for _, hid := range current {
				if r := v.projectRank[pid][hid]; r > worstRank {
					worstRank, worst = r, hid
				}
			}
			if v.projectRank[pid][sid] < worstRank {
				records = append(records, Record{Seq: seq, Student: sid, Project: pid,
					Action: "replaced",
					Reason: "project full; project prefers this student over worst held " + worst,
					ReplacedRecord: heldSeq[pid][worst]})
				next := append([]string(nil), current...)
				for i, hid := range next {
					if hid == worst {
						next[i] = sid
					}
				}
				held[pid] = next
				delete(heldSeq[pid], worst)
				heldSeq[pid][sid] = seq
				delete(matched, worst)
				matched[sid] = pid
				queue = append(queue, worst) // displaced student resumes later
				break
			}
			records = append(records, Record{Seq: seq, Student: sid, Project: pid,
				Action: "rejected", Reason: "project full and prefers all held students"})
		}
	}

	report := &MatchReport{
		Assignments:       []Assignment{},
		RemainingCapacity: map[string]int{},
		Records:           records,
	}
	for sid, pid := range matched {
		report.Assignments = append(report.Assignments, Assignment{Student: sid, Project: pid})
	}
	sort.Slice(report.Assignments, func(i, j int) bool {
		return report.Assignments[i].Student < report.Assignments[j].Student
	})
	for _, s := range v.Students {
		if _, ok := matched[s.ID]; !ok {
			report.UnmatchedStudents = append(report.UnmatchedStudents, s.ID)
		}
	}
	sort.Strings(report.UnmatchedStudents)
	for _, p := range v.Projects {
		report.RemainingCapacity[p.ID] = p.Capacity - len(held[p.ID])
	}
	return report
}

func studentPrefs(v *Validated, sid string) []string {
	for _, s := range v.Students {
		if s.ID == sid {
			return s.Preferences
		}
	}
	return nil
}

func capacityOf(v *Validated, pid string) int {
	for _, p := range v.Projects {
		if p.ID == pid {
			return p.Capacity
		}
	}
	return 0
}
