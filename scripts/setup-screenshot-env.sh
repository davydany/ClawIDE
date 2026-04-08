#!/usr/bin/env bash
#
# setup-screenshot-env.sh
#
# Populates a fresh ClawIDE instance with demo data for screenshot capture.
# Requires: jq, curl, git
# Assumes: ClawIDE is running at localhost:9800 with --projects-dir ~/projects/workspaces

set -euo pipefail

BASE_URL="${CLAWIDE_URL:-http://localhost:9800}"
WORKSPACE_DIR="${CLAWIDE_WORKSPACE_DIR:-$HOME/projects/workspaces}"
STATE_FILE="$HOME/.clawide/state.json"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m'

step() { echo -e "${GREEN}[Step $1]${NC} $2"; }
warn() { echo -e "${YELLOW}[Warning]${NC} $1"; }
fail() { echo -e "${RED}[Error]${NC} $1"; exit 1; }

# Verify prerequisites
command -v jq >/dev/null 2>&1 || fail "jq is required. Install with: brew install jq"
command -v curl >/dev/null 2>&1 || fail "curl is required"
curl -sf "$BASE_URL/" > /dev/null 2>&1 || fail "ClawIDE is not running at $BASE_URL"

# ============================================================
# Step 1: Complete onboarding (so dashboard shows, not welcome)
# ============================================================
step 1 "Completing onboarding..."
curl -sf -X POST "$BASE_URL/api/onboarding/complete" -o /dev/null || warn "Onboarding complete failed (may already be done)"
curl -sf -X POST "$BASE_URL/api/onboarding/workspace-tour-complete" -o /dev/null || warn "Tour complete failed (may already be done)"

# ============================================================
# Step 2: Import projects from workspaces directory
# ============================================================
step 2 "Importing projects..."

declare -A PROJECT_NAMES=(
  ["python-api"]="Python API"
  ["golang-microservice"]="Go Microservice"
  ["nodejs-express"]="Node.js Express"
  ["java-spring"]="Java Spring"
  ["rustapi"]="Rust API"
)

for dir_name in python-api golang-microservice nodejs-express java-spring rustapi; do
  project_path="$WORKSPACE_DIR/$dir_name"
  if [ ! -d "$project_path" ]; then
    warn "Skipping $dir_name — directory not found at $project_path"
    continue
  fi
  nice_name="${PROJECT_NAMES[$dir_name]}"
  curl -sf -X POST "$BASE_URL/projects/" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "name=$(printf '%s' "$nice_name" | sed 's/ /+/g')&path=$project_path" \
    -L -o /dev/null || warn "Failed to import $dir_name"
  echo "  Imported: $nice_name ($project_path)"
done

# Wait for state to settle
sleep 1

# ============================================================
# Step 3: Read project IDs from state.json
# ============================================================
step 3 "Reading project IDs..."

if [ ! -f "$STATE_FILE" ]; then
  fail "state.json not found at $STATE_FILE — did project import fail?"
fi

PYTHON_ID=$(jq -r '.projects[] | select(.name=="Python API") | .id' "$STATE_FILE" 2>/dev/null || echo "")
GOLANG_ID=$(jq -r '.projects[] | select(.name=="Go Microservice") | .id' "$STATE_FILE" 2>/dev/null || echo "")
NODEJS_ID=$(jq -r '.projects[] | select(.name=="Node.js Express") | .id' "$STATE_FILE" 2>/dev/null || echo "")
JAVA_ID=$(jq -r '.projects[] | select(.name=="Java Spring") | .id' "$STATE_FILE" 2>/dev/null || echo "")
RUST_ID=$(jq -r '.projects[] | select(.name=="Rust API") | .id' "$STATE_FILE" 2>/dev/null || echo "")

[ -n "$PYTHON_ID" ] || fail "Could not find Python API project ID"
[ -n "$GOLANG_ID" ] || fail "Could not find Go Microservice project ID"

echo "  Python API:       $PYTHON_ID"
echo "  Go Microservice:  $GOLANG_ID"
echo "  Node.js Express:  $NODEJS_ID"
echo "  Java Spring:      $JAVA_ID"
echo "  Rust API:         $RUST_ID"

# ============================================================
# Step 4: Star projects
# ============================================================
step 4 "Starring projects..."

for pid in "$PYTHON_ID" "$GOLANG_ID" "$NODEJS_ID"; do
  curl -sf -X PATCH "$BASE_URL/projects/$pid/star" -o /dev/null || warn "Failed to star $pid"
done
echo "  Starred: Python API, Go Microservice, Node.js Express"

# ============================================================
# Step 5: Set project colors
# ============================================================
step 5 "Setting project colors..."

declare -A PROJECT_COLORS=(
  ["$PYTHON_ID"]="#3B82F6"
  ["$GOLANG_ID"]="#10B981"
  ["$NODEJS_ID"]="#F59E0B"
  ["$JAVA_ID"]="#EF4444"
  ["$RUST_ID"]="#F97316"
)

for pid in "${!PROJECT_COLORS[@]}"; do
  color="${PROJECT_COLORS[$pid]}"
  curl -sf -X PATCH "$BASE_URL/projects/$pid/color" \
    -H "Content-Type: application/json" \
    -d "{\"color\":\"$color\"}" -o /dev/null || warn "Failed to set color for $pid"
done
echo "  Colors set for all projects"

# ============================================================
# Step 6: Create terminal sessions
# ============================================================
step 6 "Creating terminal sessions..."

# Python API — two sessions
curl -sf -X POST "$BASE_URL/projects/$PYTHON_ID/sessions/" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "name=Development" -L -o /dev/null || warn "Failed to create Development session"

curl -sf -X POST "$BASE_URL/projects/$PYTHON_ID/sessions/" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "name=Testing" -L -o /dev/null || warn "Failed to create Testing session"

# Go Microservice — one session
curl -sf -X POST "$BASE_URL/projects/$GOLANG_ID/sessions/" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "name=Debug+Session" -L -o /dev/null || warn "Failed to create Debug session"

echo "  Created: Development, Testing (Python), Debug Session (Go)"

# ============================================================
# Step 7: Git init python-api for feature workspace support
# ============================================================
step 7 "Initializing git in python-api..."

PYTHON_PATH="$WORKSPACE_DIR/python-api"
if [ ! -d "$PYTHON_PATH/.git" ]; then
  cd "$PYTHON_PATH"
  git init -b main
  git add -A
  git -c user.email="demo@clawide.dev" -c user.name="ClawIDE Demo" commit -m "Initial commit"
  cd - > /dev/null
  echo "  Git initialized in python-api"
else
  echo "  Git already initialized in python-api"
fi

# ============================================================
# Step 8: Create feature workspace
# ============================================================
step 8 "Creating feature workspace..."

curl -sf -X POST "$BASE_URL/projects/$PYTHON_ID/features/" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "name=add-authentication&base_branch=main" \
  -L -o /dev/null || warn "Failed to create feature workspace"
echo "  Created: add-authentication (Python API)"

# ============================================================
# Step 9: Set scratchpad content
# ============================================================
step 9 "Setting scratchpad content..."

SCRATCHPAD_CONTENT='## Sprint 14 Notes\n\n- [ ] Finish auth middleware for Python API\n- [ ] Add rate limiting to Go microservice\n- [x] Set up CI\/CD pipeline\n- [x] Configure Docker Compose for all services\n- [ ] Write integration tests for REST endpoints\n\n### Architecture Decisions\n- JWT for stateless auth (refresh tokens stored in Redis)\n- Rate limit: 100 req\/min per API key\n- Pagination: cursor-based for large datasets\n\n### Quick Links\n- Staging: https:\/\/staging.example.com\n- Grafana: https:\/\/monitoring.example.com\/d\/api'

curl -sf -X PUT "$BASE_URL/api/scratchpad" \
  -H "Content-Type: application/json" \
  -d "{\"content\":\"$SCRATCHPAD_CONTENT\"}" -o /dev/null || warn "Failed to set scratchpad"
echo "  Scratchpad populated with sprint notes"

# ============================================================
# Step 10: Create bookmarks
# ============================================================
step 10 "Creating bookmarks..."

curl -sf -X POST "$BASE_URL/api/bookmarks/" \
  -H "Content-Type: application/json" \
  -d "{\"project_id\":\"$PYTHON_ID\",\"name\":\"Django Docs\",\"url\":\"https://docs.djangoproject.com\",\"emoji\":\"📚\"}" \
  -o /dev/null || warn "Failed to create Django bookmark"

curl -sf -X POST "$BASE_URL/api/bookmarks/" \
  -H "Content-Type: application/json" \
  -d "{\"project_id\":\"$PYTHON_ID\",\"name\":\"Tailwind CSS\",\"url\":\"https://tailwindcss.com/docs\",\"emoji\":\"🎨\"}" \
  -o /dev/null || warn "Failed to create Tailwind bookmark"

curl -sf -X POST "$BASE_URL/api/bookmarks/" \
  -H "Content-Type: application/json" \
  -d "{\"project_id\":\"$GOLANG_ID\",\"name\":\"Go Packages\",\"url\":\"https://pkg.go.dev\",\"emoji\":\"🐹\"}" \
  -o /dev/null || warn "Failed to create Go bookmark"

curl -sf -X POST "$BASE_URL/api/bookmarks/" \
  -H "Content-Type: application/json" \
  -d "{\"project_id\":\"$PYTHON_ID\",\"name\":\"CI Dashboard\",\"url\":\"https://github.com/actions\",\"emoji\":\"🚀\"}" \
  -o /dev/null || warn "Failed to create CI bookmark"

echo "  Created 4 bookmarks"

# ============================================================
# Step 11: Create notifications
# ============================================================
step 11 "Creating notifications..."

curl -sf -X POST "$BASE_URL/api/notifications/" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"Build Succeeded\",\"body\":\"Python API CI pipeline passed all 47 tests in 2m 34s\",\"source\":\"ci\",\"level\":\"success\",\"project_id\":\"$PYTHON_ID\"}" \
  -o /dev/null || warn "Failed to create success notification"

curl -sf -X POST "$BASE_URL/api/notifications/" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"Claude Task Complete\",\"body\":\"Finished implementing JWT authentication middleware\",\"source\":\"claude\",\"level\":\"success\",\"project_id\":\"$PYTHON_ID\"}" \
  -o /dev/null || warn "Failed to create claude notification"

curl -sf -X POST "$BASE_URL/api/notifications/" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"Dependency Update\",\"body\":\"3 npm packages have security updates available\",\"source\":\"system\",\"level\":\"warning\",\"project_id\":\"$NODEJS_ID\"}" \
  -o /dev/null || warn "Failed to create warning notification"

echo "  Created 3 notifications"

# ============================================================
# Step 12: Create a project-scoped skill
# ============================================================
step 12 "Creating demo skill..."

curl -sf -X POST "$BASE_URL/projects/$PYTHON_ID/api/skills" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "deploy",
    "description": "Deploy the current project to production",
    "scope": "project",
    "content": "# Deploy\n\nRun the deployment pipeline for the current project.\n\n## Steps\n1. Run full test suite\n2. Build Docker image with version tag\n3. Push to container registry\n4. Deploy to Kubernetes staging\n5. Run smoke tests\n6. Promote to production"
  }' -o /dev/null || warn "Failed to create skill"

echo "  Created: deploy skill (project-scoped)"

# ============================================================
# Step 13: Create a project-scoped agent
# ============================================================
step 13 "Creating demo agent..."

curl -sf -X POST "$BASE_URL/projects/$PYTHON_ID/api/agents" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "code-reviewer",
    "description": "Reviews code changes for quality, security, and performance",
    "scope": "project",
    "model": "claude-sonnet-4-20250514",
    "content": "# Code Reviewer\n\nYou are a senior code reviewer. Analyze all staged changes and provide feedback on:\n\n- **Code quality**: readability, naming, structure\n- **Performance**: N+1 queries, unnecessary allocations, caching opportunities\n- **Security**: injection risks, auth bypasses, data exposure\n- **Test coverage**: missing edge cases, untested paths"
  }' -o /dev/null || warn "Failed to create agent"

echo "  Created: code-reviewer agent (project-scoped)"

# ============================================================
# Step 14: Create a project-scoped MCP server
# ============================================================
step 14 "Creating demo MCP server..."

curl -sf -X POST "$BASE_URL/projects/$PYTHON_ID/api/mcp-servers" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "context7",
    "command": "npx",
    "args": ["-y", "@upstash/context7-mcp"],
    "scope": "project"
  }' -o /dev/null || warn "Failed to create MCP server"

echo "  Created: context7 MCP server (project-scoped)"

# ============================================================
# Step 15: Create a code snippet
# ============================================================
step 15 "Creating demo snippet..."

curl -sf -X POST "$BASE_URL/api/snippets/" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "FastAPI Route Template",
    "language": "python",
    "content": "@app.get(\"/items/{item_id}\")\nasync def read_item(item_id: int, q: str = None):\n    \"\"\"Retrieve an item by ID with optional query filter.\"\"\"\n    return {\"item_id\": item_id, \"q\": q}"
  }' -o /dev/null || warn "Failed to create snippet"

echo "  Created: FastAPI Route Template snippet"

# ============================================================
# Done
# ============================================================
echo ""
echo -e "${GREEN}Demo environment setup complete!${NC}"
echo ""
echo "Projects:    $(jq '.projects | length' "$STATE_FILE")"
echo "Sessions:    $(jq '.sessions | length' "$STATE_FILE")"
echo "Features:    $(jq '.features | length' "$STATE_FILE")"
echo ""
echo "Ready for screenshot capture."
