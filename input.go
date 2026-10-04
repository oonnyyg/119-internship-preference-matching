package main

import (
	"encoding/json"
	"fmt"
	"io"
)

type Student struct {
	ID          string   `json:"id"`
	Preferences []string `json:"preferences"`
}

type Project struct {
	ID          string   `json:"id"`
	Capacity    int      `json:"capacity"`
	Preferences []string `json:"preferences"`
}

type Instance struct {
	Students []Student `json:"students"`
	Projects []Project `json:"projects"`
}

// Validated holds an instance with derived lookup structures.
type Validated struct {
	Instance
	studentRank map[string]map[string]int // student -> project -> rank (0 best)
	projectRank map[string]map[string]int // project -> student -> rank
}

func decodeInstance(r io.Reader) (*Instance, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var inst Instance
	if err := dec.Decode(&inst); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return &inst, nil
}

func validate(inst *Instance) (*Validated, error) {
	v := &Validated{
		Instance:    *inst,
		studentRank: map[string]map[string]int{},
		projectRank: map[string]map[string]int{},
	}
	projectIDs := map[string]bool{}
	for _, p := range inst.Projects {
		if p.ID == "" {
			return nil, fmt.Errorf("project with empty id")
		}
		if projectIDs[p.ID] {
			return nil, fmt.Errorf("duplicate project id %q", p.ID)
		}
		projectIDs[p.ID] = true
		if p.Capacity < 0 {
			return nil, fmt.Errorf("project %q has negative capacity %d", p.ID, p.Capacity)
		}
	}
	studentIDs := map[string]bool{}
	for _, s := range inst.Students {
		if s.ID == "" {
			return nil, fmt.Errorf("student with empty id")
		}
		if studentIDs[s.ID] {
			return nil, fmt.Errorf("duplicate student id %q", s.ID)
		}
		studentIDs[s.ID] = true
	}
	for _, s := range inst.Students {
		rank := map[string]int{}
		for i, pid := range s.Preferences {
			if !projectIDs[pid] {
				return nil, fmt.Errorf("student %q references unknown project %q", s.ID, pid)
			}
			if _, dup := rank[pid]; dup {
				return nil, fmt.Errorf("student %q lists project %q twice", s.ID, pid)
			}
			rank[pid] = i
		}
		v.studentRank[s.ID] = rank
	}
	for _, p := range inst.Projects {
		rank := map[string]int{}
		for i, sid := range p.Preferences {
			if !studentIDs[sid] {
				return nil, fmt.Errorf("project %q references unknown student %q", p.ID, sid)
			}
			if _, dup := rank[sid]; dup {
				return nil, fmt.Errorf("project %q lists student %q twice", p.ID, sid)
			}
			rank[sid] = i
		}
		v.projectRank[p.ID] = rank
	}
	return v, nil
}

// mutuallyAcceptable reports whether s and p list each other.
func (v *Validated) mutuallyAcceptable(sid, pid string) bool {
	_, sOk := v.studentRank[sid][pid]
	_, pOk := v.projectRank[pid][sid]
	return sOk && pOk
}
