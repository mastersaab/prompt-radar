#!/bin/bash

# ==============================================================================
# PromptRadar Local Development Runner
# Launches Spring Boot (8080), Go Proxy (8081), and Angular UI (4200)
# ==============================================================================

set -e
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export JAVA_HOME="/opt/homebrew/opt/openjdk@21"

echo "🎯 Starting PromptRadar Services..."

# 1. Start Spring Boot Control Plane (Port 8080)
echo "📦 [1/3] Starting Spring Boot Control Plane (:8080)..."
if [ -f "$PROJECT_ROOT/apps/radar-control-plane/build/libs/radar-control-plane-0.0.1-SNAPSHOT.jar" ]; then
    $JAVA_HOME/bin/java -jar "$PROJECT_ROOT/apps/radar-control-plane/build/libs/radar-control-plane-0.0.1-SNAPSHOT.jar" > "$PROJECT_ROOT/spring.log" 2>&1 &
    SPRING_PID=$!
else
    (cd "$PROJECT_ROOT/apps/radar-control-plane" && ./gradlew bootRun > "$PROJECT_ROOT/spring.log" 2>&1 &)
    SPRING_PID=$!
fi
echo "   ↳ Spring Boot PID: $SPRING_PID (logs: spring.log)"

# 2. Start Go Proxy Data Plane (Port 8081)
echo "⚡ [2/3] Starting Go Proxy Data Plane (:8081)..."
if [ -f "$PROJECT_ROOT/apps/radar-proxy/bin/radar-proxy" ]; then
    (cd "$PROJECT_ROOT" && ./apps/radar-proxy/bin/radar-proxy > "$PROJECT_ROOT/proxy.log" 2>&1 &)
    PROXY_PID=$!
else
    (cd "$PROJECT_ROOT/apps/radar-proxy" && go run ./cmd/server > "$PROJECT_ROOT/proxy.log" 2>&1 &)
    PROXY_PID=$!
fi
echo "   ↳ Go Proxy PID: $PROXY_PID (logs: proxy.log)"

# 3. Start Angular UI (Port 4200)
echo "🌐 [3/3] Starting Angular 17 UI (:4200)..."
(cd "$PROJECT_ROOT/apps/radar-ui" && npm start -- --port 4200 > "$PROJECT_ROOT/ui.log" 2>&1 &)
UI_PID=$!
echo "   ↳ Angular UI PID: $UI_PID (logs: ui.log)"

echo ""
echo "✅ All PromptRadar servers started!"
echo "   • Angular UI:     http://localhost:4200"
echo "   • Go Proxy:       http://localhost:8081"
echo "   • Spring Boot:    http://localhost:8080"
echo ""
echo "To stop all servers at any time, run: ./scripts/stop_dev.sh"
