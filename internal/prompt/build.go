package prompt

import (
	"fmt"
)

const MaxDiffChars = 70000

type Input struct {
	Branch   string
	DiffStat string
	Diff     string
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func BuildMessages(in Input) []Message {
	system := "You write concise git commit subjects. Use imperative mood. Output exactly one line. No quotes. No markdown."

	diff := in.Diff
	truncated := false
	if len(diff) > MaxDiffChars {
		diff = diff[:MaxDiffChars]
		truncated = true
	}

	user := fmt.Sprintf(
		"Branch: %s\n\nDiff stat:\n%s\n\nDiff:\n%s",
		in.Branch,
		in.DiffStat,
		diff,
	)
	if truncated {
		user += "\n\nNOTE: Diff was truncated due to context limits."
	}

	return []Message{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}
}
