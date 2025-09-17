#!/bin/bash

# Demo script for API Documentation
# This script demonstrates how to access and use the API documentation

set -e

echo "🚀 Relational Knowledge Engineering Platform - API Documentation Demo"
echo "=================================================================="
echo

# Check if server is running
SERVER_URL="http://localhost:8080"
echo "📡 Checking if server is running at $SERVER_URL..."

if curl -s "$SERVER_URL/health" > /dev/null; then
    echo "✅ Server is running!"
else
    echo "❌ Server is not running. Please start the server first:"
    echo "   go run command/server/main.go"
    echo
    exit 1
fi

echo

# Demo 1: Get API Information
echo "📖 Demo 1: Getting API Information"
echo "GET $SERVER_URL/"
echo
curl -s "$SERVER_URL/" | jq '.' || curl -s "$SERVER_URL/"
echo
echo

# Demo 2: Check Health
echo "🏥 Demo 2: Health Check"
echo "GET $SERVER_URL/health"
echo
curl -s "$SERVER_URL/health" | jq '.' || curl -s "$SERVER_URL/health"
echo
echo

# Demo 3: OpenAPI Specification
echo "📋 Demo 3: OpenAPI Specification"
echo "GET $SERVER_URL/docs/openapi.yaml"
echo
echo "First 10 lines of OpenAPI spec:"
curl -s "$SERVER_URL/docs/openapi.yaml" | head -10
echo
echo

# Demo 4: Interactive Documentation
echo "🌐 Demo 4: Interactive Documentation"
echo
echo "The interactive API documentation is available at:"
echo "   🔗 $SERVER_URL/docs"
echo
echo "Features:"
echo "   • Interactive API explorer"
echo "   • Try endpoints directly from the browser"
echo "   • Search functionality (press 'K' to search)"
echo "   • Code generation for multiple languages"
echo "   • Real-time request/response examples"
echo

# Demo 5: Example API Calls
echo "🔧 Demo 5: Example API Calls"
echo

echo "📝 List Documents:"
echo "GET $SERVER_URL/api/v1/documents"
curl -s "$SERVER_URL/api/v1/documents" | jq '.' || curl -s "$SERVER_URL/api/v1/documents"
echo
echo

echo "📤 Initiate Upload (example):"
echo "POST $SERVER_URL/api/v1/upload/initiate"
echo "Body: {\"filename\": \"test.pdf\", \"file_size\": 1024, \"content_type\": \"application/pdf\"}"
echo

# Demo upload initiation (will likely fail due to validation, but shows the endpoint)
curl -s -X POST "$SERVER_URL/api/v1/upload/initiate" \
  -H "Content-Type: application/json" \
  -d '{"filename": "test.pdf", "file_size": 1024, "content_type": "application/pdf"}' \
  | jq '.' 2>/dev/null || echo "Expected: This may fail due to validation, but endpoint is accessible"

echo
echo

# Summary
echo "📚 Summary"
echo "========="
echo
echo "The API documentation has been successfully integrated with Scalar!"
echo
echo "Available endpoints:"
echo "   📊 Interactive Docs: $SERVER_URL/docs"
echo "   📋 OpenAPI Spec:     $SERVER_URL/docs/openapi.yaml"
echo "   ℹ️  API Info:         $SERVER_URL/"
echo "   🏥 Health Check:     $SERVER_URL/health"
echo
echo "Next steps:"
echo "   1. Open $SERVER_URL/docs in your browser"
echo "   2. Explore the interactive documentation"
echo "   3. Try out API endpoints directly from the docs"
echo "   4. Use the search feature (press 'K') to find specific endpoints"
echo
echo "Happy coding! 🎉"