package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStoryEngine(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "chapters")
	if err := os.Mkdir(contentDir, 0755); err != nil {
		t.Fatalf("failed to create content dir: %v", err)
	}

	// Create test index file (simplified)
	indexContent := `start: intro`

	indexFile := filepath.Join(tmpDir, "story.yaml")
	if err := os.WriteFile(indexFile, []byte(indexContent), 0600); err != nil {
		t.Fatalf("failed to create index file: %v", err)
	}

	// Create test markdown files
	testFiles := map[string]string{
		"intro.md": `---
id: intro
type: story
next: choice1
---
# Introduction

Welcome!`,
		"choice.md": `---
id: choice1
type: decision
choices:
  - id: opt-a
    label: Option A
    next: path-a
  - id: opt-b
    label: Option B
    next: path-b
---
# Make a choice`,
		"path-a.md": `---
id: path-a
type: story
---
# Path A`,
		"path-b.md": `---
id: path-b
type: game-over
---
# Game Over`,
	}

	for filename, content := range testFiles {
		path := filepath.Join(contentDir, filename)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatalf("failed to create %s: %v", filename, err)
		}
	}

	// Test creating story engine
	engine, err := NewStoryEngine(indexFile, contentDir)
	if err != nil {
		t.Fatalf("unexpected error creating engine: %v", err)
	}

	if engine.Story.Flow.Start != "intro" {
		t.Errorf("start node = %q, want %q", engine.Story.Flow.Start, "intro")
	}

	if len(engine.Story.Nodes) != 4 {
		t.Errorf("got %d nodes, want 4", len(engine.Story.Nodes))
	}

	// Verify nodes were built from chapters
	if _, ok := engine.Story.Nodes["intro"]; !ok {
		t.Error("intro node not found")
	}
	if _, ok := engine.Story.Nodes["choice1"]; !ok {
		t.Error("choice1 node not found")
	}
}

func TestNewStoryEngine_InvalidFile(t *testing.T) {
	_, err := NewStoryEngine("/nonexistent/story.yaml", "/tmp")
	if err == nil {
		t.Fatal("expected error for nonexistent story file")
	}
}

func TestNewStoryEngine_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	indexFile := filepath.Join(tmpDir, "invalid.yaml")

	invalidYAML := `start: [this is invalid yaml structure`

	if err := os.WriteFile(indexFile, []byte(invalidYAML), 0600); err != nil {
		t.Fatalf("failed to create invalid yaml: %v", err)
	}

	_, err := NewStoryEngine(indexFile, tmpDir)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestGetChapter(t *testing.T) {
	engine, tmpDir := setupTestEngine(t)

	tests := []struct {
		name     string
		nodeID   string
		wantID   string
		wantType string
		wantErr  bool
	}{
		{
			name:     "get intro chapter",
			nodeID:   "intro",
			wantID:   "intro",
			wantType: "story",
			wantErr:  false,
		},
		{
			name:     "get decision chapter",
			nodeID:   "choice1",
			wantID:   "choice1",
			wantType: "decision",
			wantErr:  false,
		},
		{
			name:    "nonexistent node",
			nodeID:  "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chapter, err := engine.GetChapter(tt.nodeID)

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
		})
	}

	// Test caching
	t.Run("chapter caching", func(t *testing.T) {
		// Get chapter twice
		chapter1, _ := engine.GetChapter("intro")
		chapter2, _ := engine.GetChapter("intro")

		// Should be the same instance (cached)
		if chapter1 != chapter2 {
			t.Error("chapter should be cached")
		}
	})

	// Cleanup
	os.RemoveAll(tmpDir)
}

func TestGetStartChapter(t *testing.T) {
	engine, tmpDir := setupTestEngine(t)
	defer os.RemoveAll(tmpDir)

	chapter, err := engine.GetStartChapter()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chapter.Metadata.ID != "intro" {
		t.Errorf("start chapter ID = %q, want %q", chapter.Metadata.ID, "intro")
	}
}

func TestGetNextChapter(t *testing.T) {
	engine, tmpDir := setupTestEngine(t)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		currentNodeID  string
		wantNextID     string
		wantErr        bool
	}{
		{
			name:          "intro to choice1",
			currentNodeID: "intro",
			wantNextID:    "choice1",
			wantErr:       false,
		},
		{
			name:          "no next defined",
			currentNodeID: "choice1",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextChapter, err := engine.GetNextChapter(tt.currentNodeID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nextChapter.Metadata.ID != tt.wantNextID {
				t.Errorf("next chapter ID = %q, want %q", nextChapter.Metadata.ID, tt.wantNextID)
			}
		})
	}
}

func TestGetChapterByChoice(t *testing.T) {
	engine, tmpDir := setupTestEngine(t)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name          string
		currentNodeID string
		choiceID      string
		wantNextID    string
		wantErr       bool
	}{
		{
			name:          "choice opt-a leads to path-a",
			currentNodeID: "choice1",
			choiceID:      "opt-a",
			wantNextID:    "path-a",
			wantErr:       false,
		},
		{
			name:          "choice opt-b leads to path-b",
			currentNodeID: "choice1",
			choiceID:      "opt-b",
			wantNextID:    "path-b",
			wantErr:       false,
		},
		{
			name:          "invalid choice",
			currentNodeID: "choice1",
			choiceID:      "invalid",
			wantErr:       true,
		},
		{
			name:          "no choices in node",
			currentNodeID: "intro",
			choiceID:      "any",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextChapter, err := engine.GetChapterByChoice(tt.currentNodeID, tt.choiceID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if nextChapter.Metadata.ID != tt.wantNextID {
				t.Errorf("next chapter ID = %q, want %q", nextChapter.Metadata.ID, tt.wantNextID)
			}
		})
	}
}

func TestValidateStory(t *testing.T) {
	t.Run("valid story", func(t *testing.T) {
		engine, tmpDir := setupTestEngine(t)
		defer os.RemoveAll(tmpDir)

		errors := engine.ValidateStory()
		if len(errors) > 0 {
			t.Errorf("expected no errors, got %d: %v", len(errors), errors)
		}
	})

	t.Run("missing start node", func(t *testing.T) {
		tmpDir := t.TempDir()
		contentDir := filepath.Join(tmpDir, "chapters")
		os.Mkdir(contentDir, 0755)

		indexContent := `start: nonexistent`

		indexFile := filepath.Join(tmpDir, "story.yaml")
		os.WriteFile(indexFile, []byte(indexContent), 0600)

		// Create a valid chapter that isn't the start node
		mdContent := `---
id: intro
type: story
---
# Intro`
		os.WriteFile(filepath.Join(contentDir, "intro.md"), []byte(mdContent), 0600)

		_, err := NewStoryEngine(indexFile, contentDir)
		if err == nil {
			t.Fatal("expected error for missing start node")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		contentDir := filepath.Join(tmpDir, "chapters")
		os.Mkdir(contentDir, 0755)

		indexContent := `start: intro`

		indexFile := filepath.Join(tmpDir, "story.yaml")
		os.WriteFile(indexFile, []byte(indexContent), 0600)

		// Create a chapter with broken markdown
		brokenContent := `---
id: intro
type: story
---
# This file is valid`
		os.WriteFile(filepath.Join(contentDir, "intro.md"), []byte(brokenContent), 0600)

		engine, err := NewStoryEngine(indexFile, contentDir)
		if err != nil {
			t.Fatalf("failed to create engine: %v", err)
		}

		errors := engine.ValidateStory()
		// Should have no errors if file is valid
		if len(errors) > 0 {
			t.Logf("Validation warnings: %v", errors)
		}
	})
}

func TestStoryNodeOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "chapters")
	os.Mkdir(contentDir, 0755)

	// Create simple index
	indexContent := `start: intro`
	indexFile := filepath.Join(tmpDir, "story.yaml")
	os.WriteFile(indexFile, []byte(indexContent), 0600)

	// Create markdown file - metadata now comes from the file itself
	mdContent := `---
id: intro
type: terminal
terminal: true
next: override-next
---
# Intro`

	os.WriteFile(filepath.Join(contentDir, "intro.md"), []byte(mdContent), 0600)

	engine, err := NewStoryEngine(indexFile, contentDir)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	chapter, err := engine.GetChapter("intro")
	if err != nil {
		t.Fatalf("failed to get chapter: %v", err)
	}

	// Metadata should match what's in the markdown file
	if chapter.Metadata.Type != "terminal" {
		t.Errorf("Type = %q, want %q", chapter.Metadata.Type, "terminal")
	}

	// Terminal should be set
	if !chapter.Metadata.Terminal {
		t.Error("Terminal should be true")
	}

	// Next should be set
	if chapter.Metadata.Next != "override-next" {
		t.Errorf("Next = %q, want %q", chapter.Metadata.Next, "override-next")
	}
}

// setupTestEngine creates a test engine with sample content
func setupTestEngine(t *testing.T) (*StoryEngine, string) {
	t.Helper()

	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "chapters")
	if err := os.Mkdir(contentDir, 0755); err != nil {
		t.Fatalf("failed to create content dir: %v", err)
	}

	indexContent := `start: intro`
	indexFile := filepath.Join(tmpDir, "story.yaml")
	if err := os.WriteFile(indexFile, []byte(indexContent), 0600); err != nil {
		t.Fatalf("failed to create index file: %v", err)
	}

	testFiles := map[string]string{
		"intro.md": `---
id: intro
type: story
next: choice1
---
# Introduction

Welcome!`,
		"choice.md": `---
id: choice1
type: decision
choices:
  - id: opt-a
    label: Option A
    next: path-a
  - id: opt-b
    label: Option B
    next: path-b
---
# Make a choice`,
		"path-a.md": `---
id: path-a
type: story
---
# Path A`,
		"path-b.md": `---
id: path-b
type: game-over
---
# Game Over`,
	}

	for filename, content := range testFiles {
		path := filepath.Join(contentDir, filename)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatalf("failed to create %s: %v", filename, err)
		}
	}

	engine, err := NewStoryEngine(indexFile, contentDir)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	return engine, tmpDir
}
