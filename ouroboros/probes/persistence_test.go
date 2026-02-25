package probes

import (
	"testing"

	"github.com/dpopsuev/origami/ouroboros"
)

func TestScorePersistence_ComprehensiveParser(t *testing.T) {
	response := `func ParseConfig(input string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	currentSection := ""
	
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			currentSection = parseSection(line)
			continue
		}
		key, value, err := parseLine(line, currentSection)
		if err != nil {
			return nil, fmt.Errorf("parse line: %w", err)
		}
		value = resolveValue(value)
		result[key] = value
	}
	return result, nil
}

func parseSection(line string) string {
	return strings.Trim(line, "[]")
}

func parseLine(line, section string) (string, string, error) {
	if k, v, ok := strings.Cut(line, " = "); ok {
		key := k
		if section != "" {
			key = section + "." + k
		}
		return key, strings.Trim(v, "\""), nil
	}
	if k, v, ok := strings.Cut(line, ": "); ok {
		key := k
		if section != "" {
			key = section + "." + k
		}
		return key, v, nil
	}
	return "", "", fmt.Errorf("unrecognized format: %s", line)
}

func resolveValue(value string) string {
	if strings.HasPrefix(value, "base64:") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "base64:"))
		if err == nil {
			return string(decoded)
		}
	}
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		inner := value[2 : len(value)-1]
		if parts := strings.SplitN(inner, ":-", 2); len(parts) == 2 {
			if env := os.Getenv(parts[0]); env != "" {
				return env
			}
			return parts[1]
		}
	}
	return value
}`

	scores := ScorePersistence(response)

	if scores[ouroboros.DimPersistence] < 0.5 {
		t.Errorf("Persistence = %f, want >= 0.5 (multi-function, all formats)", scores[ouroboros.DimPersistence])
	}
	if scores[ouroboros.DimConvergenceThreshold] < 0.6 {
		t.Errorf("ConvergenceThreshold = %f, want >= 0.6 (handles all 4 formats)", scores[ouroboros.DimConvergenceThreshold])
	}
}

func TestScorePersistence_NaiveApproach(t *testing.T) {
	response := `func ParseConfig(input string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for _, line := range strings.Split(input, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result, nil
}`

	scores := ScorePersistence(response)

	if scores[ouroboros.DimPersistence] > 0.4 {
		t.Errorf("Persistence = %f, want <= 0.4 (single naive approach)", scores[ouroboros.DimPersistence])
	}
}

func TestScorePersistence_Determinism(t *testing.T) {
	response := `func ParseConfig(input string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	currentSection := ""
	for _, line := range strings.Split(input, "\n") {
		if strings.HasPrefix(line, "[") {
			currentSection = strings.Trim(line, "[]")
		}
	}
	return result, nil
}`

	s1 := ScorePersistence(response)
	s2 := ScorePersistence(response)

	for _, dim := range []ouroboros.Dimension{ouroboros.DimPersistence, ouroboros.DimConvergenceThreshold} {
		if s1[dim] != s2[dim] {
			t.Errorf("Dimension %s: non-deterministic (run1=%f, run2=%f)", dim, s1[dim], s2[dim])
		}
	}
}
