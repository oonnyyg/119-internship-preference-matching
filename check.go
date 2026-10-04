package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// CandidateAssignment is one pair in an externally submitted assignment.
type CandidateAssignment struct {
	Student string `json:"student"`
	Project string `json:"project"`
}

type CheckInput struct {
	Instance
	Assignments []CandidateAssignment `json:"assignments"`
}

type BlockingPair struct {
	Student string `json:"student"`
	Project string `json:"project"`
	Reason  string `json:"reason"`
}

type CheckReport struct {
	Valid         bool            `json:"valid"`
	Errors        []string        `json:"errors"`
	BlockingPairs []BlockingPair  `json:"blocking_pairs"`
	Stable        bool            `json:"stable"`
}

func decodeCheckInput(r io.Reader) (*CheckInput, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var in CheckInput
	if err := dec.Decode(&in); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return &in, nil
}

// check verifies structural validity of a candidate assignment and finds
// blocking pairs. An unmatched student counts as worse off than any
// acceptable project.
func check(v *Validated, in *CheckInput) *CheckReport {
	rep := &CheckReport{Valid: true, Errors: []string{}, BlockingPairs: []BlockingPair{}}

	assigned := map[string]string{} // student -> project
	load := map[string][]string{}   // project -> students
	for _, a := range in.Assignments {
		if _, ok := v.studentRank[a.Student]; !ok {
			rep.Errors = append(rep.Errors, fmt.Sprintf("unknown student %q", a.Student))
			continue
		}
		if _, ok := v.projectRank[a.Project]; !ok {
			rep.Errors = append(rep.Errors, fmt.Sprintf("unknown project %q", a.Project))
			continue
		}
		if _, dup := assigned[a.Student]; dup {
			rep.Errors = append(rep.Errors, fmt.Sprintf("student %q assigned more than once", a.Student))
			continue
		}
		if !v.mutuallyAcceptable(a.Student, a.Project) {
			rep.Errors = append(rep.Errors,
				fmt.Sprintf("student %q and project %q do not list each other", a.Student, a.Project))
			continue
		}
		assigned[a.Student] = a.Project
		load[a.Project] = append(load[a.Project], a.Student)
	}
	for _, p := range v.Projects {
		if len(load[p.ID]) > p.Capacity {
			rep.Errors = append(rep.Errors,
				fmt.Sprintf("project %q over capacity: %d assigned, capacity %d",
					p.ID, len(load[p.ID]), p.Capacity))
		}
	}
	if len(rep.Errors) > 0 {
		rep.Valid = false
	}

	// Blocking pairs: student prefers project over current assignment (any
	// acceptable project beats being unmatched), and the project has a free
	// slot or prefers the student over a current assignee.
	for _, s := range v.Students {
		cur, has := assigned[s.ID]
		curRank := len(s.Preferences) // unmatched: below every acceptable project
		if has {
			curRank = v.studentRank[s.ID][cur]
		}
		for _, p := range v.Projects {
			if !v.mutuallyAcceptable(s.ID, p.ID) {
				continue
			}
			if has && p.ID == cur {
				continue
			}
			if v.studentRank[s.ID][p.ID] >= curRank {
				continue
			}
			if len(load[p.ID]) < p.Capacity {
				rep.BlockingPairs = append(rep.BlockingPairs, BlockingPair{
					Student: s.ID, Project: p.ID,
					Reason: "student prefers project over current assignment and project has free capacity"})
				continue
			}
			for _, hid := range load[p.ID] {
				if v.projectRank[p.ID][s.ID] < v.projectRank[p.ID][hid] {
					rep.BlockingPairs = append(rep.BlockingPairs, BlockingPair{
						Student: s.ID, Project: p.ID,
						Reason: fmt.Sprintf("student prefers project and project prefers student over current assignee %q", hid)})
					break
				}
			}
		}
	}
	sort.Slice(rep.BlockingPairs, func(i, j int) bool {
		a, b := rep.BlockingPairs[i], rep.BlockingPairs[j]
		if a.Student != b.Student {
			return a.Student < b.Student
		}
		return a.Project < b.Project
	})
	rep.Stable = rep.Valid && len(rep.BlockingPairs) == 0
	return rep
}
