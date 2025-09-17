package handlers

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/database"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/models"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/services"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type DocumentHandler struct {
	dbManager   *database.Manager
	pdfService  *services.PDFService
	nlpService  *services.NLPService
	graphService *services.GraphService
}

func NewDocumentHandler(dbManager *database.Manager) *DocumentHandler {
	return &DocumentHandler{
		dbManager:    dbManager,
		pdfService:   services.NewPDFService(),
		nlpService:   services.NewNLPService(),
		graphService: services.NewGraphService(dbManager.Neo4j),
	}
}

func (h *DocumentHandler) UploadPDF(c *fiber.Ctx) error {
	file, err := c.FormFile("pdf")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No PDF file provided",
		})
	}

	if !strings.HasSuffix(strings.ToLower(file.Filename), ".pdf") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File must be a PDF",
		})
	}

	maxSizeMB := int64(50)
	if file.Size > maxSizeMB*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("File size exceeds %d MB limit", maxSizeMB),
		})
	}

	documentID := uuid.New().String()

	document := &models.Document{
		ID:          documentID,
		Filename:    file.Filename,
		ContentType: "application/pdf",
		Size:        file.Size,
		UploadedAt:  time.Now(),
		Status:      models.StatusUploaded,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collection := h.dbManager.MongoDB.GetDocumentsCollection()
	_, err = collection.InsertOne(ctx, document)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save document metadata",
		})
	}

	go h.processDocument(documentID, file)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"document_id": documentID,
		"message":     "PDF uploaded successfully and processing started",
		"filename":    file.Filename,
		"size":        file.Size,
	})
}

func (h *DocumentHandler) processDocument(documentID string, file *multipart.FileHeader) {
	ctx := context.Background()
	collection := h.dbManager.MongoDB.GetDocumentsCollection()

	h.updateDocumentStatus(ctx, collection, documentID, models.StatusProcessing)

	src, err := file.Open()
	if err != nil {
		h.updateDocumentError(ctx, collection, documentID, fmt.Sprintf("Failed to open file: %v", err))
		return
	}
	defer src.Close()

	fileBytes, err := io.ReadAll(src)
	if err != nil {
		h.updateDocumentError(ctx, collection, documentID, fmt.Sprintf("Failed to read file: %v", err))
		return
	}

	reader := &bytesReaderAt{data: fileBytes}
	
	extractedText, err := h.pdfService.ExtractText(reader, file.Size)
	if err != nil {
		h.updateDocumentError(ctx, collection, documentID, fmt.Sprintf("Failed to extract text: %v", err))
		return
	}

	processedText := h.nlpService.ProcessText(extractedText.Content)

	coOccurrenceMatrix := h.nlpService.BuildCoOccurrenceMatrix(processedText.Nouns, processedText.Sentences, 5)

	graph, err := h.graphService.CreateGraphFromNLP(ctx, documentID, services.GraphTypeCoOccurrence, processedText.Nouns, coOccurrenceMatrix, 2)
	if err != nil {
		h.updateDocumentError(ctx, collection, documentID, fmt.Sprintf("Failed to create graph: %v", err))
		return
	}

	centroidResult, err := h.graphService.FindCentroid(ctx, graph)
	if err != nil {
		h.updateDocumentError(ctx, collection, documentID, fmt.Sprintf("Failed to find centroid: %v", err))
		return
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":         models.StatusCompleted,
			"processed_at":   now,
			"extracted_text": extractedText,
			"processed_text": processedText,
			"graph_id":       graph.ID,
		},
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": documentID}, update)
	if err != nil {
		h.updateDocumentError(ctx, collection, documentID, fmt.Sprintf("Failed to update document: %v", err))
		return
	}

	cacheKey := fmt.Sprintf("centroid:%s", documentID)
	h.dbManager.Redis.Set(ctx, cacheKey, fmt.Sprintf("%v", centroidResult), 24*time.Hour)
}

func (h *DocumentHandler) GetDocument(c *fiber.Ctx) error {
	documentID := c.Params("id")
	if documentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Document ID is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := h.dbManager.MongoDB.GetDocumentsCollection()
	var document models.Document
	
	err := collection.FindOne(ctx, bson.M{"_id": documentID}).Decode(&document)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Document not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve document",
		})
	}

	return c.JSON(document)
}

func (h *DocumentHandler) GetGraphNetwork(c *fiber.Ctx) error {
	documentID := c.Params("id")
	if documentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Document ID is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	graphID := fmt.Sprintf("graph_%s", documentID)
	
	nodesResult, err := h.dbManager.Neo4j.ExecuteQuery(ctx, 
		"MATCH (n {graph_id: $graphId}) RETURN n.name as name, n.frequency as frequency, n.centrality as centrality", 
		map[string]any{"graphId": graphID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve graph nodes",
		})
	}

	edgesResult, err := h.dbManager.Neo4j.ExecuteQuery(ctx,
		"MATCH (a {graph_id: $graphId})-[r:RELATES_TO]->(b {graph_id: $graphId}) RETURN a.name as from, b.name as to, r.weight as weight",
		map[string]any{"graphId": graphID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve graph edges",
		})
	}

	var nodes []services.GraphNode
	for _, record := range nodesResult.Records {
		name, _ := record.Get("name")
		frequency, _ := record.Get("frequency")
		centrality, _ := record.Get("centrality")
		
		node := services.GraphNode{
			Name:   fmt.Sprintf("%v", name),
			Weight: float64(frequency.(int64)),
		}
		if centrality != nil {
			node.Centrality = centrality.(float64)
		}
		nodes = append(nodes, node)
	}

	var edges []services.GraphEdge
	for i, record := range edgesResult.Records {
		weight, _ := record.Get("weight")
		
		edge := services.GraphEdge{
			From:   int64(i),
			To:     int64(i + 1),
			Weight: float64(weight.(int64)),
		}
		edges = append(edges, edge)
	}

	network := services.GraphNetwork{
		ID:         graphID,
		DocumentID: documentID,
		Nodes:      nodes,
		Edges:      edges,
		CreatedAt:  time.Now().Unix(),
	}

	return c.JSON(network)
}

func (h *DocumentHandler) GetCentroid(c *fiber.Ctx) error {
	documentID := c.Params("id")
	if documentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Document ID is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cacheKey := fmt.Sprintf("centroid:%s", documentID)
	cachedResult, err := h.dbManager.Redis.Get(ctx, cacheKey)
	if err == nil && cachedResult != "" {
		return c.JSON(fiber.Map{
			"document_id":      documentID,
			"centroid_result":  cachedResult,
			"source":          "cache",
		})
	}

	graphID := fmt.Sprintf("graph_%s", documentID)
	
	result, err := h.dbManager.Neo4j.ExecuteQuery(ctx,
		"MATCH (n {graph_id: $graphId}) WHERE n.centrality IS NOT NULL RETURN n.name as name, n.centrality as centrality, n.frequency as frequency ORDER BY n.centrality DESC LIMIT 1",
		map[string]any{"graphId": graphID})
	if err != nil || len(result.Records) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Centroid not found for this document",
		})
	}

	record := result.Records[0]
	name, _ := record.Get("name")
	centrality, _ := record.Get("centrality")
	frequency, _ := record.Get("frequency")

	centroidInfo := fiber.Map{
		"document_id": documentID,
		"centroid_node": fiber.Map{
			"name":       name,
			"centrality": centrality,
			"frequency":  frequency,
		},
		"source": "database",
	}

	h.dbManager.Redis.Set(ctx, cacheKey, fmt.Sprintf("%v", centroidInfo), 24*time.Hour)

	return c.JSON(centroidInfo)
}

func (h *DocumentHandler) ListDocuments(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := h.dbManager.MongoDB.GetDocumentsCollection()
	
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve documents",
		})
	}
	defer cursor.Close(ctx)

	var documents []models.Document
	if err = cursor.All(ctx, &documents); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to decode documents",
		})
	}

	return c.JSON(fiber.Map{
		"documents": documents,
		"count":     len(documents),
	})
}

func (h *DocumentHandler) DeleteDocument(c *fiber.Ctx) error {
	documentID := c.Params("id")
	if documentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Document ID is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := h.dbManager.MongoDB.GetDocumentsCollection()
	result, err := collection.DeleteOne(ctx, bson.M{"_id": documentID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete document",
		})
	}

	if result.DeletedCount == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Document not found",
		})
	}

	graphID := fmt.Sprintf("graph_%s", documentID)
	h.dbManager.Neo4j.ExecuteQuery(ctx, 
		"MATCH (n {graph_id: $graphId}) DETACH DELETE n", 
		map[string]any{"graphId": graphID})

	cacheKey := fmt.Sprintf("centroid:%s", documentID)
	h.dbManager.Redis.Delete(ctx, cacheKey)

	return c.JSON(fiber.Map{
		"message": "Document deleted successfully",
		"document_id": documentID,
	})
}

func (h *DocumentHandler) updateDocumentStatus(ctx context.Context, collection *mongo.Collection, documentID string, status models.ProcessingStatus) {
	update := bson.M{"$set": bson.M{"status": status}}
	collection.UpdateOne(ctx, bson.M{"_id": documentID}, update)
}

func (h *DocumentHandler) ProcessMinIODocument(documentID, minioKey, filename string, size int64) error {
	ctx := context.Background()
	collection := h.dbManager.MongoDB.GetDocumentsCollection()

	// Create document record
	document := &models.Document{
		ID:          documentID,
		Filename:    filename,
		ContentType: "application/pdf",
		Size:        size,
		UploadedAt:  time.Now(),
		Status:      models.StatusProcessing,
	}

	// Insert document record
	_, err := collection.InsertOne(ctx, document)
	if err != nil {
		return fmt.Errorf("failed to save document metadata: %w", err)
	}

	// Process in background
	go h.processMinIODocument(documentID, minioKey)

	return nil
}

func (h *DocumentHandler) processMinIODocument(documentID, minioKey string) {
	ctx := context.Background()
	collection := h.dbManager.MongoDB.GetDocumentsCollection()

	h.updateDocumentStatus(ctx, collection, documentID, models.StatusProcessing)

	// Extract text from MinIO-stored PDF
	extractedText, err := h.pdfService.ExtractTextFromMinIO(ctx, h.dbManager.MinIO, minioKey)
	if err != nil {
		h.updateDocumentError(ctx, collection, documentID, fmt.Sprintf("Failed to extract text: %v", err))
		return
	}

	// Process text with NLP
	processedText := h.nlpService.ProcessText(extractedText.Content)

	// Build co-occurrence matrix
	coOccurrenceMatrix := h.nlpService.BuildCoOccurrenceMatrix(processedText.Nouns, processedText.Sentences, 5)

	// Create graph network
	graph, err := h.graphService.CreateGraphFromNLP(ctx, documentID, services.GraphTypeCoOccurrence, processedText.Nouns, coOccurrenceMatrix, 2)
	if err != nil {
		h.updateDocumentError(ctx, collection, documentID, fmt.Sprintf("Failed to create graph: %v", err))
		return
	}

	// Find centroid
	centroidResult, err := h.graphService.FindCentroid(ctx, graph)
	if err != nil {
		h.updateDocumentError(ctx, collection, documentID, fmt.Sprintf("Failed to find centroid: %v", err))
		return
	}

	// Update document with results
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":         models.StatusCompleted,
			"processed_at":   now,
			"extracted_text": extractedText,
			"processed_text": processedText,
			"graph_id":       graph.ID,
		},
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": documentID}, update)
	if err != nil {
		h.updateDocumentError(ctx, collection, documentID, fmt.Sprintf("Failed to update document: %v", err))
		return
	}

	// Cache centroid result
	cacheKey := fmt.Sprintf("centroid:%s", documentID)
	h.dbManager.Redis.Set(ctx, cacheKey, fmt.Sprintf("%v", centroidResult), 24*time.Hour)
}

func (h *DocumentHandler) ProcessDocumentWithGraphType(c *fiber.Ctx) error {
	documentID := c.Params("documentId")
	graphType := c.Query("graph_type", "co_occurrence")
	threshold := c.QueryInt("threshold", 2)
	
	if documentID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "document_id is required",
		})
	}
	
	if graphType != string(services.GraphTypeSequence) && graphType != string(services.GraphTypeCoOccurrence) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "graph_type must be 'sequence' or 'co_occurrence'",
		})
	}
	
	// For sequence graphs, threshold is not applicable for edges (always 1), but can be used for node frequency filtering
	if graphType == string(services.GraphTypeSequence) && threshold < 1 {
		threshold = 1 // Minimum threshold for sequence graphs
	}
	
	ctx := context.Background()
	collection := h.dbManager.MongoDB.GetDocumentsCollection()
	
	var document models.Document
	err := collection.FindOne(ctx, bson.M{"_id": documentID}).Decode(&document)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Document not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to find document",
		})
	}
	
	if document.ExtractedText == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Document has not been processed yet",
		})
	}
	
	processedText := h.nlpService.ProcessText(document.ExtractedText.Content)
	
	var relationshipMatrix map[string]map[string]int
	var graphTypeEnum services.GraphType
	
	if graphType == string(services.GraphTypeSequence) {
		relationshipMatrix = h.nlpService.BuildSequenceMatrix(processedText.Nouns, document.ExtractedText.Content)
		graphTypeEnum = services.GraphTypeSequence
	} else {
		relationshipMatrix = h.nlpService.BuildCoOccurrenceMatrix(processedText.Nouns, processedText.Sentences, 5)
		graphTypeEnum = services.GraphTypeCoOccurrence
	}
	
	graph, err := h.graphService.CreateGraphFromNLP(ctx, documentID, graphTypeEnum, processedText.Nouns, relationshipMatrix, threshold)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create graph: %v", err),
		})
	}
	
	centroidResult, err := h.graphService.FindCentroid(ctx, graph)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to find centroid: %v", err),
		})
	}
	
	var message string
	if graphType == string(services.GraphTypeSequence) {
		message = fmt.Sprintf("Successfully created %s graph (nodes with frequency >= %d, edges show word sequence)", graphType, threshold)
	} else {
		message = fmt.Sprintf("Successfully created %s graph (nodes with frequency >= %d, edges with weight >= %d)", graphType, threshold, threshold)
	}
	
	return c.JSON(fiber.Map{
		"document_id":     documentID,
		"graph_type":      graphType,
		"threshold":       threshold,
		"graph":          graph,
		"centroid_result": centroidResult,
		"message":        message,
	})
}

func (h *DocumentHandler) updateDocumentError(ctx context.Context, collection *mongo.Collection, documentID string, errorMsg string) {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":        models.StatusError,
			"processed_at":  now,
			"error_message": errorMsg,
		},
	}
	collection.UpdateOne(ctx, bson.M{"_id": documentID}, update)
}

// Helper struct to implement io.ReaderAt interface for backwards compatibility
type bytesReaderAt struct {
	data []byte
}

func (r *bytesReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(r.data)) {
		return 0, io.EOF
	}
	
	n := copy(p, r.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	
	return n, nil
}