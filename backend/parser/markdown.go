package parser

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"gopkg.in/yaml.v3"
)

// ChapterMetadata represents the YAML frontmatter in a markdown file.
type ChapterMetadata struct {
	ID       string   `json:"id"                 yaml:"id"`
	Type     string   `json:"type"               yaml:"type"` // story, decision, game-over, terminal
	Timer    int      `json:"timer,omitempty"    yaml:"timer,omitempty"`
	Terminal bool     `json:"terminal,omitempty" yaml:"terminal,omitempty"`
	Next     string   `json:"next,omitempty"     yaml:"next,omitempty"`
	Question string   `json:"question,omitempty" yaml:"question,omitempty"`
	Choices  []Choice `json:"choices,omitempty"  yaml:"choices,omitempty"`
}

// Choice represents a voting option.
type Choice struct {
	ID          string `json:"id"             yaml:"id"`
	Label       string `json:"label"          yaml:"label"`
	Description string `json:"description"    yaml:"description"`
	Next        string `json:"next"           yaml:"next"`
	Risk        string `json:"risk,omitempty" yaml:"risk,omitempty"` // low, medium, high
	Icon        string `json:"icon,omitempty" yaml:"icon,omitempty"`
}

// Chapter represents a parsed chapter with metadata and content.
type Chapter struct {
	Metadata ChapterMetadata
	Content  string
	RawMD    string
}

// MediaURLPrefix is the route the server exposes the chapter directory under.
const MediaURLPrefix = "/media/"

// mediaPathTransformer rewrites chapter-relative image paths to the /media/
// route. Images are stores next to the chapter files.
type mediaPathTransformer struct{}

// Transform walks the node and rewrites image paths to their actually served location.
func (mediaPathTransformer) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		image, ok := n.(*ast.Image)
		if !ok || !entering {
			return ast.WalkContinue, nil
		}

		if dest, rewrite := mediaPath(string(image.Destination)); rewrite {
			image.Destination = []byte(dest)
		}

		return ast.WalkContinue, nil
	})
}

// mediaPath prefixes a chapter-relative image path with MediaURLPrefix. It
// leaves external URLs, data URIs, already-absolute paths and anything
// reaching outside the chapter directory, unmodified.
func mediaPath(dest string) (string, bool) {
	if dest == "" || strings.HasPrefix(dest, "/") || strings.HasPrefix(dest, "#") {
		return "", false
	}

	// if it has a scheme, it's an external reference, skip it.
	if parsed, err := url.Parse(dest); err != nil || parsed.Scheme != "" {
		return "", false
	}

	trimmed := strings.TrimPrefix(dest, "./")

	// clean the path
	if cleaned := path.Clean(trimmed); cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}

	return MediaURLPrefix + trimmed, true
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
			parser.WithASTTransformers(util.Prioritized(mediaPathTransformer{}, 100)),
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

// splitFrontmatter splits YAML frontmatter from Markdown content
// Expected format:
// ---
// key: value
// ---.
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
