#!/usr/bin/env fish

# Kubernetes Quest - Run Script for Fish Shell

echo "🔨 Building server..."
go build -o ./bin/server ./backend/main.go

if test $status -eq 0
    echo "✅ Build complete!"
    echo ""
    echo "🚀 Starting Kubernetes Quest..."
    echo "🎬 Presenter: http://localhost:8080/presenter/"
    echo "🎮 Voter: http://localhost:8080/voter/"
    echo ""
    
    # Run with absolute path
    set server_path (pwd)/bin/server
    eval $server_path
else
    echo "❌ Build failed!"
    exit 1
end
