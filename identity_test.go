package framework

import "testing"

func TestHomeZoneFor(t *testing.T) {
	cases := []struct {
		pos  Position
		want MetaPhase
	}{
		{PositionPG, MetaPhaseBk},
		{PositionSG, MetaPhasePt},
		{PositionPF, MetaPhaseFc},
		{PositionC, MetaPhaseFc},
	}
	for _, tc := range cases {
		got := HomeZoneFor(tc.pos)
		if got != tc.want {
			t.Errorf("HomeZoneFor(%s) = %q, want %q", tc.pos, got, tc.want)
		}
	}
}

func TestAgentIdentity_Tag(t *testing.T) {
	id := AgentIdentity{PersonaName: "Herald", Color: Color{Name: "Crimson"}}
	tag := id.Tag()
	if tag != "[crimson/herald]" {
		t.Errorf("Tag() = %q, want %q", tag, "[crimson/herald]")
	}
}

func TestAgentIdentity_Tag_ZeroValue(t *testing.T) {
	var id AgentIdentity
	tag := id.Tag()
	if tag != "[none/anon]" {
		t.Errorf("Tag() zero value = %q, want %q", tag, "[none/anon]")
	}
}
