# Relational Knowledge Engineering Platform Server

A Go-based server that extracts text from PDF documents, processes it using NLP to identify key nouns, and creates graph networks to find the most central concepts (centroid nodes) in the document.

## Features

- **PDF Text Extraction**: Upload and extract text from PDF documents
- **NLP Processing**: Extract nouns and analyze text relationships
- **Graph Network Creation**: Build graph networks from text relationships
- **Centroid Analysis**: Find the most central node in the graph (lowest average path length to all other nodes)
- **Multi-Database Support**: MongoDB, DragonflyDB (Redis-compatible), Neo4j, and Qdrant
- **RESTful API**: Clean REST endpoints for all operations
- **Caching**: Redis caching for improved performance
- **Docker Support**: Full Docker Compose setup for all services

## Architecture

### Databases Used

- **MongoDB**: Document storage (PDF metadata, extracted text, processing logs)
- **DragonflyDB**: High-performance caching and session management
- **Neo4j**: Graph database for storing and querying word relationships
- **Qdrant**: Vector database for future semantic similarity features

### Services

- **PDF Service**: Text extraction from PDF files
- **NLP Service**: Natural language processing, noun extraction, co-occurrence analysis
- **Graph Service**: Graph network creation and centroid calculation
- **Database Clients**: Connection management for all databases

## Quick Start

### Prerequisites

- Go 1.25+
- Docker and Docker Compose

### Setup

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd relational-knowledge-engineering-platform-server
   ```

2. **Start the databases**
   ```bash
   docker-compose up -d
   ```

3. **Run the server**
   ```bash
   go run command/server/main.go
   ```

   Or build and run:
   ```bash
   go build -o server command/server/main.go
   ./server
   ```

The server will start on `localhost:3000` by default.

## API Endpoints

### Document Management

- `POST /api/v1/upload-pdf` - Upload a PDF file for processing
- `GET /api/v1/documents` - List all documents
- `GET /api/v1/documents/:id` - Get document details
- `DELETE /api/v1/documents/:id` - Delete a document

### Graph Analysis

- `GET /api/v1/graphs/:id` - Get graph network for a document
- `GET /api/v1/graphs/:id/centroid` - Get centroid analysis for a document

### Health Check

- `GET /health` - Server health status
- `GET /` - Welcome message

## Usage Example

### 1. Upload a PDF

```bash
curl -X POST -F "pdf=@document.pdf" http://localhost:3000/api/v1/upload-pdf
```

Response:
```json
{
  "document_id": "uuid-here",
  "message": "PDF uploaded successfully and processing started",
  "filename": "document.pdf",
  "size": 1024576
}
```

### 2. Check Processing Status

```bash
curl http://localhost:3000/api/v1/documents/uuid-here
```

### 3. Get Graph Network

```bash
curl http://localhost:3000/api/v1/graphs/uuid-here
```

### 4. Get Centroid Analysis

```bash
curl http://localhost:3000/api/v1/graphs/uuid-here/centroid
```

Response:
```json
{
  "document_id": "uuid-here",
  "centroid_node": {
    "name": "system",
    "centrality": 0.95,
    "frequency": 15
  },
  "source": "database"
}
```

## Configuration

The server can be configured using environment variables:

### Server Configuration
- `SERVER_HOST` (default: "localhost")
- `SERVER_PORT` (default: 3000)
- `UPLOAD_MAX_SIZE_MB` (default: 50)

### Database Configuration

#### MongoDB
- `MONGODB_HOST` (default: "localhost")
- `MONGODB_PORT` (default: 27017)
- `MONGODB_USERNAME` (default: "app_user")
- `MONGODB_PASSWORD` (default: "app_password123")
- `MONGODB_DATABASE` (default: "relational_knowledge_db")

#### Redis/DragonflyDB
- `REDIS_HOST` (default: "localhost")
- `REDIS_PORT` (default: 6379)
- `REDIS_PASSWORD` (default: "")
- `REDIS_DB` (default: 0)

#### Neo4j
- `NEO4J_HOST` (default: "localhost")
- `NEO4J_PORT` (default: 7687)
- `NEO4J_USERNAME` (default: "neo4j")
- `NEO4J_PASSWORD` (default: "password123")
- `NEO4J_DATABASE` (default: "neo4j")

#### Qdrant
- `QDRANT_HOST` (default: "localhost")
- `QDRANT_PORT` (default: 6333)
- `QDRANT_API_KEY` (default: "")

## Development

### Running Tests

```bash
go test ./...
```

### Project Structure

```
├── command/server/          # Main application entry point
├── internal/
│   ├── bootstrap/          # Application bootstrap and routing
│   ├── config/            # Configuration management
│   ├── database/          # Database clients and connections
│   ├── handlers/          # HTTP handlers
│   ├── models/           # Data models
│   └── services/         # Business logic services
├── scripts/              # Database initialization scripts
├── docker-compose.yml    # Docker services configuration
└── README.md
```

### Key Algorithms

#### Noun Extraction
The NLP service uses pattern matching and linguistic rules to identify likely nouns:
- Filters stop words and short words
- Identifies words with noun-like suffixes (-tion, -ness, -ment, etc.)
- Excludes verb forms (-ing, -ed) and adverbs (-ly)
- Uses stemming for word normalization

#### Graph Creation
- Builds co-occurrence matrices based on word proximity in sentences
- Creates weighted edges between nouns that appear together
- Stores relationships in Neo4j for efficient graph queries

#### Centroid Calculation
- Uses Dijkstra's algorithm to find shortest paths between all nodes
- Calculates average path length for each node
- Identifies the node with the lowest average path length as the centroid
- Represents the most "central" concept that can reach all other concepts efficiently

## Docker Services

The `docker-compose.yml` includes:
- **MongoDB 7.0** with authentication
- **DragonflyDB 1.15** (Redis-compatible)
- **Neo4j 5.15 Community** with APOC plugins
- **Qdrant 1.7.4** for vector operations

All services are networked together and include persistent volumes.

## Performance Considerations

- **Async Processing**: PDF processing happens in background goroutines
- **Caching**: Redis caching for centroid results and frequent queries
- **Connection Pooling**: Efficient database connection management
- **Graceful Shutdown**: Proper cleanup of database connections

## Future Enhancements

- Semantic similarity using Qdrant vector storage
- Support for other document types (DOCX, TXT)
- Advanced NLP using transformer models
- Web interface for document management
- Batch processing capabilities
- API rate limiting and authentication

## License

MIT License