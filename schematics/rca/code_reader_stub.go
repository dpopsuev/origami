package rca

import "context"

// StubCodeReader returns empty results for all operations.
// Used in stub calibration where no code access is needed.
type StubCodeReader struct{}

var _ CodeReader = (*StubCodeReader)(nil)

func (s *StubCodeReader) EnsureCloned(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}

func (s *StubCodeReader) SearchCode(_ context.Context, _ string, _ []string) ([]SearchResult, error) {
	return nil, nil
}

func (s *StubCodeReader) ReadFile(_ context.Context, _, _ string) ([]byte, error) {
	return nil, nil
}

func (s *StubCodeReader) ListTree(_ context.Context, _ string, _ int) ([]TreeEntry, error) {
	return nil, nil
}

// MockCodeReader reads from a local testdata directory, simulating code access.
// Used in dry calibration with canned code fixtures.
type MockCodeReader struct {
	BaseDir string
}

var _ CodeReader = (*MockCodeReader)(nil)

func (m *MockCodeReader) EnsureCloned(_ context.Context, org, repo, _ string) (string, error) {
	return m.BaseDir + "/" + org + "/" + repo, nil
}

func (m *MockCodeReader) SearchCode(ctx context.Context, localPath string, keywords []string) ([]SearchResult, error) {
	return nil, nil
}

func (m *MockCodeReader) ReadFile(ctx context.Context, localPath, filePath string) ([]byte, error) {
	return nil, nil
}

func (m *MockCodeReader) ListTree(ctx context.Context, localPath string, maxDepth int) ([]TreeEntry, error) {
	return nil, nil
}
