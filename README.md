# 🎮 Kubernetes Quest - Interactive Adventure Presentation

A "Choose Your Own Adventure" style interactive presentation system with real-time audience voting. Perfect for engaging tech talks, workshops, and demos!

## 🌟 Features

- **Real-time Voting**: Audience votes on what happens next using their phones
- **Beautiful UI**: Modern, responsive interfaces built with Alpine.js and Tailwind CSS
- **Markdown-Based Content**: Write your presentation in Markdown - no HTML required!
- **Branching Stories**: Create complex decision trees with multiple paths and outcomes
- **Live Results**: See votes update in real-time with WebSocket communication
- **Game Over States**: Fun "YOU'RE DEAD" endings for poor choices
- **Cookie-Based IDs**: Unique voter identification without signup
- **Mobile-Friendly**: Voting interface works perfectly on phones

## 🏗️ Architecture

```
┌─────────────────┐
│  Voter Phones   │ ← Mobile-friendly voting UI
│   (Alpine.js)   │
└────────┬────────┘
         │ WebSocket
         ↓
┌─────────────────┐
│   Go Backend    │ ← Vote aggregation & state
│  (WebSockets)   │
└────────┬────────┘
         │ WebSocket
         ↓
┌─────────────────┐
│ Presenter View  │ ← Main presentation display
│   (Alpine.js)   │
└─────────────────┘
```

## 🚀 Quick Start

### Option 1: Docker (Recommended)

```bash
# Build and run with Docker Compose
docker-compose up --build

# Access the application
# Presenter: http://localhost:8080/presenter/
# Voters: http://localhost:8080/voter/
```

### Option 2: Local Development

```bash
# Build the backend
cd voting
go build -o server ./backend/main.go

# Run the server
/path/to/voting/server

# Or with custom paths
/path/to/voting/server -addr=:8080 \
  -content=./content/chapters \
  -story=./content/story.yaml \
  -static=./frontend
```

## 📖 Creating Your Own Adventure

### 1. Story Structure

Your adventure consists of:
- **Markdown chapters** (`content/chapters/*.md`)
- **Story flow definition** (`content/story.yaml`)

### 2. Writing a Chapter

Create a markdown file with YAML frontmatter:

```markdown
---
id: my-chapter
type: story
---

# Chapter Title

Your story content goes here in **Markdown**!

You can use:
- Lists
- **Bold** and *italic*
- Code blocks
- And more!
```

#### Chapter Types

- `story` - Regular narrative chapter
- `decision` - Voting/choice point
- `game-over` - Dead end (presentation ends or restarts)
- `terminal` - Chapter with live terminal demo

### 3. Creating Decision Points

```markdown
---
id: important-choice
type: decision
timer: 60
choices:
  - id: option-a
    label: 🚀 Try the new technology
    description: "Bold but risky"
    icon: 🚀
    next: tech-path
    risk: high
  
  - id: option-b
    label: 🛡️ Use proven solution
    description: "Safe and boring"
    icon: 🛡️
    next: safe-path
    risk: low
---

# A Critical Decision

What should we do next?
```

### 4. Defining the Flow

Edit `content/story.yaml`:

```yaml
flow:
  start: intro  # Starting chapter ID

nodes:
  intro:
    file: chapters/01-intro.md
    type: story
    next: first-choice

  first-choice:
    file: chapters/02-choice.md
    type: decision
    # Next determined by voting

  tech-path:
    file: chapters/03a-tech.md
    type: story
    next: another-chapter

  safe-path:
    file: chapters/03b-safe.md
    type: story
    next: another-chapter
```

## 🎯 Presenting Your Adventure

### Setup

1. **Deploy to server** (Hetzner, DigitalOcean, etc.)
2. **Share the voter URL** with your audience (QR code works great!)
3. **Open presenter view** on your screen/projector

### During Presentation

1. Navigate through story chapters
2. When you hit a decision point, click "Start Voting"
3. Audience votes on their phones
4. Watch results update in real-time
5. After voting ends, click "Continue" to follow the winning path
6. Enjoy the journey!

### Pro Tips

- **Test beforehand**: Run through your adventure solo first
- **Have backup paths**: Plan for unexpected voting outcomes
- **Add humor**: "Game Over" screens are great for laughs
- **Time management**: Use shorter timers (30-45s) for quick engagement
- **QR codes**: Generate one for the voter URL and display it on slides
- **Announce IDs**: Let people know they can vote multiple times (last vote counts)

## 📁 Project Structure

```
voting/
├── backend/
│   ├── main.go              # Entry point
│   ├── parser/
│   │   ├── markdown.go      # Markdown parser
│   │   └── story.go         # Story flow engine
│   └── server/
│       ├── server.go        # HTTP/WebSocket server
│       └── votes.go         # Vote management
├── frontend/
│   ├── voter/
│   │   └── index.html       # Voting interface
│   └── presenter/
│       └── index.html       # Presentation view
├── content/
│   ├── chapters/            # Markdown chapters
│   │   ├── 01-intro.md
│   │   ├── 02-choice.md
│   │   └── ...
│   └── story.yaml           # Flow definition
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## 🛠️ Development

### Running Tests

```bash
go test ./...
```

### Building for Production

```bash
# Build binary
CGO_ENABLED=0 GOOS=linux go build -o server backend/main.go

# Or use Docker
docker build -t kube-quest .
docker run -p 8080:8080 kube-quest
```

### Hot Reload (Development)

Use `air` for hot reload during development:

```bash
go install github.com/cosmtrek/air@latest
air
```

## 🚢 Deploying to Hetzner

### Using Docker

```bash
# On your Hetzner server
git clone <your-repo>
cd voting
docker-compose up -d
```

### Using Systemd

```bash
# Build binary
go build -o server backend/main.go

# Create systemd service
sudo nano /etc/systemd/system/kube-quest.service
```

```ini
[Unit]
Description=Kubernetes Quest Interactive Presentation
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/kube-quest
ExecStart=/opt/kube-quest/server -addr=:8080
Restart=always

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start
sudo systemctl enable kube-quest
sudo systemctl start kube-quest
```

### Reverse Proxy (Nginx)

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 🎨 Customization

### Styling

Edit the Tailwind classes in the HTML files:
- `frontend/voter/index.html` - Voting interface
- `frontend/presenter/index.html` - Presentation view

### Voting Duration

Set timer in chapter frontmatter:

```yaml
timer: 60  # seconds
```

Or default to 60 seconds if not specified.

## 🐛 Troubleshooting

### WebSocket Connection Failed

- Check firewall rules (port 8080)
- Ensure WebSocket upgrade headers are passed through proxy
- Verify CORS settings in `server.go`

### Votes Not Updating

- Check browser console for errors
- Verify WebSocket connection in Network tab
- Restart the server

### Markdown Not Rendering

- Validate YAML frontmatter syntax
- Check file paths in `story.yaml`
- Look for errors in server logs

## 📝 Example Use Cases

- **Tech Talks**: Interactive demos where audience chooses the direction
- **Workshops**: Let students decide what to learn next
- **Team Meetings**: Make decisions collaboratively
- **Conferences**: Engaging alternative to static slides
- **Training**: Gamified learning experiences

## 🤝 Contributing

Ideas for future enhancements:
- [ ] Sound effects for voting/results
- [ ] Presenter dashboard with stats
- [ ] Export presentation recordings
- [ ] Integration with live terminal (xterm.js)
- [ ] Slack/Teams integration for remote audiences
- [ ] Analytics and voting history
- [ ] Multiple presentation sessions simultaneously

## 📄 License

MIT License - feel free to use this for your own presentations!

## 🙏 Acknowledgments

Built with:
- [Alpine.js](https://alpinejs.dev/) - Minimal JavaScript framework
- [Tailwind CSS](https://tailwindcss.com/) - Utility-first CSS
- [Goldmark](https://github.com/yuin/goldmark) - Markdown parser for Go
- [Gorilla WebSocket](https://github.com/gorilla/websocket) - WebSocket library
- [Gorilla Mux](https://github.com/gorilla/mux) - HTTP router

---

Made with ❤️ for interactive presentations. Now go create something awesome! 🚀
