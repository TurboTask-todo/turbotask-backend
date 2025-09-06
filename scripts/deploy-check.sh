#!/bin/bash

# QuantumTask API Deployment Pre-Check Script
# Run this before deploying to Render

set -e

echo "🚀 QuantumTask API Deployment Pre-Check"
echo "======================================="

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Error: go.mod not found. Run this script from the backend-go directory."
    exit 1
fi

echo "✅ Found go.mod file"

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "🔍 Go version: $GO_VERSION"

# Check if required files exist
echo ""
echo "📋 Checking required deployment files:"

files=("Dockerfile" ".dockerignore" "cmd/server/main.go")
for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file exists"
    else
        echo "❌ $file missing - required for deployment"
        exit 1
    fi
done

# Check go.mod and go.sum
echo ""
echo "📦 Checking Go modules:"
if go mod verify; then
    echo "✅ Go modules verified"
else
    echo "❌ Go modules verification failed"
    echo "🔧 Running go mod tidy..."
    go mod tidy
fi

# Test build
echo ""
echo "🔨 Testing build:"
if go build -o ./tmp/main ./cmd/server; then
    echo "✅ Build successful"
    rm -f ./tmp/main
    rmdir ./tmp 2>/dev/null || true
else
    echo "❌ Build failed - fix build errors before deploying"
    exit 1
fi

# Check for common deployment issues
echo ""
echo "🔍 Checking for potential deployment issues:"

# Check if PORT is handled in code
if grep -r "PORT" cmd/ internal/ >/dev/null 2>&1; then
    echo "✅ PORT environment variable handling found"
else
    echo "⚠️  Warning: Make sure your app reads PORT from environment variable"
fi

# Check if health endpoint exists
if grep -r "/health" cmd/ internal/ >/dev/null 2>&1; then
    echo "✅ Health endpoint found"
else
    echo "⚠️  Warning: Health endpoint not found - recommended for Render"
fi

# Check database connection handling
if grep -r "DATABASE_URL\|DB_URL" cmd/ internal/ >/dev/null 2>&1; then
    echo "✅ Database URL environment variable handling found"
else
    echo "⚠️  Warning: Make sure your app reads DATABASE_URL from environment"
fi

# Check for hardcoded localhost references
if grep -r "localhost\|127.0.0.1" cmd/ internal/ | grep -v "test\|example" >/dev/null 2>&1; then
    echo "⚠️  Warning: Found localhost references - make sure they're configurable"
    grep -r "localhost\|127.0.0.1" cmd/ internal/ | grep -v "test\|example" | head -3
else
    echo "✅ No hardcoded localhost references found"
fi

# Check dependencies
echo ""
echo "📚 Checking key dependencies:"
key_deps=("github.com/gin-gonic/gin" "github.com/lib/pq" "github.com/golang-jwt/jwt")
for dep in "${key_deps[@]}"; do
    if grep "$dep" go.mod >/dev/null 2>&1; then
        echo "✅ $dep found in dependencies"
    else
        echo "ℹ️  $dep not found (may not be needed)"
    fi
done

echo ""
echo "🎯 Pre-deployment checklist:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Before deploying to Render, make sure:"
echo "1. 📁 Push all changes to your GitHub repository"
echo "2. 🔐 Prepare your environment variables (JWT_SECRET, etc.)"
echo "3. 🗄️ Set up PostgreSQL and Redis services on Render first"
echo "4. 🌐 Update OAuth redirect URLs for production domain"
echo "5. 📧 Configure SMTP settings for email functionality"
echo "6. 🔍 Test your endpoints after deployment"
echo ""
echo "💰 Estimated monthly cost on Render:"
echo "   • Starter plan: ~$21/month (Web + DB + Redis)"
echo "   • Production plan: ~$60/month (recommended)"
echo ""
echo "📖 For detailed deployment steps, see DEPLOYMENT_GUIDE.md"
echo ""
echo "🚀 Ready for deployment! Good luck!"
