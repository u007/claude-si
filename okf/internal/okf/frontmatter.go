package okf

import (
	"regexp"
	"strings"
)

var (
	frontmatterClose = regexp.MustCompile(`(?s)^(.*?)\n---\r?(\n|$)`)
	frontmatterKey   = regexp.MustCompile(`^([^\t\n\x0b\f\r #][^:]*):[\t\n\x0b\f\r ]+(.*?)[\t\n\x0b\f\r ]*$`)
	frontmatterBare  = regexp.MustCompile(`^([^\t\n\x0b\f\r #][^:]*):$`)
	frontmatterBlank = regexp.MustCompile(`^[\t\n\x0b\f\r ]*(#.*)?$`)
	blockScalarStyle = regexp.MustCompile(`^([|>])([+-]?)[0-9]*[\t\n\x0b\f\r ]*$`)
	singleQuoted     = regexp.MustCompile(`(?s)^'(.*)'[\t\n\x0b\f\r ]*(#.*)?$`)
	doubleQuoted     = regexp.MustCompile(`(?s)^"(.*)"[\t\n\x0b\f\r ]*(#.*)?$`)
	plainComment     = regexp.MustCompile(`[\t\n\x0b\f\r ]+#.*$`)
)

func parseFrontmatter(raw []byte) (map[string]string, bool) {
	block, ok := frontmatterBlock(raw)
	if !ok {
		return nil, false
	}
	block = strings.ReplaceAll(block, "\r", "")

	fields := make(map[string]string)
	key := ""
	keySet := false
	buf := make([]string, 0, 1)
	flush := func() {
		if keySet {
			fields[key] = scalarValue(buf[0], buf[1:])
		}
		buf = buf[:0]
	}

	for line := range strings.SplitSeq(block, "\n") {
		if match := frontmatterKey.FindStringSubmatch(line); match != nil {
			flush()
			key = match[1]
			keySet = true
			buf = append(buf, match[2])
			continue
		}
		if match := frontmatterBare.FindStringSubmatch(line); match != nil {
			flush()
			key = match[1]
			keySet = true
			buf = append(buf, "")
			continue
		}
		if frontmatterBlank.MatchString(line) && !keySet {
			continue
		}
		if !keySet {
			return nil, false
		}
		buf = append(buf, line)
	}
	flush()
	if len(fields) == 0 {
		return nil, false
	}
	return fields, true
}

func frontmatterBlock(raw []byte) (string, bool) {
	text := string(raw)
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return "", false
	}
	rest := text[strings.IndexByte(text, '\n')+1:]
	if strings.HasPrefix(rest, "---") {
		return "", true
	}
	match := frontmatterClose.FindStringSubmatch(rest)
	if match == nil {
		return "", false
	}
	return match[1], true
}

func okfVersion(raw []byte) (string, bool) {
	block, ok := frontmatterBlock(raw)
	if !ok {
		return "", false
	}
	for line := range strings.SplitSeq(block, "\n") {
		line = trimPerlSpace(line)
		if line == "---" || line == "..." {
			break
		}
		value, found := strings.CutPrefix(line, "okf_version:")
		if !found {
			continue
		}
		return strings.Trim(trimPerlSpace(value), `"' `), true
	}
	return "", false
}

func scalarValue(first string, more []string) string {
	if match := blockScalarStyle.FindStringSubmatch(first); match != nil {
		style := match[1]
		chomp := match[2]
		indent := 0
		for _, line := range more {
			if hasNonSpace(line) {
				indent = leadingPerlSpace(line)
				break
			}
		}
		lines := make([]string, len(more))
		for i, line := range more {
			if len(line) >= indent {
				lines[i] = line[indent:]
			}
		}
		for len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		var value string
		if style == "|" {
			value = strings.Join(lines, "\n")
		} else {
			value = foldBlock(lines)
		}
		if chomp != "-" {
			value += "\n"
		}
		return value
	}

	lines := make([]string, 0, len(more)+1)
	lines = append(lines, first)
	lines = append(lines, more...)
	text := foldScalar(lines)
	if strings.HasPrefix(text, "[") || strings.HasPrefix(text, "{") {
		return ""
	}
	if text == "" && len(more) > 0 && startsWithPerlSpaceDash(more[0]) {
		return ""
	}
	if match := singleQuoted.FindStringSubmatch(text); match != nil {
		return strings.ReplaceAll(match[1], "''", "'")
	}
	if match := doubleQuoted.FindStringSubmatch(text); match != nil {
		return decodeDoubleQuoted(match[1])
	}
	return plainComment.ReplaceAllString(text, "")
}

func foldScalar(raw []string) string {
	lines := make([]string, len(raw))
	for i, line := range raw {
		lines[i] = trimPerlSpace(line)
	}
	for len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString(lines[0])
	blank := 0
	for _, line := range lines[1:] {
		if line == "" {
			blank++
			continue
		}
		if blank > 0 {
			out.WriteString(strings.Repeat("\n", blank))
		} else {
			out.WriteByte(' ')
		}
		out.WriteString(line)
		blank = 0
	}
	return out.String()
}

func foldBlock(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString(lines[0])
	endsNewline := strings.HasSuffix(lines[0], "\n")
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			out.WriteByte('\n')
			endsNewline = true
			continue
		}
		previous := lines[i-1]
		if previous == "" || startsWithPerlSpace(line) || startsWithPerlSpace(previous) {
			if !endsNewline {
				out.WriteByte('\n')
			}
		} else {
			out.WriteByte(' ')
		}
		out.WriteString(line)
		endsNewline = strings.HasSuffix(line, "\n")
	}
	return out.String()
}

func decodeDoubleQuoted(value string) string {
	var out strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' || i+1 >= len(value) {
			out.WriteByte(value[i])
			continue
		}
		i++
		switch value[i] {
		case 'n':
			out.WriteByte('\n')
		case 't':
			out.WriteByte('\t')
		case 'r':
			out.WriteByte('\r')
		case '"':
			out.WriteByte('"')
		case '\\':
			out.WriteByte('\\')
		case '/':
			out.WriteByte('/')
		case '0':
			out.WriteByte(0)
		case ' ':
			out.WriteByte(' ')
		default:
			out.WriteByte('\\')
			out.WriteByte(value[i])
		}
	}
	return out.String()
}

func trimPerlSpace(value string) string {
	return strings.Trim(value, "\t\n\x0b\f\r ")
}

func hasNonSpace(value string) bool {
	return strings.Trim(value, "\t\n\x0b\f\r ") != ""
}

func leadingPerlSpace(value string) int {
	return len(value) - len(strings.TrimLeft(value, "\t\n\x0b\f\r "))
}

func startsWithPerlSpaceDash(value string) bool {
	trimmed := strings.TrimLeft(value, "\t\n\x0b\f\r ")
	return len(trimmed) < len(value) && strings.HasPrefix(trimmed, "-")
}

func startsWithPerlSpace(value string) bool {
	return len(value) > 0 && strings.TrimLeft(value, "\t\n\x0b\f\r ") != value
}
