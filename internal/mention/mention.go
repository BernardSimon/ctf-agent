package mention

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Mention struct {
	Raw   string
	Path  string
	Start int // 1-indexed, 0 = entire file
	End   int // 1-indexed, 0 = same as Start or entire file
}

// mentionRe matches @path, @path:N, or @path:N~M
var mentionRe = regexp.MustCompile(`@([^\s:]+)(?::(\d+)(?:~(\d+))?)?`)

// Parse extracts @-mentions, returns cleaned input and the mention list.
func Parse(input string) (string, []Mention) {
	var mentions []Mention
	cleaned := mentionRe.ReplaceAllStringFunc(input, func(raw string) string {
		m := parseSingle(raw)
		mentions = append(mentions, m)
		return ""
	})
	return strings.TrimSpace(cleaned), mentions
}

func parseSingle(raw string) Mention {
	sub := mentionRe.FindStringSubmatch(raw)
	if sub == nil {
		return Mention{Raw: raw}
	}
	m := Mention{Raw: raw, Path: sub[1]}
	if sub[2] != "" {
		m.Start, _ = strconv.Atoi(sub[2])
	}
	if sub[3] != "" {
		m.End, _ = strconv.Atoi(sub[3])
	}
	return m
}

// Content reads file content for the mention (respecting line range).
func (m Mention) Content() (string, int, error) {
	if m.Start == 0 {
		data, err := os.ReadFile(m.Path)
		if err != nil {
			return "", 0, fmt.Errorf("无法读取 %s: %w", m.Path, err)
		}
		lines := strings.Count(string(data), "\n")
		return string(data), lines, nil
	}

	f, err := os.Open(m.Path)
	if err != nil {
		return "", 0, fmt.Errorf("无法读取 %s: %w", m.Path, err)
	}
	defer f.Close()

	end := m.End
	if end == 0 {
		end = m.Start
	}

	scanner := bufio.NewScanner(f)
	var lines []string
	lineNum := 1
	for scanner.Scan() {
		if lineNum >= m.Start && lineNum <= end {
			lines = append(lines, scanner.Text())
		}
		if lineNum > end {
			break
		}
		lineNum++
	}
	return strings.Join(lines, "\n"), len(lines), nil
}

// BuildContext assembles the <file> blocks to prepend to the user message.
// Returns the context string and any errors from unresolvable mentions.
func BuildContext(mentions []Mention) (string, []error) {
	var sb strings.Builder
	var errs []error
	for _, m := range mentions {
		content, _, err := m.Content()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		sb.WriteString(fmt.Sprintf("<file path=%q", m.Path))
		if m.Start > 0 {
			if m.End > 0 && m.End != m.Start {
				sb.WriteString(fmt.Sprintf(" lines=%q", fmt.Sprintf("%d-%d", m.Start, m.End)))
			} else {
				sb.WriteString(fmt.Sprintf(" lines=%q", strconv.Itoa(m.Start)))
			}
		}
		sb.WriteString(">\n")
		sb.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			sb.WriteByte('\n')
		}
		sb.WriteString("</file>\n")
	}
	return sb.String(), errs
}
