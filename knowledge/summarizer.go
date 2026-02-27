package knowledge

// SummarizeStrategy controls how a source's content is presented within
// a token budget.
type SummarizeStrategy string

const (
	StrategyFull      SummarizeStrategy = "full"
	StrategySummary   SummarizeStrategy = "summary"
	StrategyOnDemand  SummarizeStrategy = "on_demand"
	StrategyIndexOnly SummarizeStrategy = "index_only"
)

// Summarizer reduces source content to fit within a token budget.
type Summarizer interface {
	Summarize(content string, budget int, strategy SummarizeStrategy) string
}

// TruncateSummarizer is a simple implementation that truncates content
// to fit within a character budget. For production use, replace with
// an LLM-backed summarizer that extracts key information.
type TruncateSummarizer struct{}

// Summarize truncates or transforms content based on strategy and budget.
// Budget is in approximate characters (1 token ~= 4 chars).
func (TruncateSummarizer) Summarize(content string, budget int, strategy SummarizeStrategy) string {
	charBudget := budget * 4
	if charBudget <= 0 {
		charBudget = len(content)
	}

	switch strategy {
	case StrategyFull:
		if len(content) <= charBudget {
			return content
		}
		return content[:charBudget] + "\n... [truncated to fit token budget]"

	case StrategySummary:
		if len(content) <= charBudget {
			return content
		}
		half := charBudget / 2
		if half < 100 {
			half = 100
		}
		if half > len(content) {
			return content
		}
		tail := len(content) - half
		if tail < half {
			tail = half
		}
		return content[:half] + "\n... [middle omitted] ...\n" + content[tail:]

	case StrategyOnDemand:
		return "[content available on demand — " + itoa(len(content)) + " chars]"

	case StrategyIndexOnly:
		lines := countLines(content)
		return "[" + itoa(lines) + " lines, " + itoa(len(content)) + " chars — index only]"

	default:
		return content
	}
}

// BudgetAllocator distributes a total token budget across sources
// based on their priority. Higher-priority sources get more tokens.
type BudgetAllocator struct {
	TotalBudget int
}

// BudgetEntry is one source's token allocation.
type BudgetEntry struct {
	SourceName string
	Budget     int
	Strategy   SummarizeStrategy
}

// Allocate distributes the total budget evenly across sources.
// Sources with ReadPolicy == ReadAlways get StrategyFull.
// Other sources get StrategySummary if content would exceed their share,
// or StrategyFull if it fits.
func (ba BudgetAllocator) Allocate(sources []Source) []BudgetEntry {
	if len(sources) == 0 {
		return nil
	}

	perSource := ba.TotalBudget / len(sources)
	if perSource < 100 {
		perSource = 100
	}

	entries := make([]BudgetEntry, len(sources))
	for i, src := range sources {
		strategy := StrategySummary
		if src.IsAlwaysRead() {
			strategy = StrategyFull
		}
		entries[i] = BudgetEntry{
			SourceName: src.Name,
			Budget:     perSource,
			Strategy:   strategy,
		}
	}
	return entries
}

func countLines(s string) int {
	n := 1
	for _, c := range s {
		if c == '\n' {
			n++
		}
	}
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
