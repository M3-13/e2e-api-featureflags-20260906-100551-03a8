package hash

import "testing"

func TestEvaluateDeterministic(t *testing.T) {
	first := Evaluate("myflag", "alice", 50, true)
	for i := 0; i < 100; i++ {
		if got := Evaluate("myflag", "alice", 50, true); got != first {
			t.Fatalf("result changed across calls: %v then %v", first, got)
		}
	}
}

func TestEvaluateRolloutZeroNeverActive(t *testing.T) {
	for _, u := range []string{"a", "b", "c", "alice", "bob"} {
		if Evaluate("flag", u, 0, true) {
			t.Fatalf("rollout 0 evaluated active for user %q", u)
		}
	}
}

func TestEvaluateRolloutHundredAlwaysActive(t *testing.T) {
	for _, u := range []string{"a", "b", "c", "alice", "bob"} {
		if !Evaluate("flag", u, 100, true) {
			t.Fatalf("rollout 100 evaluated inactive for user %q", u)
		}
	}
}

func TestEvaluateDisabledAlwaysFalse(t *testing.T) {
	if Evaluate("flag", "alice", 100, false) {
		t.Fatal("disabled flag with rollout 100 evaluated active")
	}
	if Evaluate("flag", "alice", 0, false) {
		t.Fatal("disabled flag with rollout 0 evaluated active")
	}
}

func TestEvaluateStablePerUser(t *testing.T) {
	for _, u := range []string{"alice", "bob", "carol"} {
		first := Evaluate("flag", u, 50, true)
		for i := 0; i < 20; i++ {
			if Evaluate("flag", u, 50, true) != first {
				t.Fatalf("user %q result unstable", u)
			}
		}
	}
}
