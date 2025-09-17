package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/database"
	"gonum.org/v1/gonum/graph/path"
	"gonum.org/v1/gonum/graph/simple"
)

type GraphService struct {
	neo4jClient *database.Neo4jClient
}

type GraphNode struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Weight   float64 `json:"weight"`
	Centrality float64 `json:"centrality"`
}

type GraphEdge struct {
	From   int64   `json:"from"`
	To     int64   `json:"to"`
	Weight float64 `json:"weight"`
}

type GraphType string

const (
	GraphTypeSequence     GraphType = "sequence"
	GraphTypeCoOccurrence GraphType = "co_occurrence"
)

type GraphNetwork struct {
	ID          string      `json:"id"`
	DocumentID  string      `json:"document_id"`
	Type        GraphType   `json:"type"`
	Nodes       []GraphNode `json:"nodes"`
	Edges       []GraphEdge `json:"edges"`
	CentroidNode *GraphNode  `json:"centroid_node,omitempty"`
	CreatedAt   int64       `json:"created_at"`
}

type CentroidResult struct {
	Node                *GraphNode `json:"node"`
	AveragePathLength   float64    `json:"average_path_length"`
	Closeness          float64    `json:"closeness"`
	TotalConnections   int        `json:"total_connections"`
}

func NewGraphService(neo4jClient *database.Neo4jClient) *GraphService {
	return &GraphService{
		neo4jClient: neo4jClient,
	}
}

func (g *GraphService) CreateGraphFromNLP(ctx context.Context, documentID string, graphType GraphType, nouns []NounEntity, relationshipMatrix map[string]map[string]int, threshold int) (*GraphNetwork, error) {
	graphID := fmt.Sprintf("graph_%s", documentID)
	
	err := g.clearExistingGraph(ctx, graphID)
	if err != nil {
		return nil, fmt.Errorf("failed to clear existing graph: %w", err)
	}

	nodeMap := make(map[string]int64)
	nodes := make([]GraphNode, 0)
	edges := make([]GraphEdge, 0)

	nodeID := int64(1)
	for _, noun := range nouns {
		// For sequence graphs, include all nouns; for co-occurrence graphs, apply frequency threshold
		shouldIncludeNode := false
		if graphType == GraphTypeSequence {
			shouldIncludeNode = noun.Frequency >= 1 // Include all words in sequence graphs
		} else {
			shouldIncludeNode = noun.Frequency >= threshold // Apply threshold for co-occurrence graphs
		}
		
		if shouldIncludeNode {
			node := GraphNode{
				ID:     nodeID,
				Name:   noun.Word,
				Weight: float64(noun.Frequency),
			}
			nodes = append(nodes, node)
			nodeMap[noun.Word] = nodeID
			nodeID++

			err := g.neo4jClient.CreateNode(ctx, "Word", noun.Word, map[string]interface{}{
				"graph_id":   graphID,
				"graph_type": string(graphType),
				"frequency":  noun.Frequency,
				"stemmed":    noun.Stemmed,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create neo4j node for %s: %w", noun.Word, err)
			}
		}
	}

	for word1, connections := range relationshipMatrix {
		fromID, fromExists := nodeMap[word1]
		if !fromExists {
			continue
		}

		for word2, weight := range connections {
			toID, toExists := nodeMap[word2]
			if !toExists {
				continue
			}
			
			// For sequence graphs, always include edges (weight >= 1); for co-occurrence graphs, apply threshold
			shouldIncludeEdge := false
			if graphType == GraphTypeSequence {
				shouldIncludeEdge = weight >= 1 // Always include sequence edges
			} else {
				shouldIncludeEdge = weight >= threshold // Apply threshold for co-occurrence edges
			}
			
			if !shouldIncludeEdge {
				continue
			}

			edge := GraphEdge{
				From:   fromID,
				To:     toID,
				Weight: float64(weight),
			}
			edges = append(edges, edge)

			relationshipType := "RELATES_TO"
			if graphType == GraphTypeSequence {
				relationshipType = "FOLLOWS"
			}
			
			err := g.neo4jClient.CreateRelationship(ctx, word1, word2, relationshipType, map[string]interface{}{
				"weight":     weight,
				"graph_id":   graphID,
				"graph_type": string(graphType),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create neo4j relationship between %s and %s: %w", word1, word2, err)
			}
		}
	}

	network := &GraphNetwork{
		ID:         graphID,
		DocumentID: documentID,
		Type:       graphType,
		Nodes:      nodes,
		Edges:      edges,
		CreatedAt:  generateTimestamp(),
	}

	return network, nil
}

func (g *GraphService) FindCentroid(ctx context.Context, network *GraphNetwork) (*CentroidResult, error) {
	if len(network.Nodes) == 0 {
		return nil, fmt.Errorf("graph has no nodes")
	}

	graph := g.buildGonumGraph(network)
	
	allPaths := path.DijkstraAllPaths(graph)

	var bestNode *GraphNode
	bestScore := math.Inf(1)
	var bestResult *CentroidResult

	for _, node := range network.Nodes {
		result := g.calculateNodeCentrality(graph, allPaths, node, network.Nodes)
		
		if result.AveragePathLength < bestScore && result.AveragePathLength > 0 {
			bestScore = result.AveragePathLength
			bestNode = &node
			bestResult = result
		}
	}

	if bestNode != nil {
		bestNode.Centrality = bestResult.Closeness
		network.CentroidNode = bestNode
		
		err := g.updateNodeCentrality(ctx, network.ID, bestNode.Name, bestResult.Closeness)
		if err != nil {
			return nil, fmt.Errorf("failed to update node centrality in neo4j: %w", err)
		}
	}

	return bestResult, nil
}

func (g *GraphService) buildGonumGraph(network *GraphNetwork) *simple.WeightedUndirectedGraph {
	graph := simple.NewWeightedUndirectedGraph(0, math.Inf(1))

	nodeMap := make(map[int64]simple.Node)
	for _, node := range network.Nodes {
		graphNode := graph.NewNode()
		graph.AddNode(graphNode)
		nodeMap[node.ID] = graphNode.(simple.Node)
	}

	for _, edge := range network.Edges {
		fromNode, fromExists := nodeMap[edge.From]
		toNode, toExists := nodeMap[edge.To]
		
		if fromExists && toExists {
			weight := 1.0 / edge.Weight
			if weight <= 0 {
				weight = 1.0
			}
			weightedEdge := graph.NewWeightedEdge(fromNode, toNode, weight)
			graph.SetWeightedEdge(weightedEdge)
		}
	}

	return graph
}

func (g *GraphService) calculateNodeCentrality(graph *simple.WeightedUndirectedGraph, allPaths path.AllShortest, targetNode GraphNode, allNodes []GraphNode) *CentroidResult {
	nodeMap := make(map[int64]simple.Node)
	reverseNodeMap := make(map[int64]int64)
	
	nodes := graph.Nodes()
	i := int64(0)
	for nodes.Next() {
		node := nodes.Node()
		simpleNode := node.(simple.Node)
		nodeMap[allNodes[i].ID] = simpleNode
		reverseNodeMap[simpleNode.ID()] = allNodes[i].ID
		i++
	}

	targetGraphNode, exists := nodeMap[targetNode.ID]
	if !exists {
		return &CentroidResult{
			Node:              &targetNode,
			AveragePathLength: math.Inf(1),
			Closeness:         0,
			TotalConnections:  0,
		}
	}

	totalDistance := 0.0
	reachableNodes := 0
	
	for _, otherNode := range allNodes {
		if otherNode.ID == targetNode.ID {
			continue
		}
		
		otherGraphNode, exists := nodeMap[otherNode.ID]
		if !exists {
			continue
		}

		path, weight, exists := allPaths.Between(targetGraphNode.ID(), otherGraphNode.ID())
		if exists && len(path) > 0 && weight < math.Inf(1) {
			totalDistance += weight
			reachableNodes++
		}
	}

	var avgPathLength float64
	var closeness float64
	
	if reachableNodes > 0 {
		avgPathLength = totalDistance / float64(reachableNodes)
		closeness = float64(reachableNodes) / totalDistance
	} else {
		avgPathLength = math.Inf(1)
		closeness = 0
	}

	return &CentroidResult{
		Node:              &targetNode,
		AveragePathLength: avgPathLength,
		Closeness:         closeness,
		TotalConnections:  reachableNodes,
	}
}

func (g *GraphService) GetTopCentralNodes(ctx context.Context, network *GraphNetwork, topN int) ([]CentroidResult, error) {
	if len(network.Nodes) == 0 {
		return nil, fmt.Errorf("graph has no nodes")
	}

	graph := g.buildGonumGraph(network)
	allPaths := path.DijkstraAllPaths(graph)

	var results []CentroidResult
	
	for _, node := range network.Nodes {
		result := g.calculateNodeCentrality(graph, allPaths, node, network.Nodes)
		results = append(results, *result)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Closeness == results[j].Closeness {
			return results[i].AveragePathLength < results[j].AveragePathLength
		}
		return results[i].Closeness > results[j].Closeness
	})

	if topN > len(results) {
		topN = len(results)
	}

	return results[:topN], nil
}

func (g *GraphService) clearExistingGraph(ctx context.Context, graphID string) error {
	cypher := "MATCH (n {graph_id: $graphId}) DETACH DELETE n"
	params := map[string]any{"graphId": graphID}
	
	_, err := g.neo4jClient.ExecuteQuery(ctx, cypher, params)
	return err
}

func (g *GraphService) updateNodeCentrality(ctx context.Context, graphID, nodeName string, centrality float64) error {
	cypher := "MATCH (n {graph_id: $graphId, name: $name}) SET n.centrality = $centrality"
	params := map[string]any{
		"graphId":    graphID,
		"name":       nodeName,
		"centrality": centrality,
	}
	
	_, err := g.neo4jClient.ExecuteQuery(ctx, cypher, params)
	return err
}

func generateTimestamp() int64 {
	return time.Now().Unix()
}