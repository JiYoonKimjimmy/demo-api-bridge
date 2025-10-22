#!/bin/bash

# API Documentation Generation Script (Bash)
# Generates docs.go automatically based on swagger.yaml file.

echo "Starting API documentation generation..."

# Move to project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "Project root: $PROJECT_ROOT"

# 2. Execute swag init to regenerate docs.go
echo "Running swag init..."
if swag init -g cmd/api-bridge/main.go -o api-docs; then
    echo "docs.go generated successfully!"
else
    echo "Failed to run swag init"
    exit 1
fi

# 3. Delete swagger.json file (keep YAML only)
if [ -f "api-docs/swagger.json" ]; then
    rm "api-docs/swagger.json"
    echo "swagger.json file deleted (YAML only)"
fi

# 4. Remove LeftDelim, RightDelim from docs.go (swag version compatibility)
echo "Fixing docs.go compatibility..."
sed -i.bak 's/LeftDelim:[[:space:]]*"[^"]*",[[:space:]]*//g' api-docs/docs.go
sed -i.bak 's/RightDelim:[[:space:]]*"[^"]*",[[:space:]]*//g' api-docs/docs.go
sed -i.bak 's/,[[:space:]]*LeftDelim:[[:space:]]*"[^"]*"//g' api-docs/docs.go
sed -i.bak 's/,[[:space:]]*RightDelim:[[:space:]]*"[^"]*"//g' api-docs/docs.go
rm -f api-docs/docs.go.bak
echo "docs.go compatibility fixed!"

echo "API documentation generation completed!"
echo "Updated files:"
echo "   - api-docs/docs.go"
echo "   - api-docs/swagger.yaml"
echo ""
echo "Usage:"
echo "   Run this script after modifying swagger.yaml file"
echo "   ./scripts/generate-docs.sh"
