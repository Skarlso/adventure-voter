package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"

	"github.com/skarlso/kube_adventures/voting/backend/server"
)

// version is set at build time via -ldflags.
var version string

// Frontend embeds the frontend directory at compile time.
//
//go:embed frontend
var frontendFS embed.FS

func main() {
	addr := flag.String("addr", ":8080", "HTTP server address")
	contentDir := flag.String("content", "content/chapters", "Path to content directory")
	storyFile := flag.String("story", "content/story.yaml", "Path to story.yaml file")
	presenterSecret := flag.String("presenter-secret", "", "Presenter authentication secret (optional, disables auth if empty)")
	versionFlag := flag.Bool("version", false, "Print version and exit")

	flag.Parse()

	if *versionFlag {
		if version == "" {
			version = "0.0.0-dev"
		}

		fmt.Println(version) //nolint:forbidigo // version printing

		return
	}

	absContentDir, err := filepath.Abs(*contentDir)
	if err != nil {
		log.Fatalf("Failed to resolve content directory: %v", err)
	}

	absStoryFile, err := filepath.Abs(*storyFile)
	if err != nil {
		log.Fatalf("Failed to resolve story file: %v", err)
	}

	// frontend filesystem with "frontend" prefix stripped
	embeddedFS, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatalf("Failed to get embedded frontend: %v", err)
	}

	srv, err := server.NewServer(absStoryFile, absContentDir, embeddedFS, *presenterSecret)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Printf("Adventure server starting...")
	log.Printf("Content: %s", absContentDir)
	log.Printf("Story: %s", absStoryFile)
	log.Printf("Static: embedded")
	log.Printf("Server: http://localhost%s", *addr)
	log.Printf("Voter: http://localhost%s/voter", *addr)
	log.Printf("Presenter: http://localhost%s/presenter", *addr)

	if *presenterSecret != "" {
		log.Printf("Presenter authentication: ENABLED")
	} else {
		log.Printf("Presenter authentication: DISABLED")
	}

	if err := srv.Start(*addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
