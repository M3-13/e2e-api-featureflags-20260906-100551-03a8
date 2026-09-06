package store

import "testing"

func TestCreateAndGet(t *testing.T) {
	s := New()
	f := Flag{Key: "alpha", Enabled: true, Description: "desc", RolloutPercent: 50}
	if err := s.Create(f); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	got, ok := s.Get("alpha")
	if !ok {
		t.Fatal("Get returned ok=false for existing key")
	}
	if got != f {
		t.Fatalf("Get = %+v, want %+v", got, f)
	}
}

func TestCreateDuplicate(t *testing.T) {
	s := New()
	if err := s.Create(Flag{Key: "alpha"}); err != nil {
		t.Fatalf("first Create returned error: %v", err)
	}
	if err := s.Create(Flag{Key: "alpha"}); err != ErrExists {
		t.Fatalf("duplicate Create error = %v, want ErrExists", err)
	}
}

func TestGetMissing(t *testing.T) {
	s := New()
	if _, ok := s.Get("nope"); ok {
		t.Fatal("Get returned ok=true for missing key")
	}
}

func TestListEmptyIsNotNil(t *testing.T) {
	s := New()
	l := s.List()
	if l == nil {
		t.Fatal("List returned nil for empty store, want []")
	}
	if len(l) != 0 {
		t.Fatalf("List length = %d, want 0", len(l))
	}
}

func TestList(t *testing.T) {
	s := New()
	s.Create(Flag{Key: "a"})
	s.Create(Flag{Key: "b"})
	l := s.List()
	if len(l) != 2 {
		t.Fatalf("List length = %d, want 2", len(l))
	}
}

func TestUpdate(t *testing.T) {
	s := New()
	s.Create(Flag{Key: "a", Enabled: true})

	got, ok := s.Update("a", Flag{Key: "a", Enabled: false, Description: "x", RolloutPercent: 10})
	if !ok {
		t.Fatal("Update returned ok=false for existing key")
	}
	if got.Enabled || got.Description != "x" || got.RolloutPercent != 10 {
		t.Fatalf("Update result = %+v, want updated flag", got)
	}

	if _, ok := s.Update("missing", Flag{Key: "missing"}); ok {
		t.Fatal("Update returned ok=true for missing key")
	}
}

func TestDelete(t *testing.T) {
	s := New()
	s.Create(Flag{Key: "a"})

	if !s.Delete("a") {
		t.Fatal("Delete returned false for existing key")
	}
	if _, ok := s.Get("a"); ok {
		t.Fatal("Get returned ok=true after Delete")
	}
	if s.Delete("a") {
		t.Fatal("second Delete returned true for missing key")
	}
	if s.Delete("never-existed") {
		t.Fatal("Delete returned true for never-existing key")
	}
}
