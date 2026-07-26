#!/bin/bash
set -e

echo "Starting Postgres..."
if [ ! "$(docker ps -q -f name=nodefall-postgres)" ]; then
  if [ "$(docker ps -aq -f name=nodefall-postgres)" ]; then
    docker start nodefall-postgres
  else
    docker run --name nodefall-postgres \
      -e POSTGRES_USER=nodefall \
      -e POSTGRES_PASSWORD=devpassword \
      -e POSTGRES_DB=nodefall \
      -p 5432:5432 \
      -d postgres:16
  fi
fi

echo "Installing Go dependencies..."
go mod download

echo "Installing frontend dependencies..."
(cd frontend/app && npm install)

echo ""
echo "Done. Still needed manually:"
echo "  1. Copy .env.example to .env (or export NODEFALL_JWT_SECRET / NODEFALL_DB_URL in your shell)"
echo "  2. Run services individually:"
echo "     go run ./services/auth"
echo "     go run ./services/game/main.go"
echo "     cd frontend/app && npm run dev"
