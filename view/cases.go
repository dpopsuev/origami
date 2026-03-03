package view

// CaseResult represents the outcome of a single calibration or analysis case.
type CaseResult struct {
	CaseID     string  `json:"case_id"`
	DefectType string  `json:"defect_type,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Summary    string  `json:"summary,omitempty"`
	Status     string  `json:"status"` // "pass", "fail", "skip", "pending"
}

// CaseResultSet is an ordered collection of case results.
type CaseResultSet struct {
	Cases []CaseResult `json:"cases"`
}

// NewCaseResultSet creates an empty case result set.
func NewCaseResultSet() *CaseResultSet {
	return &CaseResultSet{}
}

// Add appends a case result. If a case with the same ID exists, it is replaced.
func (s *CaseResultSet) Add(c CaseResult) {
	for i, existing := range s.Cases {
		if existing.CaseID == c.CaseID {
			s.Cases[i] = c
			return
		}
	}
	s.Cases = append(s.Cases, c)
}

// Len returns the number of cases.
func (s *CaseResultSet) Len() int {
	return len(s.Cases)
}

// ByID returns the case with the given ID, or nil.
func (s *CaseResultSet) ByID(id string) *CaseResult {
	for i := range s.Cases {
		if s.Cases[i].CaseID == id {
			return &s.Cases[i]
		}
	}
	return nil
}
