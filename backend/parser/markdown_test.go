package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMarkdown(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantID      string
		wantType    string
		wantTimer   int
		wantContent string
		wantErr     bool
	}{
		{
			name: "valid chapter with frontmatter",
			input: `---
id: test-chapter
type: story
timer: 60
---
# Test Chapter

This is a test chapter with **bold** text.`,
			wantID:      "test-chapter",
			wantType:    "story",
			wantTimer:   60,
			wantContent: "<h1 id=\"test-chapter\">Test Chapter</h1>\n<p>This is a test chapter with <strong>bold</strong> text.</p>\n",
			wantErr:     false,
		},
		{
			name: "chapter without frontmatter",
			input: `# Simple Chapter

Just markdown content.`,
			wantID:      "",
			wantType:    "",
			wantTimer:   0,
			wantContent: "<h1 id=\"simple-chapter\">Simple Chapter</h1>\n<p>Just markdown content.</p>\n",
			wantErr:     false,
		},
		{
			name: "decision chapter with choices",
			input: `---
id: decision-point
type: decision
timer: 45
choices:
  - id: choice-a
    label: Choice A
    description: First choice
    next: path-a
    risk: low
  - id: choice-b
    label: Choice B
    description: Second choice
    next: path-b
    risk: high
---
# Make a Decision

What will you do?`,
			wantID:    "decision-point",
			wantType:  "decision",
			wantTimer: 45,
			wantErr:   false,
		},
		{
			name: "unclosed frontmatter",
			input: `---
id: broken
type: story

# Missing closing delimiter`,
			wantErr: true,
		},
		{
			name: "invalid yaml",
			input: `---
id: broken
type: [invalid yaml structure
---
# Content`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chapter, err := ParseMarkdown([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if chapter.Metadata.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", chapter.Metadata.ID, tt.wantID)
			}

			if chapter.Metadata.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", chapter.Metadata.Type, tt.wantType)
			}

			if chapter.Metadata.Timer != tt.wantTimer {
				t.Errorf("Timer = %d, want %d", chapter.Metadata.Timer, tt.wantTimer)
			}

			if tt.wantContent != "" && chapter.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", chapter.Content, tt.wantContent)
			}

			// Verify choices for decision chapter
			if tt.name == "decision chapter with choices" {
				if len(chapter.Metadata.Choices) != 2 {
					t.Errorf("got %d choices, want 2", len(chapter.Metadata.Choices))
				}

				if len(chapter.Metadata.Choices) > 0 {
					choice := chapter.Metadata.Choices[0]
					if choice.ID != "choice-a" {
						t.Errorf("choice ID = %q, want %q", choice.ID, "choice-a")
					}
					if choice.Risk != "low" {
						t.Errorf("choice risk = %q, want %q", choice.Risk, "low")
					}
				}
			}
		})
	}
}

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		wantFrontmatter  string
		wantMarkdown     string
		wantErr          bool
	}{
		{
			name: "valid frontmatter with unix newlines",
			input: `---
key: value
---
# Content`,
			wantFrontmatter: "key: value",
			wantMarkdown:    "# Content",
			wantErr:         false,
		},
		{
			name: "no frontmatter",
			input: `# Just Content

No frontmatter here.`,
			wantFrontmatter: "",
			wantMarkdown: `# Just Content

No frontmatter here.`,
			wantErr: false,
		},
		{
			name: "unclosed frontmatter",
			input: `---
key: value
# Missing closing delimiter`,
			wantErr: true,
		},
		{
			name: "empty frontmatter",
			input: `---

---
# Content`,
			wantFrontmatter: "",
			wantMarkdown:    "# Content",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frontmatter, markdown, err := splitFrontmatter([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(frontmatter) != tt.wantFrontmatter {
				t.Errorf("frontmatter = %q, want %q", string(frontmatter), tt.wantFrontmatter)
			}

			if string(markdown) != tt.wantMarkdown {
				t.Errorf("markdown = %q, want %q", string(markdown), tt.wantMarkdown)
			}
		})
	}
}

func TestParseMarkdownFile(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-chapter.md")

	content := `---
id: file-test
type: story
---
# File Test

Content from file.`

	if err := os.WriteFile(testFile, []byte(content), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	chapter, err := ParseMarkdownFile(testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chapter.Metadata.ID != "file-test" {
		t.Errorf("ID = %q, want %q", chapter.Metadata.ID, "file-test")
	}

	if chapter.Metadata.Type != "story" {
		t.Errorf("Type = %q, want %q", chapter.Metadata.Type, "story")
	}

	if !strings.Contains(chapter.Content, "File Test") {
		t.Errorf("Content does not contain expected text: %q", chapter.Content)
	}
}

func TestParseMarkdownFile_NotFound(t *testing.T) {
	_, err := ParseMarkdownFile("/nonexistent/file.md")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestMarkdownFeatures(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		contains []string
	}{
		{
			name:     "strikethrough",
			markdown: "This is ~~strikethrough~~ text",
			contains: []string{"<del>strikethrough</del>"},
		},
		{
			name: "task list",
			markdown: `- [x] Completed task
- [ ] Incomplete task`,
			contains: []string{"checked", "checkbox"},
		},
		{
			name: "code block",
			markdown: "```go\nfunc main() {}\n```",
			contains: []string{"<code", "func main()"},
		},
		{
			name: "links",
			markdown: "[Link text](https://example.com)",
			contains: []string{"<a", "href", "example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chapter, err := ParseMarkdown([]byte(tt.markdown))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, substr := range tt.contains {
				if !strings.Contains(chapter.Content, substr) {
					t.Errorf("content does not contain %q: %q", substr, chapter.Content)
				}
			}
		})
	}
}

func TestParseMarkdown_ImagePaths(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     string
	}{
		{
			name:     "sibling image is rewritten to the media route",
			markdown: "![diagram](etcd.png)",
			want:     `src="/media/etcd.png"`,
		},
		{
			name:     "explicit dot-slash is normalised",
			markdown: "![diagram](./etcd.png)",
			want:     `src="/media/etcd.png"`,
		},
		{
			name:     "subdirectory is preserved",
			markdown: "![diagram](diagrams/etcd.png)",
			want:     `src="/media/diagrams/etcd.png"`,
		},
		{
			name:     "query string survives the rewrite",
			markdown: "![diagram](etcd.png?v=2)",
			want:     `src="/media/etcd.png?v=2"`,
		},
		{
			name:     "already absolute path is left alone",
			markdown: "![diagram](/media/etcd.png)",
			want:     `src="/media/etcd.png"`,
		},
		{
			name:     "external url is left alone",
			markdown: "![gopher](https://example.com/gopher.png)",
			want:     `src="https://example.com/gopher.png"`,
		},
		{
			name:     "data uri is left alone",
			markdown: "![dot](data:image/gif;base64,R0lGODlhAQABAAAAACw=)",
			want:     `src="data:image/gif;base64,R0lGODlhAQABAAAAACw="`,
		},
		{
			name:     "path escaping the chapter dir is left visibly broken",
			markdown: "![oops](../outside.png)",
			want:     `src="../outside.png"`,
		},
		{
			name:     "links are not touched",
			markdown: "[notes](notes.md)",
			want:     `href="notes.md"`,
		},
		{
			name:     "alt text is preserved",
			markdown: "![a goblin (pixel)](goblin.png)",
			want:     `alt="a goblin (pixel)"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chapter, err := ParseMarkdown([]byte(tt.markdown))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(chapter.Content, tt.want) {
				t.Errorf("expected %q in %q", tt.want, chapter.Content)
			}
		})
	}
}

func TestParseMarkdown_ImagePathsKeepRawMD(t *testing.T) {
	chapter, err := ParseMarkdown([]byte("![diagram](etcd.png)"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// the editor round-trips RawMD back to disk, so the rewrite must not leak
	// into the source the author wrote
	if !strings.Contains(chapter.RawMD, "(etcd.png)") {
		t.Errorf("expected raw markdown to keep the relative path, got %q", chapter.RawMD)
	}

	if strings.Contains(chapter.RawMD, "/media/") {
		t.Errorf("rewrite leaked into raw markdown: %q", chapter.RawMD)
	}
}
