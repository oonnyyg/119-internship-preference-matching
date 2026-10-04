package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// AssignmentFile is the candidate assignment submitted for checking.
type AssignmentFile struct {
	Assignments []Assignment `json:"assignments"`
}

// BlockingPair describes an unstable student/project pair.
type BlockingPair struct {
	Student string `json:"student"`
	Project string `json:"project"`
	Reason  string `json:"reason"`
}

// CheckReport is the JSON output of the check command.
type CheckReport struct {
	Valid         bool           `json:"valid"`
	Stable        bool           `json:"stable"`
	Errors        []string       `json:"errors"`
	BlockingPairs []BlockingPair `json:"blockingPairs"`
}

// LoadAssignments reads a candidate assignment file.
func LoadAssignments(r io.Reader) (*AssignmentFile, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var af AssignmentFile
	if err := dec.Decode(&af); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return &af, nil
}

// Check validates a candidate assignment and searches for blocking pairs.
func Check(in *Input, af *AssignmentFile) *CheckReport {
	rep := &CheckReport{Valid: true, Stable: true}
	matched := map[string]string{}
	load := map[string][]string{}
	for _, a := range af.Assignments {
		if in.project(a.Project) == nil {
			rep.Errors = append(rep.Errors, fmt.Sprintf("unknown project %q", a.Project))
			continue
		}
		if _, ok := studentByID(in, a.Student); !ok {
			rep.Errors = append(rep.Errors, fmt.Sprintf("unknown student %q", a.Student))
			continue
		}
		if _, dup := matched[a.Student]; dup {
			rep.Errors = append(rep.Errors, fmt.Sprintf("student %q assigned more than once", a.Student))
			continue
		}
		if !in.Acceptable(a.Student, a.Project) {
			rep.Errors = append(rep.Errors,
				fmt.Sprintf("student %q and project %q do not list each other as acceptable", a.Student, a.Project))
		}
		matched[a.Student] = a.Project
		load[a.Project] = append(load[a.Project], a.Student)
	}
	for _, p := range in.Projects {
		if len(load[p.ID]) > p.Capacity {
			rep.Errors = append(rep.Errors,
				fmt.Sprintf("project %q has %d assignees but capacity %d", p.ID, len(load[p.ID]), p.Capacity))
		}
	}
	if len(rep.Errors) > 0 {
		rep.Valid = false
	}
	for _, s := range in.Students {
		current, has := matched[s.ID]
		currentRank := len(s.Preferences)
		if has {
			currentRank = in.studentRank(s.ID, current)
		}
		for _, pid := range s.Preferences {
			if in.studentRank(s.ID, pid) >= currentRank {
				continue
			}
			if !in.Acceptable(s.ID, pid) {
				continue
			}
			p := in.project(pid)
			if len(load[pid]) < p.Capacity {
				rep.BlockingPairs = append(rep.BlockingPairs, BlockingPair{
					Student: s.ID, Project: pid,
					Reason: fmt.Sprintf("student %s prefers %s and the project has a free slot", s.ID, pid),
				})
				continue
			}
			for _, h := range load[pid] {
				if in.projectRank(pid, s.ID) < in.projectRank(pid, h) {
					rep.BlockingPairs = append(rep.BlockingPairs, BlockingPair{
						Student: s.ID, Project: pid,
						Reason: fmt.Sprintf("student %s prefers %s and the project prefers %s over %s", s.ID, pid, s.ID, h),
					})
					break
				}
			}
		}
	}
	if rep.BlockingPairs == nil {
		rep.BlockingPairs = []BlockingPair{}
	}
	if len(rep.BlockingPairs) > 0 {
		rep.Stable = false
	}
	sort.Slice(rep.BlockingPairs, func(i, j int) bool {
		if rep.BlockingPairs[i].Student != rep.BlockingPairs[j].Student {
			return rep.BlockingPairs[i].Student < rep.BlockingPairs[j].Student
		}
		return rep.BlockingPairs[i].Project < rep.BlockingPairs[j].Project
	})
	if rep.Errors == nil {
		rep.Errors = []string{}
	}
	return rep
}

func studentByID(in *Input, id string) (Student, bool) {
	for _, s := range in.Students {
		if s.ID == id {
			return s, true
		}
	}
	return Student{}, false
}
