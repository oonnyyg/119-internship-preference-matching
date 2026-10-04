package main

import (
	"reflect"
	"sort"
	"testing"
)

func mustInput(t *testing.T, in *Input) *Input {
	t.Helper()
	if err := in.Validate(); err != nil {
		t.Fatalf("invalid test input: %v", err)
	}
	return in
}

func assignmentMap(rep *MatchReport) map[string]string {
	m := map[string]string{}
	for _, a := range rep.Assignments {
		m[a.Student] = a.Project
	}
	return m
}

func TestBasicMatch(t *testing.T) {
	in := mustInput(t, &Input{
		Students: []Student{
			{ID: "s1", Preferences: []string{"p1", "p2"}},
			{ID: "s2", Preferences: []string{"p1", "p2"}},
			{ID: "s3", Preferences: []string{"p2"}},
		},
		Projects: []Project{
			{ID: "p1", Capacity: 1, Preferences: []string{"s2", "s1"}},
			{ID: "p2", Capacity: 1, Preferences: []string{"s1", "s2", "s3"}},
		},
	})
	rep := Match(in)
	want := map[string]string{"s2": "p1", "s1": "p2"}
	if got := assignmentMap(rep); !reflect.DeepEqual(got, want) {
		t.Fatalf("assignments = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(rep.UnmatchedStudents, []string{"s3"}) {
		t.Fatalf("unmatched = %v", rep.UnmatchedStudents)
	}
	for _, r := range rep.Remaining {
		if r.Remaining != 0 {
			t.Fatalf("project %s remaining = %d, want 0", r.Project, r.Remaining)
		}
	}
}

func TestReplacementChain(t *testing.T) {
	in := mustInput(t, &Input{
		Students: []Student{
			{ID: "s1", Preferences: []string{"p1"}},
			{ID: "s2", Preferences: []string{"p1", "p2"}},
			{ID: "s3", Preferences: []string{"p1", "p2"}},
		},
		Projects: []Project{
			{ID: "p1", Capacity: 1, Preferences: []string{"s3", "s2", "s1"}},
			{ID: "p2", Capacity: 1, Preferences: []string{"s2", "s3"}},
		},
	})
	rep := Match(in)
	want := map[string]string{"s3": "p1", "s2": "p2"}
	if got := assignmentMap(rep); !reflect.DeepEqual(got, want) {
		t.Fatalf("assignments = %v, want %v", got, want)
	}
	var replacing *ProposalRecord
	for i := range rep.Proposals {
		if rep.Proposals[i].Decision == "accepted-replacing" {
			replacing = &rep.Proposals[i]
		}
	}
	if replacing == nil {
		t.Fatal("expected an accepted-replacing record")
	}
	if replacing.Replaced == "" || replacing.ReplacedRecord == 0 {
		t.Fatalf("replacement record missing link: %+v", replacing)
	}
	linked := rep.Proposals[replacing.ReplacedRecord-1]
	if linked.Student != replacing.Replaced || linked.Project != replacing.Project {
		t.Fatalf("replacedRecord %d points to %+v, want held record of %s",
			replacing.ReplacedRecord, linked, replacing.Replaced)
	}
}

func TestZeroCapacityAndUnmatched(t *testing.T) {
	in := mustInput(t, &Input{
		Students: []Student{
			{ID: "s1", Preferences: []string{"p1", "p2"}},
		},
		Projects: []Project{
			{ID: "p1", Capacity: 0, Preferences: []string{"s1"}},
			{ID: "p2", Capacity: 1, Preferences: []string{"s1"}},
		},
	})
	rep := Match(in)
	if got := assignmentMap(rep); got["s1"] != "p2" {
		t.Fatalf("assignments = %v", got)
	}
	for _, r := range rep.Remaining {
		if r.Project == "p1" && r.Remaining != 0 {
			t.Fatalf("zero-capacity project remaining = %d", r.Remaining)
		}
	}
}

func TestAsymmetricLists(t *testing.T) {
	// s1 lists p1, but p1 does not list s1: no match allowed there.
	in := mustInput(t, &Input{
		Students: []Student{
			{ID: "s1", Preferences: []string{"p1", "p2"}},
			{ID: "s2", Preferences: []string{"p1"}},
		},
		Projects: []Project{
			{ID: "p1", Capacity: 1, Preferences: []string{"s2"}},
			{ID: "p2", Capacity: 1, Preferences: []string{"s1"}},
		},
	})
	rep := Match(in)
	want := map[string]string{"s2": "p1", "s1": "p2"}
	if got := assignmentMap(rep); !reflect.DeepEqual(got, want) {
		t.Fatalf("assignments = %v, want %v", got, want)
	}
	var sawRejection bool
	for _, r := range rep.Proposals {
		if r.Student == "s1" && r.Project == "p1" && r.Decision == "rejected" {
			sawRejection = true
		}
	}
	if !sawRejection {
		t.Fatal("expected rejection record for s1 -> p1")
	}
}

func TestInputOrderIndependence(t *testing.T) {
	base := &Input{
		Students: []Student{
			{ID: "s1", Preferences: []string{"p1", "p2"}},
			{ID: "s2", Preferences: []string{"p2", "p1"}},
			{ID: "s3", Preferences: []string{"p1"}},
		},
		Projects: []Project{
			{ID: "p1", Capacity: 1, Preferences: []string{"s1", "s3", "s2"}},
			{ID: "p2", Capacity: 1, Preferences: []string{"s2", "s1"}},
		},
	}
	shuffled := &Input{
		Students: []Student{base.Students[2], base.Students[0], base.Students[1]},
		Projects: []Project{base.Projects[1], base.Projects[0]},
	}
	got1 := assignmentMap(Match(mustInput(t, base)))
	got2 := assignmentMap(Match(mustInput(t, shuffled)))
	if !reflect.DeepEqual(got1, got2) {
		t.Fatalf("order changed result: %v vs %v", got1, got2)
	}
}

func TestValidationErrors(t *testing.T) {
	cases := []Input{
		{Students: []Student{{ID: "s1"}, {ID: "s1"}}},
		{Projects: []Project{{ID: "p1", Capacity: -1}}},
		{Students: []Student{{ID: "s1", Preferences: []string{"nope"}}}},
		{Projects: []Project{{ID: "p1", Preferences: []string{"ghost"}}}},
		{Students: []Student{{ID: "s1", Preferences: []string{"p1", "p1"}}},
			Projects: []Project{{ID: "p1"}}},
	}
	for i, in := range cases {
		if err := in.Validate(); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
}

func TestMatchResultIsStable(t *testing.T) {
	in := mustInput(t, &Input{
		Students: []Student{
			{ID: "s1", Preferences: []string{"p1", "p2"}},
			{ID: "s2", Preferences: []string{"p1", "p2"}},
			{ID: "s3", Preferences: []string{"p2", "p1"}},
		},
		Projects: []Project{
			{ID: "p1", Capacity: 1, Preferences: []string{"s3", "s1", "s2"}},
			{ID: "p2", Capacity: 2, Preferences: []string{"s1", "s2", "s3"}},
		},
	})
	rep := Match(in)
	chk := Check(in, &AssignmentFile{Assignments: rep.Assignments})
	if !chk.Valid || !chk.Stable {
		t.Fatalf("match result failed check: %+v", chk)
	}
	_ = sort.Strings
}
