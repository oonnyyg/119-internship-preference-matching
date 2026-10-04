package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

// Student describes one applicant and its ranked project preferences.
type Student struct {
	ID          string   `json:"id"`
	Preferences []string `json:"preferences"`
}

// Project describes one internship project, its capacity and its
// ranked order over acceptable students.
type Project struct {
	ID          string   `json:"id"`
	Capacity    int      `json:"capacity"`
	Preferences []string `json:"preferences"`
}

// Input is the top-level matching specification.
type Input struct {
	Students []Student `json:"students"`
	Projects []Project `json:"projects"`
}

// LoadInput reads and validates a matching specification.
func LoadInput(r io.Reader) (*Input, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var in Input
	if err := dec.Decode(&in); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := in.Validate(); err != nil {
		return nil, err
	}
	return &in, nil
}

// Validate enforces uniqueness, referential integrity and capacity rules.
func (in *Input) Validate() error {
	studentIDs := map[string]bool{}
	for _, s := range in.Students {
		if s.ID == "" {
			return fmt.Errorf("student with empty id")
		}
		if studentIDs[s.ID] {
			return fmt.Errorf("duplicate student id %q", s.ID)
		}
		studentIDs[s.ID] = true
	}
	projectIDs := map[string]bool{}
	for _, p := range in.Projects {
		if p.ID == "" {
			return fmt.Errorf("project with empty id")
		}
		if projectIDs[p.ID] {
			return fmt.Errorf("duplicate project id %q", p.ID)
		}
		projectIDs[p.ID] = true
		if p.Capacity < 0 {
			return fmt.Errorf("project %q has negative capacity %d", p.ID, p.Capacity)
		}
	}
	for _, s := range in.Students {
		seen := map[string]bool{}
		for _, pid := range s.Preferences {
			if !projectIDs[pid] {
				return fmt.Errorf("student %q references unknown project %q", s.ID, pid)
			}
			if seen[pid] {
				return fmt.Errorf("student %q lists project %q twice", s.ID, pid)
			}
			seen[pid] = true
		}
	}
	for _, p := range in.Projects {
		seen := map[string]bool{}
		for _, sid := range p.Preferences {
			if !studentIDs[sid] {
				return fmt.Errorf("project %q references unknown student %q", p.ID, sid)
			}
			if seen[sid] {
				return fmt.Errorf("project %q lists student %q twice", p.ID, sid)
			}
			seen[sid] = true
		}
	}
	return nil
}

// Acceptable reports whether both sides list each other.
func (in *Input) Acceptable(studentID, projectID string) bool {
	return in.studentRank(studentID, projectID) >= 0 && in.projectRank(projectID, studentID) >= 0
}

func (in *Input) studentRank(studentID, projectID string) int {
	for _, s := range in.Students {
		if s.ID == studentID {
			for i, pid := range s.Preferences {
				if pid == projectID {
					return i
				}
			}
			return -1
		}
	}
	return -1
}

func (in *Input) projectRank(projectID, studentID string) int {
	for _, p := range in.Projects {
		if p.ID == projectID {
			for i, sid := range p.Preferences {
				if sid == studentID {
					return i
				}
			}
			return -1
		}
	}
	return -1
}

func (in *Input) project(projectID string) *Project {
	for i := range in.Projects {
		if in.Projects[i].ID == projectID {
			return &in.Projects[i]
		}
	}
	return nil
}

// sortedStudentIDs returns student IDs in lexicographic order so the
// result does not depend on the ordering of the input arrays.
func (in *Input) sortedStudentIDs() []string {
	ids := make([]string, 0, len(in.Students))
	for _, s := range in.Students {
		ids = append(ids, s.ID)
	}
	sort.Strings(ids)
	return ids
}
