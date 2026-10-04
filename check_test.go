package main

import (
	"testing"
)

func checkFixture() *Instance {
	return &Instance{
		Students: []Student{
			{ID: "s1", Preferences: []string{"p1", "p2"}},
			{ID: "s2", Preferences: []string{"p1", "p2"}},
		},
		Projects: []Project{
			{ID: "p1", Capacity: 1, Preferences: []string{"s1", "s2"}},
			{ID: "p2", Capacity: 1, Preferences: []string{"s1", "s2"}},
		},
	}
}

func runCheck(t *testing.T, inst *Instance, as []CandidateAssignment) *CheckReport {
	t.Helper()
	v := mustValidate(t, inst)
	return check(v, &CheckInput{Instance: *inst, Assignments: as})
}

func TestCheckStableAssignment(t *testing.T) {
	rep := runCheck(t, checkFixture(), []CandidateAssignment{
		{Student: "s1", Project: "p1"},
		{Student: "s2", Project: "p2"},
	})
	if !rep.Valid || !rep.Stable {
		t.Fatalf("expected valid+stable, got %+v", rep)
	}
}

// TestCheckUnstableAssignment: s2 holds p1 while s1 (preferred by p1) sits at
// its lower choice p2, and s2 prefers p1-held slot... construct blocking pair:
// s1 at p2, s2 at p1 -> s1 prefers p1, p1 prefers s1 over s2: blocking.
func TestCheckUnstableAssignment(t *testing.T) {
	rep := runCheck(t, checkFixture(), []CandidateAssignment{
		{Student: "s1", Project: "p2"},
		{Student: "s2", Project: "p1"},
	})
	if !rep.Valid {
		t.Fatalf("expected structurally valid, errors=%v", rep.Errors)
	}
	if rep.Stable {
		t.Fatalf("expected unstable")
	}
	if len(rep.BlockingPairs) != 1 {
		t.Fatalf("blocking pairs = %+v", rep.BlockingPairs)
	}
	bp := rep.BlockingPairs[0]
	if bp.Student != "s1" || bp.Project != "p1" {
		t.Fatalf("blocking pair = %+v, want s1/p1", bp)
	}
}

// TestCheckBlockingViaFreeSlot: unmatched student + free acceptable slot blocks.
func TestCheckBlockingViaFreeSlot(t *testing.T) {
	rep := runCheck(t, checkFixture(), []CandidateAssignment{
		{Student: "s1", Project: "p1"},
	})
	if rep.Stable {
		t.Fatalf("expected unstable: s2 unmatched with free p2 slot")
	}
	found := false
	for _, bp := range rep.BlockingPairs {
		if bp.Student == "s2" && bp.Project == "p2" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing blocking pair s2/p2: %+v", rep.BlockingPairs)
	}
}

func TestCheckStructuralErrors(t *testing.T) {
	rep := runCheck(t, checkFixture(), []CandidateAssignment{
		{Student: "s1", Project: "p1"},
		{Student: "s1", Project: "p2"}, // duplicate student
		{Student: "ghost", Project: "p1"},
	})
	if rep.Valid {
		t.Fatalf("expected invalid, got %+v", rep)
	}
	if len(rep.Errors) < 2 {
		t.Fatalf("errors = %v", rep.Errors)
	}
}

func TestCheckCapacityViolation(t *testing.T) {
	rep := runCheck(t, checkFixture(), []CandidateAssignment{
		{Student: "s1", Project: "p1"},
		{Student: "s2", Project: "p1"}, // p1 capacity is 1
	})
	if rep.Valid {
		t.Fatalf("expected capacity error")
	}
}

func TestCheckOneSidedPairRejected(t *testing.T) {
	inst := &Instance{
		Students: []Student{{ID: "s1", Preferences: []string{"p1"}}},
		Projects: []Project{{ID: "p1", Capacity: 1, Preferences: []string{}}},
	}
	rep := runCheck(t, inst, []CandidateAssignment{{Student: "s1", Project: "p1"}})
	if rep.Valid {
		t.Fatalf("expected acceptability error, got %+v", rep)
	}
}
