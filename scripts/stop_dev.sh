#!/bin/bash

# ==============================================================================
# PromptRadar Local Development Stopper
# Gracefully kills processes running on 8080, 8081, and 4200
# ==============================================================================

echo "🛑 Stopping PromptRadar Services..."

kill_port() {
    local port=$1
    local name=$2
    local pid=$(lsof -ti :$port || true)
    if [ -n "$pid" ]; then
        echo "   Stopping $name on port $port (PID: $pid)..."
        kill -9 $pid 2>/dev/null || true
    else
        echo "   Port $port ($name) is already free."
    fi
}

kill_port 8080 "Spring Boot Control Plane"
kill_port 8081 "Go Proxy Data Plane"
kill_port 4200 "Angular UI"

echo "✅ All PromptRadar servers stopped."
