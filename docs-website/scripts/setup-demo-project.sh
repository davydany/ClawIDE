#!/usr/bin/env bash
#
# setup-demo-project.sh
#
# Creates a sample project at ~/projects/clawide-demo/ for screenshot capture.
# This provides realistic content when capturing ClawIDE screenshots for the docs site.

set -euo pipefail

DEMO_DIR="$HOME/projects/clawide-demo"

if [ -d "$DEMO_DIR" ]; then
  echo "Demo project already exists at $DEMO_DIR"
  echo "Remove it first if you want a fresh setup: rm -rf $DEMO_DIR"
  exit 0
fi

echo "Creating demo project at $DEMO_DIR ..."
mkdir -p "$DEMO_DIR"
cd "$DEMO_DIR"

git init

# README.md
cat > README.md << 'EOF'
# ClawIDE Demo Project

A sample Python project used for documentation screenshots.

## Getting Started

```bash
pip install -r requirements.txt
python main.py
```

## Running with Docker

```bash
docker compose up
```
EOF

# main.py
cat > main.py << 'PYEOF'
"""ClawIDE Demo – sample application."""

import os
from datetime import datetime


def greet(name: str) -> str:
    """Return a greeting message."""
    return f"Hello, {name}! Welcome to ClawIDE."


def get_uptime() -> str:
    """Return the current server uptime string."""
    return datetime.now().isoformat()


def main() -> None:
    """Entry point for the demo application."""
    app_name = os.getenv("APP_NAME", "ClawIDE Demo")
    print(f"Starting {app_name} ...")
    print(greet("Developer"))
    print(f"Server time: {get_uptime()}")


if __name__ == "__main__":
    main()
PYEOF

# requirements.txt
cat > requirements.txt << 'EOF'
flask==3.1.0
requests==2.32.3
python-dotenv==1.0.1
pytest==8.3.4
EOF

# docker-compose.yml
cat > docker-compose.yml << 'EOF'
services:
  app:
    build: .
    ports:
      - "8000:8000"
    volumes:
      - .:/app
    environment:
      - APP_NAME=ClawIDE Demo
      - DEBUG=true

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
EOF

# .env (sample)
cat > .env << 'EOF'
APP_NAME=ClawIDE Demo
DEBUG=true
SECRET_KEY=demo-secret-key
EOF

# Initial commit
git add -A
git commit -m "Initial commit – demo project for ClawIDE screenshots"

echo ""
echo "Demo project created at $DEMO_DIR"
echo "You can now start ClawIDE and open this project for screenshots."
