# API Documentation

This directory contains the API documentation for the Relational Knowledge Engineering Platform Server.

## Overview

The API documentation is built using **Scalar**, a modern, interactive API documentation tool that provides a beautiful and user-friendly interface for exploring the API.

## Accessing the Documentation

Once the server is running, you can access the API documentation at:

- **Interactive Docs**: http://localhost:8080/docs
- **OpenAPI Spec**: http://localhost:8080/docs/openapi.yaml
- **API Info**: http://localhost:8080/

## Features

The documentation includes:

### 📊 **Interactive API Explorer**
- Try out API endpoints directly from the documentation
- Real-time request/response examples
- Automatic code generation in multiple languages

### 📝 **Comprehensive Documentation**
- Detailed endpoint descriptions
- Request/response schemas
- Error handling examples
- Authentication information

### 🔍 **Search Functionality**
- Press `K` to open the search dialog
- Search through endpoints, schemas, and descriptions
- Quick navigation to any part of the API

## API Endpoints Overview

### Health & Status
- `GET /` - API information and available endpoints
- `GET /health` - Service health check

### Document Upload
#### Legacy Upload
- `POST /api/v1/upload-pdf` - Single PDF upload

#### Chunked Upload (Recommended for large files)
- `POST /api/v1/upload/initiate` - Start a new upload session
- `POST /api/v1/upload/chunk` - Upload file chunks
- `POST /api/v1/upload/{sessionId}/complete` - Complete upload
- `DELETE /api/v1/upload/{sessionId}/abort` - Cancel upload
- `GET /api/v1/upload/{sessionId}/progress` - Check upload progress

### Document Management
- `GET /api/v1/documents` - List all documents
- `GET /api/v1/documents/{id}` - Get specific document
- `DELETE /api/v1/documents/{id}` - Delete document

### Knowledge Graphs
- `GET /api/v1/graphs/{id}` - Get knowledge graph network
- `GET /api/v1/graphs/{id}/centroid` - Get graph centrality analysis

## Quick Start Examples

### 1. Check API Status
```bash
curl http://localhost:8080/health
```

### 2. Upload a PDF (Legacy)
```bash
curl -X POST http://localhost:8080/api/v1/upload-pdf \
  -F "file=@document.pdf"
```

### 3. Chunked Upload Process
```bash
# 1. Initiate upload
curl -X POST http://localhost:8080/api/v1/upload/initiate \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "large-document.pdf",
    "file_size": 52428800,
    "content_type": "application/pdf"
  }'

# 2. Upload chunks (repeat for each chunk)
curl -X POST http://localhost:8080/api/v1/upload/chunk \
  -F "session_id=your-session-id" \
  -F "chunk_index=1" \
  -F "chunk=@chunk1.bin"

# 3. Complete upload
curl -X POST http://localhost:8080/api/v1/upload/your-session-id/complete
```

### 4. List Documents
```bash
curl http://localhost:8080/api/v1/documents
```

## Error Handling

The API uses standard HTTP status codes and returns consistent error responses:

```json
{
  "error": "Error description",
  "code": 400,
  "details": {
    "field": "Additional error context"
  }
}
```

Common status codes:
- `200` - Success
- `400` - Bad Request (invalid parameters)
- `404` - Resource not found
- `500` - Internal server error

## Development

### Updating the Documentation

1. Edit the OpenAPI specification in `docs/openapi.yaml`
2. The changes will be automatically reflected in the Scalar documentation
3. No restart required - just refresh the docs page

### Local Testing

The documentation is served directly by the application server, so you can test it locally by:

1. Starting the server: `go run command/server/main.go`
2. Opening http://localhost:8080/docs in your browser

## Contributing

When adding new endpoints or modifying existing ones:

1. Update the OpenAPI specification in `docs/openapi.yaml`
2. Add appropriate examples and descriptions
3. Test the documentation locally
4. Ensure all schema definitions are complete

## Troubleshooting

### Documentation Not Loading
- Ensure the server is running on the correct port
- Check that `docs/openapi.yaml` exists and is valid YAML
- Verify network connectivity and firewall settings

### OpenAPI Spec Errors
- Validate the YAML syntax
- Check that all `$ref` references are correct
- Ensure all required fields are defined in schemas

### Performance Issues
- The documentation is cached by the browser
- Force refresh with Ctrl+F5 or Cmd+Shift+R
- Clear browser cache if needed

## Additional Resources

- [Scalar Documentation](https://github.com/scalar/scalar)
- [OpenAPI Specification](https://swagger.io/specification/)
- [API Design Best Practices](https://swagger.io/blog/api-design/api-design-best-practices/)