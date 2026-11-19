package db

import "testing"

// StatusCounts must agree with Stats() and with counting ListByKind rows by
// state, which is what the /api/stats handler did before it was collapsed into
// a single aggregate.
func TestExplorationStatusCountsMatchesNodeStates(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer d.Close() //nolint:errcheck

	expID, err := d.CreateExploration("status counts", "goal")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Exec(`DELETE FROM explorations WHERE id=$1`, expID) //nolint:errcheck
	es := d.Exploration(expID)

	empty, err := es.StatusCounts()
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.ByKind) != 0 || empty.IntentsRunning != 0 || empty.GoalsMet != 0 {
		t.Fatalf("empty exploration: %+v", empty)
	}

	g1, err := es.AddGoal(map[string]any{"text": "a"}, "human")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := es.AddGoal(map[string]any{"text": "b"}, "human"); err != nil {
		t.Fatal(err)
	}
	if err := es.SetNodeState(g1, "met"); err != nil {
		t.Fatal(err)
	}
	var running []int64
	for i := 0; i < 3; i++ {
		id, err := es.AddIntent(map[string]any{"summary": "intent"}, i, nil, "planner")
		if err != nil {
			t.Fatal(err)
		}
		running = append(running, id)
	}
	for _, id := range running[:2] {
		if ok, err := es.ClaimIntent(id, "worker"); err != nil || !ok {
			t.Fatalf("claim %d: ok=%v err=%v", id, ok, err)
		}
	}

	got, err := es.StatusCounts()
	if err != nil {
		t.Fatal(err)
	}
	stats, err := es.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ByKind) != len(stats) {
		t.Fatalf("ByKind %v != Stats %v", got.ByKind, stats)
	}
	for k, v := range stats {
		if got.ByKind[k] != v {
			t.Fatalf("ByKind[%s]=%d, Stats=%d", k, got.ByKind[k], v)
		}
	}

	wantRunning, wantMet := 0, 0
	intents, _ := es.ListByKind(KindIntent, 100000)
	for _, n := range intents {
		if n.State == "running" {
			wantRunning++
		}
	}
	goals, _ := es.ListByKind(KindGoal, 10000)
	for _, n := range goals {
		if n.State == "met" {
			wantMet++
		}
	}
	if got.IntentsRunning != wantRunning || wantRunning != 2 {
		t.Fatalf("IntentsRunning=%d, ListByKind count=%d, want 2", got.IntentsRunning, wantRunning)
	}
	if got.GoalsMet != wantMet || wantMet != 1 {
		t.Fatalf("GoalsMet=%d, ListByKind count=%d, want 1", got.GoalsMet, wantMet)
	}
	if got.ByKind[KindGoal] != len(goals) {
		t.Fatalf("goal total %d, want %d", got.ByKind[KindGoal], len(goals))
	}
}
