package mention

import (
	"os"
	"path/filepath"
	"strings"
)

// FileCompleter implements readline.AutoCompleter for @file tokens.
type FileCompleter struct{}

// Do satisfies readline.AutoCompleter.
// offset = chars to replace before cursor; each candidate is the full replacement.
func (c *FileCompleter) Do(line []rune, pos int) ([][]rune, int) {
	// Scan back from cursor to find start of current whitespace-delimited token.
	start := pos
	for start > 0 && line[start-1] != ' ' && line[start-1] != '\t' {
		start--
	}
	word := string(line[start:pos])

	if !strings.HasPrefix(word, "@") {
		return nil, 0
	}
	partial := word[1:] // path portion after @

	// Once the user typed the colon for line spec, stop offering file completions.
	if strings.ContainsRune(partial, ':') {
		return nil, 0
	}

	matches := fileMatches(partial)
	if len(matches) == 0 {
		return nil, 0
	}

	out := make([][]rune, len(matches))
	for i, m := range matches {
		out[i] = []rune("@" + m)
	}
	return out, len(word)
}

// fileMatches returns file/dir names under the directory implied by partial.
func fileMatches(partial string) []string {
	dir, prefix := filepath.Split(partial)
	searchDir := dir
	if searchDir == "" {
		searchDir = "."
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return nil
	}

	var matches []string
	for _, e := range entries {
		name := e.Name()
		// Skip hidden files unless the user explicitly starts typing a dot.
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			continue
		}
		path := filepath.Join(dir, name)
		if dir == "" {
			path = name
		}
		if e.IsDir() {
			path += "/"
		}
		matches = append(matches, path)
	}
	return matches
}
