#!/bin/bash

# Quick start script for Todo API

set -e

echo "🚀 Starting Todo API..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

# Start PostgreSQL
echo "📦 Starting PostgreSQL..."
docker-compose up -d

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL to be ready..."
sleep 5

# Check if PostgreSQL is ready
until docker exec todo-api-postgres pg_isready -U todo_user > /dev/null 2>&1; do
    echo "   Waiting for PostgreSQL..."
    sleep 1
done

echo "✅ PostgreSQL is ready!"

# Set default environment variables if not set
export DATABASE_URL=${DATABASE_URL:-"postgres://todo_user:todo_password@localhost:5432/todo_db?sslmode=disable"}
export JWT_SECRET=${JWT_SECRET:-"your-secret-key-change-in-production-min-32-chars"}

echo "🔧 Configuration:"
echo "   DATABASE_URL: $DATABASE_URL"
echo "   JWT_SECRET: ${JWT_SECRET:0:10}..."

# Build and run
echo "🔨 Building application..."
go build -o todo-api ./main.go

echo "🎯 Starting Todo API server..."
echo ""
echo "API will be available at: http://localhost:8080"
echo ""
echo "Try these commands:"
echo "  curl http://localhost:8080/health"
echo "  curl http://localhost:8080/metrics"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

./todo-api
