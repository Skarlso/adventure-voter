package parser

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3"
)

// ChapterMetadata represents the YAML frontmatter in a markdown file.
type ChapterMetadata struct {
	ID       string   `yaml:"id"`
	Type     string   `yaml:"type"` // story, decision, game-over, terminal
	Timer    int      `yaml:"timer,omitempty"`
	Terminal bool     `yaml:"terminal,omitempty"`
	Next     string   `yaml:"next,omitempty"`
	Question string   `yaml:"question,omitempty"`
	Choices  []Choice `yaml:"choices,omitempty"`
}

// Choice represents a voting option.
type Choice struct {
	ID          string `yaml:"id"`
	Label       string `yaml:"label"`
	Description string `yaml:"description"`
	Next        string `yaml:"next"`
	Risk        string `yaml:"risk,omitempty"` // low, medium, high
	Icon        string `yaml:"icon,omitempty"`
}

// Chapter represents a parsed chapter with metadata and content.
type Chapter struct {
	Metadata ChapterMetadata
	Content  string
	RawMD    string
}

// ParseMarkdownFile reads and parses a markdown file with YAML frontmatter.
func ParseMarkdownFile(filePath string) (*Chapter, error) {
	content, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return ParseMarkdown(content)
}

// ParseMarkdown parses markdown content with YAML frontmatter.
func ParseMarkdown(content []byte) (*Chapter, error) {
	frontmatter, markdown, err := splitFrontmatter(content)
	if err != nil {
		return nil, err
	}

	var metadata ChapterMetadata
	if len(frontmatter) > 0 {
		err := yaml.Unmarshal(frontmatter, &metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
		}
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	var buf bytes.Buffer
	if err := md.Convert(markdown, &buf); err != nil {
		return nil, fmt.Errorf("failed to convert markdown: %w", err)
	}

	return &Chapter{
		Metadata: metadata,
		Content:  buf.String(),
		RawMD:    string(markdown),
	}, nil
}

// splitFrontmatter splits YAML frontmatter from markdown content
// Expected format:
// ---
// key: value
// ---
// # Markdown content.
func splitFrontmatter(content []byte) (frontmatter []byte, markdown []byte, err error) {
	if !bytes.HasPrefix(content, []byte("---\n")) && !bytes.HasPrefix(content, []byte("---\r\n")) {
		return nil, content, nil
	}

	start := 4 // skip first "---\n"
	if bytes.HasPrefix(content, []byte("---\r\n")) {
		start = 5
	}

	end := bytes.Index(content[start:], []byte("\n---\n"))
	if end == -1 {
		end = bytes.Index(content[start:], []byte("\n---\r\n"))
		if end == -1 {
			return nil, nil, errors.New("unclosed frontmatter")
		}
	}

	frontmatter = content[start : start+end]

	markdownStart := start + end + 5 // skip "\n---\n"
	if bytes.Contains(content[start+end:start+end+6], []byte("\r\n")) {
		markdownStart = start + end + 6
	}

	if markdownStart < len(content) {
		markdown = content[markdownStart:]
	}

	return frontmatter, markdown, nil
}
