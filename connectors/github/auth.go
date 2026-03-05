package github

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultTokenFile = ".github-token"
	tokenEnvVar      = "GITHUB_TOKEN"
)

// ResolveToken finds a GitHub token using the following precedence:
//  1. $GITHUB_TOKEN environment variable
//  2. File at the given path (first line, trimmed)
//  3. File at the default path (.github-token)
//
// Returns empty string with no error if no token is found (public repos only).
func ResolveToken(tokenFilePath string) (string, error) {
	if tok := os.Getenv(tokenEnvVar); tok != "" {
		return strings.TrimSpace(tok), nil
	}

	paths := []string{tokenFilePath, defaultTokenFile}
	for _, p := range paths {
		if p == "" {
			continue
		}
		data, err := os.ReadFile(p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("read token file %s: %w", p, err)
		}
		line := strings.TrimSpace(strings.SplitN(string(data), "\n", 2)[0])
		if line != "" {
			return line, nil
		}
	}
	return "", nil
}

// cloneURL builds the HTTPS clone URL, optionally embedding the token
// for private repo access.
func cloneURL(org, repo, token string) string {
	if token != "" {
		return fmt.Sprintf("https://%s@github.com/%s/%s.git", token, org, repo)
	}
	return fmt.Sprintf("https://github.com/%s/%s.git", org, repo)
}
