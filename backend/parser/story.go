package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// StoryNode represents a node in the adventure flow.
type StoryNode struct {
	File     string `yaml:"file"`
	Type     string `yaml:"type"` // story, decision, game-over, terminal
	Terminal bool   `yaml:"terminal,omitempty"`
	Next     string `yaml:"next,omitempty"`
}

// Story represents the entire adventure flow.
type Story struct {
	Flow  StoryFlow            `yaml:"flow"`
	Nodes map[string]StoryNode `yaml:"nodes"`
}

// StoryFlow defines the entry point.
type StoryFlow struct {
	Start string `yaml:"start"`
}

// StoryEngine manages the adventure state and navigation.
type StoryEngine struct {
	Story      *Story
	ContentDir string
	chapters   map[string]*Chapter // Cache parsed chapters
}

// NewStoryEngine creates a new story engine.
func NewStoryEngine(storyPath, contentDir string) (*StoryEngine, error) {
	content, err := os.ReadFile(storyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read story file: %w", err)
	}

	var story Story
	if err := yaml.Unmarshal(content, &story); err != nil {
		return nil, fmt.Errorf("failed to parse story YAML: %w", err)
	}

	return &StoryEngine{
		Story:      &story,
		ContentDir: contentDir,
		chapters:   make(map[string]*Chapter),
	}, nil
}

// GetChapter retrieves and parses a chapter by node ID.
func (se *StoryEngine) GetChapter(nodeID string) (*Chapter, error) {
	// Check cache first
	if chapter, ok := se.chapters[nodeID]; ok {
		return chapter, nil
	}

	// Get node definition
	node, ok := se.Story.Nodes[nodeID]
	if !ok {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	// Parse chapter file
	filePath := filepath.Join(se.ContentDir, node.File)

	chapter, err := ParseMarkdownFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chapter %s: %w", nodeID, err)
	}

	// Override metadata from story.yaml if specified
	if node.Type != "" {
		chapter.Metadata.Type = node.Type
	}

	if node.Terminal {
		chapter.Metadata.Terminal = node.Terminal
	}

	if node.Next != "" && chapter.Metadata.Next == "" {
		chapter.Metadata.Next = node.Next
	}

	// Cache the chapter
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

	// Find the choice
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

	// Check if start node exists
	if _, ok := se.Story.Nodes[se.Story.Flow.Start]; !ok {
		errors = append(errors, fmt.Errorf("start node '%s' not found", se.Story.Flow.Start))
	}

	// Check all nodes
	for nodeID, node := range se.Story.Nodes {
		// Check if file exists
		filePath := filepath.Join(se.ContentDir, node.File)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("file not found for node '%s': %s", nodeID, filePath))

			continue
		}

		// Try to parse the chapter
		if _, err := se.GetChapter(nodeID); err != nil {
			errors = append(errors, fmt.Errorf("failed to parse node '%s': %w", nodeID, err))
		}
	}

	return errors
}
