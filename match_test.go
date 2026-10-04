package main

import (
	"reflect"
	"testing"
)

func mustValidate(t *testing.T, inst *Instance) *Validated {
	t.Helper()
	v, err := validate(inst)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	return v
}

func assignmentMap(rep *MatchReport) map[string]string {
	m := map[string]string{}
	for _, a := range rep.Assignments {
		m[a.Student] = a.Project
	}
	return m
}

// TestReplacementChain: s3 displaces s2 at p1, s2 then falls back to p2.
func TestReplacementChain(t *testing.T) {
	inst := &Instance{
		Students: []Student{
			{ID: "s1", Preferences: []string{"p1"}},
			{ID: "s2", Preferences: []string{"p1", "p2"}},
			{ID: "s3", Preferences: []string{"p1"}},
		},
		Projects: []Project{
			{ID: "p1", Capacity: 2, Preferences: []string{"s1", "s3", "s2"}},
			{ID: "p2", Capacity: 1, Preferences: []string{"s2"}},
		},
	}
	rep := match(mustValidate(t, inst))
	got := assignmentMap(rep)
	want := map[string]string{"s1": "p1", "s2": "p2", "s3": "p1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("assignments = %v, want %v", got, want)
	}
	// The replacement record must reference the displaced held record.
	var replaced, held *Record
	for i := range rep.Records {
		r := &rep.Records[i]
		if r.Action == "held" && r.Student == "s2" && r.Project == "p1" {
			held = r
		}
		if r.Action == "replaced" {
			replaced = r
		}
	}
	if held == nil || replaced == nil {
		t.Fatalf("expected held and replaced records, got %+v", rep.Records)
	}
	if replaced.ReplacedRecord != held.Seq {
		t.Fatalf("replaced record references seq %d, want %d", replaced.ReplacedRecord, held.Seq)
	}
	if replaced.Student != "s3" {
		t.Fatalf("replacing student = %q, want s3", replaced.Student)
	}
}

// TestRecordsReconcile: replaying held/replaced records yields the final assignment.
func TestRecordsReconcile(t *testing.T) {
	inst := &Instance{
		Students: []Student{
			{ID: "a", Preferences: []string{"p1", "p2"}},
			{ID: "b", Preferences: []string{"p1", "p2"}},
			{ID: "c", Preferences: []string{"p2", "p1"}},
		},
		Projects: []Project{
			{ID: "p1", Capacity: 1, Preferences: []string{"c", "b", "a"}},
			{ID: "p2", Capacity: 1, Preferences: []string{"a", "b", "c"}},
		},
	}
	rep := match(mustValidate(t, inst))
	live := map[int]bool{} // record seq -> still held
	owner := map[int]string{}
	final := map[string]string{}
	for _, r := range rep.Records {
		switch r.Action {
		case "held":
			live[r.Seq] = true
			owner[r.Seq] = r.Student
			final[r.Student] = r.Project
		case "replaced":
			if !live[r.ReplacedRecord] {
				t.Fatalf("record %d replaces non-live record %d", r.Seq, r.ReplacedRecord)
			}
			delete(live, r.ReplacedRecord)
			delete(final, owner[r.ReplacedRecord])
			live[r.Seq] = true
			owner[r.Seq] = r.Student
			final[r.Student] = r.Project
		}
	}
	if !reflect.DeepEqual(final, assignmentMap(rep)) {
		t.Fatalf("replayed %v != reported %v", final, assignmentMap(rep))
	}
}

// TestAsymmetricLists: pairs listed by only one side must not match.
func TestAsymmetricLists(t *testing.T) {
	inst := &Instance{
		Students: []Student{
			{ID: "s1", Preferences: []string{"p1", "p2"}},
			{ID: "s2", Preferences: []string{"p1"}},
		},
		Projects: []Project{
			{ID: "p1", Capacity: 1, Preferences: []string{"s1"}}, // does not list s2
			{ID: "p2", Capacity: 1, Preferences: []string{}},     // lists nobody
		},
	}
	rep := match(mustValidate(t, inst))
	got := assignmentMap(rep)
	want := map[string]string{"s1": "p1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("assignments = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(rep.UnmatchedStudents, []string{"s2"}) {
		t.Fatalf("unmatched = %v, want [s2]", rep.UnmatchedStudents)
	}
	if rep.RemainingCapacity["p2"] != 1 {
		t.Fatalf("p2 remaining = %d, want 1", rep.RemainingCapacity["p2"])
	}
}

// TestZeroCapacityAndUnmatched: zero-capacity projects hold nobody.
func TestZeroCapacityAndUnmatched(t *testing.T) {
	inst := &Instance{
		Students: []Student{{ID: "s1", Preferences: []string{"p1", "p2"}}},
		Projects: []Project{
			{ID: "p1", Capacity: 0, Preferences: []string{"s1"}},
			{ID: "p2", Capacity: 1, Preferences: []string{"s1"}},
		},
	}
	rep := match(mustValidate(t, inst))
	if got := assignmentMap(rep); got["s1"] != "p2" {
		t.Fatalf("assignments = %v", got)
	}
	if rep.RemainingCapacity["p1"] != 0 {
		t.Fatalf("p1 remaining = %d, want 0", rep.RemainingCapacity["p1"])
	}
	var sawReject bool
	for _, r := range rep.Records {
		if r.Project == "p1" && r.Action == "rejected" {
			sawReject = true
		}
	}
	if !sawReject {
		t.Fatalf("expected rejection record for zero-capacity p1: %+v", rep.Records)
	}
}

// TestInputOrderInvariant: permuting input arrays must not change the result.
func TestInputOrderInvariant(t *testing.T) {
	base := &Instance{
		Students: []Student{
			{ID: "s1", Preferences: []string{"p1", "p2"}},
			{ID: "s2", Preferences: []string{"p1", "p2"}},
			{ID: "s3", Preferences: []string{"p2", "p1"}},
		},
		Projects: []Project{
			{ID: "p1", Capacity: 1, Preferences: []string{"s2", "s1", "s3"}},
			{ID: "p2", Capacity: 1, Preferences: []string{"s3", "s1", "s2"}},
		},
	}
	permuted := &Instance{
		Students: []Student{base.Students[2], base.Students[0], base.Students[1]},
		Projects: []Project{base.Projects[1], base.Projects[0]},
	}
	r1 := match(mustValidate(t, base))
	r2 := match(mustValidate(t, permuted))
	if !reflect.DeepEqual(r1, r2) {
		t.Fatalf("order changed result:\n%+v\n%+v", r1, r2)
	}
}

func TestValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		inst Instance
	}{
		{"dup student", Instance{Students: []Student{{ID: "s1"}, {ID: "s1"}}}},
		{"dup project", Instance{Projects: []Project{{ID: "p1"}, {ID: "p1"}}}},
		{"negative capacity", Instance{Projects: []Project{{ID: "p1", Capacity: -1}}}},
		{"unknown project ref", Instance{
			Students: []Student{{ID: "s1", Preferences: []string{"nope"}}},
		}},
		{"unknown student ref", Instance{
			Students: []Student{{ID: "s1"}},
			Projects: []Project{{ID: "p1", Preferences: []string{"nope"}}},
		}},
		{"dup preference", Instance{
			Students: []Student{{ID: "s1", Preferences: []string{"p1", "p1"}}},
			Projects: []Project{{ID: "p1"}},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := validate(&tc.inst); err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}
