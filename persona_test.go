package framework

import "testing"

func TestAllPersonas_Count(t *testing.T) {
	all := AllPersonas()
	if len(all) != 8 {
		t.Errorf("len(AllPersonas) = %d, want 8", len(all))
	}
}

func TestThesisPersonas_Count(t *testing.T) {
	thesis := ThesisPersonas()
	if len(thesis) != 4 {
		t.Errorf("len(ThesisPersonas) = %d, want 4", len(thesis))
	}
	for _, p := range thesis {
		if p.Identity.Alignment != AlignmentThesis {
			t.Errorf("persona %q has alignment %q, want thesis", p.Identity.PersonaName, p.Identity.Alignment)
		}
	}
}

func TestAntithesisPersonas_Count(t *testing.T) {
	antithesis := AntithesisPersonas()
	if len(antithesis) != 4 {
		t.Errorf("len(AntithesisPersonas) = %d, want 4", len(antithesis))
	}
	for _, p := range antithesis {
		if p.Identity.Alignment != AlignmentAntithesis {
			t.Errorf("persona %q has alignment %q, want antithesis", p.Identity.PersonaName, p.Identity.Alignment)
		}
	}
}

func TestPersonaByName_Herald(t *testing.T) {
	p, ok := PersonaByName("Herald")
	if !ok {
		t.Fatal("PersonaByName(Herald) not found")
	}
	if p.Identity.Color.Name != "Crimson" {
		t.Errorf("Herald color = %q, want Crimson", p.Identity.Color.Name)
	}
	if p.Identity.Element != ElementFire {
		t.Errorf("Herald element = %q, want fire", p.Identity.Element)
	}
	if p.Identity.Position != PositionPG {
		t.Errorf("Herald position = %q, want PG", p.Identity.Position)
	}
	if p.Identity.Alignment != AlignmentThesis {
		t.Errorf("Herald alignment = %q, want thesis", p.Identity.Alignment)
	}
}

func TestPersonaByName_CaseInsensitive(t *testing.T) {
	_, ok := PersonaByName("herald")
	if !ok {
		t.Error("PersonaByName should be case-insensitive")
	}
	_, ok = PersonaByName("CHALLENGER")
	if !ok {
		t.Error("PersonaByName should be case-insensitive")
	}
}

func TestPersonaByName_NotFound(t *testing.T) {
	_, ok := PersonaByName("nonexistent")
	if ok {
		t.Error("PersonaByName should return false for nonexistent name")
	}
}

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

func TestPersonas_UniqueNames(t *testing.T) {
	all := AllPersonas()
	seen := make(map[string]bool, len(all))
	for _, p := range all {
		name := p.Identity.PersonaName
		if seen[name] {
			t.Errorf("duplicate persona name: %s", name)
		}
		seen[name] = true
	}
}

func TestPersonas_UniqueColors(t *testing.T) {
	all := AllPersonas()
	seen := make(map[string]bool, len(all))
	for _, p := range all {
		hex := p.Identity.Color.Hex
		if seen[hex] {
			t.Errorf("duplicate color hex: %s (persona %s)", hex, p.Identity.PersonaName)
		}
		seen[hex] = true
	}
}

func TestPersonas_AllPositionsCovered(t *testing.T) {
	positions := map[Position]int{PositionPG: 0, PositionSG: 0, PositionPF: 0, PositionC: 0}
	for _, p := range AllPersonas() {
		positions[p.Identity.Position]++
	}
	for pos, count := range positions {
		if count != 2 {
			t.Errorf("position %s has %d personas, want 2 (1 thesis + 1 antithesis)", pos, count)
		}
	}
}

func TestPersonas_AllHaveStepAffinity(t *testing.T) {
	for _, p := range AllPersonas() {
		if len(p.Identity.StepAffinity) == 0 {
			t.Errorf("persona %s has no step affinity", p.Identity.PersonaName)
		}
	}
}

func TestPersonas_AllHavePromptPreamble(t *testing.T) {
	for _, p := range AllPersonas() {
		if p.Identity.PromptPreamble == "" {
			t.Errorf("persona %s has empty prompt preamble", p.Identity.PersonaName)
		}
	}
}

func TestPersonas_HomeZoneMatchesPosition(t *testing.T) {
	for _, p := range AllPersonas() {
		expected := HomeZoneFor(p.Identity.Position)
		if p.Identity.HomeZone != expected {
			t.Errorf("persona %s: HomeZone=%q but HomeZoneFor(%s)=%q",
				p.Identity.PersonaName, p.Identity.HomeZone, p.Identity.Position, expected)
		}
	}
}

func TestAgentIdentity_Tag(t *testing.T) {
	id := AgentIdentity{PersonaName: "Herald", Color: ColorCrimson}
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

func TestColorPalette_HexFormat(t *testing.T) {
	colors := []Color{
		ColorCrimson, ColorCerulean, ColorCobalt, ColorAmber,
		ColorScarlet, ColorSapphire, ColorObsidian, ColorIron,
	}
	for _, c := range colors {
		if len(c.Hex) != 7 || c.Hex[0] != '#' {
			t.Errorf("color %s has invalid hex: %q", c.Name, c.Hex)
		}
		if c.Family == "" {
			t.Errorf("color %s has empty family", c.Name)
		}
	}
}
