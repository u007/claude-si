package okf

import (
	"reflect"
	"testing"
)

func TestParseFrontmatterScalars(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want map[string]string
	}{
		{"plain", "---\ntitle: Alpha\ndescription: First # note\n---\n", map[string]string{"title": "Alpha", "description": "First"}},
		{"single quoted", "---\ntitle: 'It''s alpha'\n---\n", map[string]string{"title": "It's alpha"}},
		{"double quoted", "---\ntitle: \"Alpha\\nBeta\"\n---\n", map[string]string{"title": "Alpha\nBeta"}},
		{"literal", "---\ndescription: |\n  Alpha\n  Beta\n---\n", map[string]string{"description": "Alpha\nBeta\n"}},
		{"folded", "---\ndescription: >\n  Alpha\n  Beta\n---\n", map[string]string{"description": "Alpha Beta\n"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseFrontmatter([]byte(tc.raw))
			if !ok || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, %v; want %#v", got, ok, tc.want)
			}
		})
	}
}

func TestParseFrontmatterEdgeCases(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want map[string]string
		ok   bool
	}{
		{
			name: "first nonblank line sets block indent",
			raw:  "---\ndescription: |\n    first\n  second\n---\n",
			want: map[string]string{"description": "first\ncond\n"},
			ok:   true,
		},
		{
			name: "folded block preserves extra indentation",
			raw:  "---\ndescription: >\n  Alpha\n    Indented\n  Beta\n---\n",
			want: map[string]string{"description": "Alpha\n  Indented\nBeta\n"},
			ok:   true,
		},
		{
			name: "key requires colon followed by space",
			raw:  "---\ntitle:value\n---\n",
			ok:   false,
		},
		{
			name: "comment-only block is not conforming",
			raw:  "---\n# comment\n---\n",
			ok:   false,
		},
		{
			name: "empty key value plus continuation keeps leading space",
			raw:  "---\ntitle:\n  Alpha\n---\n",
			want: map[string]string{"title": " Alpha"},
			ok:   true,
		},
		{
			name: "flow mapping is empty",
			raw:  "---\nmeta: {a: b}\n---\n",
			want: map[string]string{"meta": ""},
			ok:   true,
		},
		{
			name: "flow sequence is empty",
			raw:  "---\ntags: [a, b]\n---\n",
			want: map[string]string{"tags": ""},
			ok:   true,
		},
		{
			name: "block sequence follows Perl folding",
			raw:  "---\ndescription:\n  - a\n  - b\n---\n",
			want: map[string]string{"description": " - a - b"},
			ok:   true,
		},
		{
			name: "strip chomping removes final newline",
			raw:  "---\ndescription: |-\n  Alpha\n---\n",
			want: map[string]string{"description": "Alpha"},
			ok:   true,
		},
		{
			name: "value beginning with hash stays literal",
			raw:  "---\ntitle: # x\n---\n",
			want: map[string]string{"title": "# x"},
			ok:   true,
		},
		{
			name: "duplicate key keeps last value",
			raw:  "---\ntitle: First\ntitle: Second\n---\n",
			want: map[string]string{"title": "Second"},
			ok:   true,
		},
		{
			name: "unclosed frontmatter is not conforming",
			raw:  "---\ntitle: Alpha\n",
			ok:   false,
		},
		{
			name: "CRLF frontmatter parses",
			raw:  "---\r\ntitle: Alpha\r\n---\r\n",
			want: map[string]string{"title": "Alpha"},
			ok:   true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseFrontmatter([]byte(tc.raw))
			if ok != tc.ok || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, %v; want %#v, %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestCompileGlob(t *testing.T) {
	cases := []struct {
		pattern string
		match   []string
		noMatch []string
	}{
		{"*.md", []string{"a.md"}, []string{"a/b.md", "a.txt"}},
		{"?", []string{"a"}, []string{"", "ab", "a/b"}},
		{"draft[.md", []string{"draft[.md"}, []string{"draftx.md"}},
		{"draft\\[.md", []string{"draft[.md"}, []string{"draftx.md"}},
		{"[ab].md", []string{"a.md", "b.md"}, []string{"c.md"}},
		{"[!a].md", []string{"b.md"}, []string{"a.md"}},
	}
	for _, tc := range cases {
		t.Run(tc.pattern, func(t *testing.T) {
			re := compileGlob(tc.pattern)
			for _, value := range tc.match {
				if !re.MatchString(value) {
					t.Fatalf("%q must match %q", tc.pattern, value)
				}
			}
			for _, value := range tc.noMatch {
				if re.MatchString(value) {
					t.Fatalf("%q must not match %q", tc.pattern, value)
				}
			}
		})
	}
}

func TestOKFVersion(t *testing.T) {
	cases := []struct {
		name  string
		raw   string
		value string
		found bool
	}{
		{"plain", "---\nokf_version: 0.1\n---\n", "0.1", true},
		{"CRLF", "---\r\nokf_version: 0.2\r\n---\r\n", "0.2", true},
		{"quoted", "---\nokf_version: \" 0.3 \"\n---\n", "0.3", true},
		{"empty marker is present", "---\nokf_version:\n---\n", "", true},
		{"outside frontmatter is absent", "# okf_version: 0.1\n", "", false},
		{"unclosed frontmatter is absent", "---\nokf_version: 0.1\n", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value, found := okfVersion([]byte(tc.raw))
			if value != tc.value || found != tc.found {
				t.Fatalf("got %q, %v; want %q, %v", value, found, tc.value, tc.found)
			}
		})
	}
}
