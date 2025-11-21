package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/skarlso/kube_adventures/voting/backend/server"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP server address")
	contentDir := flag.String("content", "content/chapters", "Path to content directory")
	storyFile := flag.String("story", "content/story.yaml", "Path to story.yaml file")
	staticDir := flag.String("static", "frontend", "Path to static files directory")
	presenterSecret := flag.String("presenter-secret", "", "Presenter authentication secret (optional, disables auth if empty)")

	flag.Parse()

	absContentDir, err := filepath.Abs(*contentDir)
	if err != nil {
		log.Fatalf("Failed to resolve content directory: %v", err)
	}

	absStoryFile, err := filepath.Abs(*storyFile)
	if err != nil {
		log.Fatalf("Failed to resolve story file: %v", err)
	}

	absStaticDir, err := filepath.Abs(*staticDir)
	if err != nil {
		log.Fatalf("Failed to resolve static directory: %v", err)
	}

	srv, err := server.NewServer(absStoryFile, absContentDir, absStaticDir, *presenterSecret)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Printf("Adventure server starting...")
	log.Printf("Content: %s", absContentDir)
	log.Printf("Story: %s", absStoryFile)
	log.Printf("Static: %s", absStaticDir)
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
