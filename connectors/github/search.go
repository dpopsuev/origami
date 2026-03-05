package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/dpopsuev/origami/schematics/rca"
)

const maxSearchResults = 50

// SearchCode runs ripgrep on the local clone and returns matching results.
func SearchCode(ctx context.Context, localPath string, keywords []string) ([]rca.SearchResult, error) {
	if len(keywords) == 0 {
		return nil, nil
	}

	pattern := strings.Join(keywords, "|")

	args := []string{
		"--json",
		"--max-count", "5",
		"--max-filesize", "1M",
		"--type-add", "code:*.{go,py,rs,js,ts,yaml,yml,json,sh,c,h,cpp,hpp}",
		"--type", "code",
		"-e", pattern,
		localPath,
	}

	cmd := exec.CommandContext(ctx, "rg", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("ripgrep search: %w", err)
	}

	return parseRipgrepJSON(output, localPath)
}

type rgMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type rgMatch struct {
	Path    rgPath      `json:"path"`
	Lines   rgText      `json:"lines"`
	LineNum int         `json:"line_number"`
	SubM    []rgSubmatch `json:"submatches"`
}

type rgPath struct {
	Text string `json:"text"`
}

type rgText struct {
	Text string `json:"text"`
}

type rgSubmatch struct {
	Match rgText `json:"match"`
}

func parseRipgrepJSON(data []byte, basePath string) ([]rca.SearchResult, error) {
	var results []rca.SearchResult

	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		var msg rgMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if msg.Type != "match" {
			continue
		}
		var m rgMatch
		if err := json.Unmarshal(msg.Data, &m); err != nil {
			continue
		}

		relPath := strings.TrimPrefix(m.Path.Text, basePath+"/")
		results = append(results, rca.SearchResult{
			File:    relPath,
			Line:    m.LineNum,
			Snippet: strings.TrimSpace(m.Lines.Text),
			Score:   float64(len(m.SubM)),
		})

		if len(results) >= maxSearchResults {
			break
		}
	}
	return results, nil
}
