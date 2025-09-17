package database

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/yothgewalt/relational-knowledge-engineering-platform-server/internal/config"
)

type Neo4jClient struct {
	Driver neo4j.DriverWithContext
	config config.Neo4jConfig
}

func NewNeo4jClient(cfg config.Neo4jConfig) (*Neo4jClient, error) {
	uri := fmt.Sprintf("bolt://%s:%d", cfg.Host, cfg.Port)
	
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	err = driver.VerifyConnectivity(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	return &Neo4jClient{
		Driver: driver,
		config: cfg,
	}, nil
}

func (n *Neo4jClient) Close(ctx context.Context) error {
	return n.Driver.Close(ctx)
}

func (n *Neo4jClient) ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (*neo4j.EagerResult, error) {
	return neo4j.ExecuteQuery(ctx, n.Driver, cypher, params,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(n.config.Database))
}

func (n *Neo4jClient) CreateSession(ctx context.Context) neo4j.SessionWithContext {
	return n.Driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
	})
}

func (n *Neo4jClient) CreateNode(ctx context.Context, label, name string, properties map[string]any) error {
	session := n.CreateSession(ctx)
	defer session.Close(ctx)

	cypher := fmt.Sprintf("CREATE (n:%s {name: $name}) SET n += $properties RETURN n", label)
	params := map[string]any{
		"name":       name,
		"properties": properties,
	}

	_, err := session.Run(ctx, cypher, params)
	return err
}

func (n *Neo4jClient) CreateRelationship(ctx context.Context, fromNode, toNode, relationshipType string, properties map[string]any) error {
	session := n.CreateSession(ctx)
	defer session.Close(ctx)

	cypher := `
		MATCH (a {name: $fromNode}), (b {name: $toNode})
		CREATE (a)-[r:%s $properties]->(b)
		RETURN r
	`
	cypher = fmt.Sprintf(cypher, relationshipType)
	
	params := map[string]any{
		"fromNode":   fromNode,
		"toNode":     toNode,
		"properties": properties,
	}

	_, err := session.Run(ctx, cypher, params)
	return err
}

func (n *Neo4jClient) FindShortestPath(ctx context.Context, startNode, endNode string) (*neo4j.EagerResult, error) {
	cypher := `
		MATCH (start {name: $startNode}), (end {name: $endNode})
		CALL apoc.algo.dijkstra(start, end, 'RELATES_TO', 'weight') YIELD path, weight
		RETURN path, weight
	`
	params := map[string]any{
		"startNode": startNode,
		"endNode":   endNode,
	}

	return n.ExecuteQuery(ctx, cypher, params)
}

func (n *Neo4jClient) GetAllNodes(ctx context.Context) (*neo4j.EagerResult, error) {
	cypher := "MATCH (n) RETURN n.name as name, labels(n) as labels, properties(n) as properties"
	return n.ExecuteQuery(ctx, cypher, map[string]any{})
}