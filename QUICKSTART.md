# Quick Start Guide

## 🚀 Get Running in 60 Seconds

### 1. Build and Run

```bash
cd voting

# Option 1: Use the fish script
./run.fish

# Option 2: Manual build and run
go build -o server ./backend/main.go
(pwd)/server

# Option 3: Use make (if installed)
make run
```

### 2. Open in Browser

- **Presenter View**: http://localhost:8080/presenter/
- **Voter View**: http://localhost:8080/voter/

### 3. Test It Out

1. Open the **Presenter View** on your main screen
2. Open the **Voter View** on your phone (or another browser tab)
3. Click through the intro
4. When you reach a decision point, click "Start Voting" in the presenter
5. Vote on your phone
6. Watch results update in real-time!
7. Click "Continue the Adventure" to see where the story goes

## 📱 Testing with Multiple Voters

Open the voter URL in multiple browser tabs or incognito windows:
```
http://localhost:8080/voter/
```

Each will get a unique voter ID automatically!

## 🎬 Presentation Flow

1. **Story chapters** - Click "Continue" to advance
2. **Decision points** - Click "Start Voting" to begin
3. **Voting** - Watch real-time results
4. **Results** - After timer ends, click "Continue the Adventure"
5. **Game Over** - Dead ends with fun messages!

## 🐛 Troubleshooting

### "Connection Failed"
- Make sure the server is running: `./server`
- Check http://localhost:8080 in your browser

### "No chapter displayed"
- Verify files exist in `content/chapters/`
- Check server logs for errors
- Validate `content/story.yaml` syntax

### "Votes not updating"
- Open browser console (F12)
- Check for WebSocket connection errors
- Ensure firewall allows port 8080

## 🎨 Customizing

### Change Voting Timer
Edit chapter frontmatter:
```yaml
---
timer: 30  # 30 seconds instead of default 60
---
```

### Add New Chapters
1. Create `.md` file in `content/chapters/`
2. Add entry to `content/story.yaml`
3. Link from other chapters via `next:` or `choices:`

### Styling
Edit HTML files:
- `frontend/voter/index.html` - Voting interface
- `frontend/presenter/index.html` - Presentation view

## 📦 Docker Deployment

```bash
# Build
docker build -t kube-quest .

# Run
docker run -p 8080:8080 kube-quest

# Or use docker-compose
docker-compose up
```

## 🎯 Next Steps

- Add more chapters to your adventure
- Create branching paths
- Add "Game Over" scenarios
- Deploy to a public server for your presentation
- Share the voter URL via QR code

## 💡 Tips for Presentations

1. **Test beforehand** - Run through your entire adventure
2. **QR codes** - Generate one for http://your-server.com/voter/
3. **Time limits** - Keep votes short (30-60 seconds)
4. **Backup plans** - Have paths for unexpected outcomes
5. **Engagement** - Add humor and drama!

Have fun! 🎮
