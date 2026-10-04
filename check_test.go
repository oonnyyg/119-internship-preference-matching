package main

import "testing"

func checkFixture(t *testing.T) *Input {
	return mustInput(t, &Input{
		Students: []Student{
			{ID: "s1", Preferences: []string{"p1", "p2"}},
			{ID: "s2", Preferences: []string{"p1", "p2"}},
		},
		Projects: []Project{
			{ID: "p1", Capacity: 1, Preferences: []string{"s1", "s2"}},
			{ID: "p2", Capacity: 1, Preferences: []string{"s1", "s2"}},
		},
	})
}

func TestCheckStableAssignment(t *testing.T) {
	in := checkFixture(t)
	rep := Check(in, &AssignmentFile{Assignments: []Assignment{
		{Student: "s1", Project: "p1"},
		{Student: "s2", Project: "p2"},
	}})
	if !rep.Valid || !rep.Stable {
		t.Fatalf("expected valid and stable, got %+v", rep)
	}
}

func TestCheckBlockingPairFreeSlot(t *testing.T) {
	in := checkFixture(t)
	rep := Check(in, &AssignmentFile{Assignments: []Assignment{
		{Student: "s1", Project: "p2"},
	}})
	if rep.Stable {
		t.Fatal("expected instability")
	}
	if len(rep.BlockingPairs) == 0 || rep.BlockingPairs[0].Student != "s1" || rep.BlockingPairs[0].Project != "p1" {
		t.Fatalf("blocking pairs = %+v", rep.BlockingPairs)
	}
}

func TestCheckBlockingPairPreferredStudent(t *testing.T) {
	in := checkFixture(t)
	// p1 is full with s2, but p1 prefers s1 who is matched to worse p2.
	rep := Check(in, &AssignmentFile{Assignments: []Assignment{
		{Student: "s1", Project: "p2"},
		{Student: "s2", Project: "p1"},
	}})
	if rep.Stable {
		t.Fatal("expected instability")
	}
	found := false
	for _, bp := range rep.BlockingPairs {
		if bp.Student == "s1" && bp.Project == "p1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("blocking pairs = %+v", rep.BlockingPairs)
	}
}

func TestCheckCapacityViolation(t *testing.T) {
	in := checkFixture(t)
	rep := Check(in, &AssignmentFile{Assignments: []Assignment{
		{Student: "s1", Project: "p1"},
		{Student: "s2", Project: "p1"},
	}})
	if rep.Valid {
		t.Fatal("expected capacity violation")
	}
}

func TestCheckUnacceptablePair(t *testing.T) {
	in := mustInput(t, &Input{
		Students: []Student{{ID: "s1", Preferences: []string{"p1"}}},
		Projects: []Project{{ID: "p1", Capacity: 1, Preferences: []string{}}},
	})
	rep := Check(in, &AssignmentFile{Assignments: []Assignment{
		{Student: "s1", Project: "p1"},
	}})
	if rep.Valid {
		t.Fatal("expected acceptability error")
	}
}

func TestCheckDuplicateAndUnknown(t *testing.T) {
	in := checkFixture(t)
	rep := Check(in, &AssignmentFile{Assignments: []Assignment{
		{Student: "s1", Project: "p1"},
		{Student: "s1", Project: "p2"},
		{Student: "ghost", Project: "p1"},
	}})
	if rep.Valid {
		t.Fatal("expected errors for duplicate and unknown student")
	}
	if len(rep.Errors) < 2 {
		t.Fatalf("errors = %v", rep.Errors)
	}
}
