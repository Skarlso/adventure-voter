package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// StoryIndex represents the minimal index file that just defines the start.
type StoryIndex struct {
	Start string `yaml:"start"`
}

// Story represents the entire adventure flow (built from chapters).
type Story struct {
	Flow  StoryFlow            `yaml:"flow"`
	Nodes map[string]StoryNode `yaml:"nodes"`
}

// StoryFlow defines the entry point.
type StoryFlow struct {
	Start string `yaml:"start"`
}

// StoryNode represents a node in the adventure flow.
type StoryNode struct {
	File     string `yaml:"file"`
	Type     string `yaml:"type"` // story, decision, game-over, terminal
	Terminal bool   `yaml:"terminal,omitempty"`
	Next     string `yaml:"next,omitempty"`
}

// StoryEngine manages the adventure state and navigation.
type StoryEngine struct {
	Story      *Story
	ContentDir string
	chapters   map[string]*Chapter // Cache parsed chapters
}

// NewStoryEngine creates a new story engine.
func NewStoryEngine(indexPath, contentDir string) (*StoryEngine, error) {
	content, err := os.ReadFile(filepath.Clean(indexPath))
	if err != nil {
		return nil, fmt.Errorf("failed to read index file: %w", err)
	}

	var index StoryIndex
	if err := yaml.Unmarshal(content, &index); err != nil {
		return nil, fmt.Errorf("failed to parse index YAML: %w", err)
	}

	story, err := buildStoryFromChapters(contentDir, index.Start)
	if err != nil {
		return nil, fmt.Errorf("failed to build story from chapters: %w", err)
	}

	return &StoryEngine{
		Story:      story,
		ContentDir: contentDir,
		chapters:   make(map[string]*Chapter),
	}, nil
}

// buildStoryFromChapters scans the content directory and builds the story graph.
func buildStoryFromChapters(contentDir, startNode string) (*Story, error) {
	nodes := make(map[string]StoryNode)

	files, err := filepath.Glob(filepath.Join(contentDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to scan content directory: %w", err)
	}

	for _, filePath := range files {
		chapter, err := ParseMarkdownFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
		}

		if chapter.Metadata.ID == "" {
			continue
		}

		relPath, err := filepath.Rel(contentDir, filePath)
		if err != nil {
			relPath = filepath.Base(filePath)
		}

		node := StoryNode{
			File:     relPath,
			Type:     chapter.Metadata.Type,
			Terminal: chapter.Metadata.Terminal || chapter.Metadata.Type == "terminal",
			Next:     chapter.Metadata.Next,
		}

		nodes[chapter.Metadata.ID] = node
	}

	if _, ok := nodes[startNode]; !ok {
		return nil, fmt.Errorf("start node '%s' not found in chapters", startNode)
	}

	return &Story{
		Flow:  StoryFlow{Start: startNode},
		Nodes: nodes,
	}, nil
}

// GetChapter retrieves and parses a chapter by node ID.
func (se *StoryEngine) GetChapter(nodeID string) (*Chapter, error) {
	if chapter, ok := se.chapters[nodeID]; ok {
		return chapter, nil
	}

	node, ok := se.Story.Nodes[nodeID]
	if !ok {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	filePath := filepath.Join(se.ContentDir, node.File)

	chapter, err := ParseMarkdownFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chapter %s: %w", nodeID, err)
	}

	if node.Type != "" {
		chapter.Metadata.Type = node.Type
	}

	if node.Terminal {
		chapter.Metadata.Terminal = node.Terminal
	}

	if node.Next != "" && chapter.Metadata.Next == "" {
		chapter.Metadata.Next = node.Next
	}

	se.chapters[nodeID] = chapter

	return chapter, nil
}

// GetStartChapter returns the first chapter.
func (se *StoryEngine) GetStartChapter() (*Chapter, error) {
	return se.GetChapter(se.Story.Flow.Start)
}

// GetNextChapter gets the next chapter based on current node.
func (se *StoryEngine) GetNextChapter(currentNodeID string) (*Chapter, error) {
	chapter, err := se.GetChapter(currentNodeID)
	if err != nil {
		return nil, err
	}

	if chapter.Metadata.Next == "" {
		return nil, fmt.Errorf("no next chapter defined for %s", currentNodeID)
	}

	return se.GetChapter(chapter.Metadata.Next)
}

// GetChapterByChoice gets the next chapter based on a choice ID.
func (se *StoryEngine) GetChapterByChoice(currentNodeID, choiceID string) (*Chapter, error) {
	chapter, err := se.GetChapter(currentNodeID)
	if err != nil {
		return nil, err
	}

	for _, choice := range chapter.Metadata.Choices {
		if choice.ID == choiceID {
			return se.GetChapter(choice.Next)
		}
	}

	return nil, fmt.Errorf("choice not found: %s", choiceID)
}

// ValidateStory checks if all nodes and files exist.
func (se *StoryEngine) ValidateStory() []error {
	var errors []error

	if _, ok := se.Story.Nodes[se.Story.Flow.Start]; !ok {
		errors = append(errors, fmt.Errorf("start node '%s' not found", se.Story.Flow.Start))
	}

	for nodeID, node := range se.Story.Nodes {
		filePath := filepath.Join(se.ContentDir, node.File)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("file not found for node '%s': %s", nodeID, filePath))

			continue
		}

		if _, err := se.GetChapter(nodeID); err != nil {
			errors = append(errors, fmt.Errorf("failed to parse node '%s': %w", nodeID, err))
		}
	}

	return errors
}
