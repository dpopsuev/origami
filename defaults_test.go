package framework

import "testing"

func TestDefaultWalker_Deterministic(t *testing.T) {
	w1 := DefaultWalker()
	w2 := DefaultWalker()

	id1 := w1.Identity()
	id2 := w2.Identity()

	if id1.PersonaName != id2.PersonaName {
		t.Fatalf("persona mismatch: %q vs %q", id1.PersonaName, id2.PersonaName)
	}
	if id1.Element != id2.Element {
		t.Fatalf("element mismatch: %q vs %q", id1.Element, id2.Element)
	}
}

func TestDefaultWalker_UsesEarthElement(t *testing.T) {
	w := DefaultWalker()
	if w.Identity().Element != ElementEarth {
		t.Fatalf("expected Earth element, got %q", w.Identity().Element)
	}
}

func TestDefaultWalker_UsesSentinelPersona(t *testing.T) {
	w := DefaultWalker()
	if w.Identity().PersonaName != "Sentinel" {
		t.Fatalf("expected Sentinel persona, got %q", w.Identity().PersonaName)
	}
}

func TestDefaultWalkerWithElement_OverridesElement(t *testing.T) {
	w := DefaultWalkerWithElement(ElementFire)
	id := w.Identity()
	if id.Element != ElementFire {
		t.Fatalf("expected Fire element, got %q", id.Element)
	}
	if id.PersonaName != "Sentinel" {
		t.Fatalf("persona should remain Sentinel, got %q", id.PersonaName)
	}
}

func TestDefaultWalker_HasValidState(t *testing.T) {
	w := DefaultWalker()
	s := w.State()
	if s == nil {
		t.Fatal("state should not be nil")
	}
	if s.ID != "default" {
		t.Fatalf("expected state ID 'default', got %q", s.ID)
	}
	if s.Status != "running" {
		t.Fatalf("expected status 'running', got %q", s.Status)
	}
}
