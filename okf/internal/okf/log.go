package okf

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

var (
	logAction   = regexp.MustCompile(`^(Creation|Update|Deprecation|Deletion)$`)
	logBlank    = regexp.MustCompile(`^[\t\n\x0b\f\r ]*$`)
	logTitle    = "# Directory Update Log"
	logPathBase = "log.md"
)

func RenderLog(root, action, docPath, summary string, now time.Time) ([]byte, error) {
	if !logAction.MatchString(action) {
		return nil, errors.New("okf: action must be Creation, Update, Deprecation or Deletion")
	}
	docPath = strings.TrimLeft(docPath, "/")
	for segment := range strings.FieldsFuncSeq(docPath, func(char rune) bool {
		return char == '/' || char == '\\'
	}) {
		if segment == ".." {
			return nil, errors.New("okf: doc path must stay inside the bundle")
		}
	}
	if docPath == "index.md" || docPath == logPathBase {
		return nil, errors.New("okf: doc path must not be index.md or log.md")
	}
	if strings.HasPrefix(docPath, `\`) || filepath.IsAbs(docPath) || filepath.VolumeName(docPath) != "" {
		return nil, errors.New("okf: doc path must stay inside the bundle")
	}
	docPath = filepath.ToSlash(docPath)
	if strings.ContainsAny(summary, "\r\n") {
		return nil, errors.New("okf: summary must be one line")
	}
	if action != "Deletion" {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(docPath)))
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("okf: %s/%s does not exist", root, docPath)
		}
	}

	logPath := filepath.Join(root, logPathBase)
	lines, err := readLogLines(logPath)
	if err != nil {
		return nil, err
	}
	date := now.Format("2006-01-02")
	entry := fmt.Sprintf("* **%s**: %s ([%s](/%s))", action, summary, docPath, docPath)
	today := regexp.MustCompile(`^##[\t\n\x0b\f\r ]+` + regexp.QuoteMeta(date) + `[\t\n\x0b\f\r ]*$`)

	todayAt := -1
	for i, line := range lines {
		if today.MatchString(line) {
			todayAt = i
			break
		}
	}
	if todayAt >= 0 {
		at := todayAt + 1
		for at < len(lines) && logBlank.MatchString(lines[at]) {
			at++
		}
		lines = slices.Insert(lines, at, entry)
	} else {
		titleAt := -1
		for i, line := range lines {
			if strings.HasPrefix(line, logTitle) {
				titleAt = i
				break
			}
		}
		if titleAt < 0 {
			return nil, fmt.Errorf("okf: %s has no '# Directory Update Log' heading", logPath)
		}
		at := titleAt + 1
		for at < len(lines) && logBlank.MatchString(lines[at]) {
			at++
		}
		block := []string{"", "## " + date, "", entry}
		if at < len(lines) {
			block = append(block, "")
		}
		lines = slices.Replace(lines, titleAt+1, at, block...)
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

func readLogLines(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []string{logTitle}, nil
		}
		return nil, fmt.Errorf("okf: cannot read %s: %w", path, err)
	}
	lines := strings.Split(string(raw), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, nil
}
