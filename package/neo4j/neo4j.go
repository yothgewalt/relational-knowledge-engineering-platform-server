package neo4j

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	neo4jConfig "github.com/neo4j/neo4j-go-driver/v5/neo4j/config"
)

type Neo4jConfig struct {
	URI      string
	Username string
	Password string
	Database string
}

type HealthStatus struct {
	Connected      bool          `json:"connected"`
	Authenticated  bool          `json:"authenticated"`
	DatabaseExists bool          `json:"database_exists"`
	URI            string        `json:"uri"`
	Database       string        `json:"database"`
	Latency        time.Duration `json:"latency"`
	Error          string        `json:"error,omitempty"`
}

type Neo4jService interface {
	HealthCheck(ctx context.Context) HealthStatus
	GetDriver() neo4j.DriverWithContext
	Close() error

	Run(ctx context.Context, cypher string, params map[string]interface{}) (neo4j.ResultWithContext, error)
	RunWrite(ctx context.Context, cypher string, params map[string]interface{}) (neo4j.ResultWithContext, error)
	RunRead(ctx context.Context, cypher string, params map[string]interface{}) (neo4j.ResultWithContext, error)

	CreateNode(ctx context.Context, labels []string, properties map[string]interface{}) (string, error)
	GetNode(ctx context.Context, id string) (map[string]interface{}, error)
	UpdateNode(ctx context.Context, id string, properties map[string]interface{}) error
	DeleteNode(ctx context.Context, id string) error

	CreateRelationship(ctx context.Context, fromID, toID, relType string, properties map[string]interface{}) (string, error)
	DeleteRelationship(ctx context.Context, id string) error

	FindNodes(ctx context.Context, label string, properties map[string]interface{}) ([]map[string]interface{}, error)
	FindPath(ctx context.Context, fromID, toID string, maxDepth int) ([]map[string]interface{}, error)

	BeginTransaction(ctx context.Context) (neo4j.ExplicitTransaction, error)
}

type Neo4jClient struct {
	driver neo4j.DriverWithContext
	config Neo4jConfig
	mu     sync.RWMutex
}

func NewNeo4jService(config Neo4jConfig) (*Neo4jClient, error) {
	if config.URI == "" {
		return nil, fmt.Errorf("Neo4j URI is required")
	}

	if config.Username == "" {
		return nil, fmt.Errorf("Neo4j username is required")
	}

	if config.Password == "" {
		return nil, fmt.Errorf("Neo4j password is required")
	}

	if config.Database == "" {
		config.Database = "neo4j"
	}

	auth := neo4j.BasicAuth(config.Username, config.Password, "")

	driver, err := neo4j.NewDriverWithContext(config.URI, auth, func(config *neo4jConfig.Config) {
		config.MaxConnectionPoolSize = 50
		config.MaxConnectionLifetime = 5 * time.Minute
		config.ConnectionAcquisitionTimeout = 2 * time.Minute
		config.SocketConnectTimeout = 5 * time.Second
		config.SocketKeepalive = true
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	return &Neo4jClient{
		driver: driver,
		config: config,
	}, nil
}

func (n *Neo4jClient) HealthCheck(ctx context.Context) HealthStatus {
	n.mu.RLock()
	defer n.mu.RUnlock()

	start := time.Now()

	status := HealthStatus{
		URI:      n.config.URI,
		Database: n.config.Database,
	}

	err := n.driver.VerifyConnectivity(ctx)
	if err != nil {
		status.Connected = false
		status.Authenticated = false
		status.DatabaseExists = false
		status.Error = fmt.Sprintf("connectivity failed: %v", err)
		status.Latency = time.Since(start)
		return status
	}

	status.Connected = true

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	_, err = session.Run(ctx, "RETURN 1 as test", nil)
	if err != nil {
		status.Authenticated = false
		status.DatabaseExists = false
		status.Error = fmt.Sprintf("authentication or database access failed: %v", err)
		status.Latency = time.Since(start)
		return status
	}

	status.Authenticated = true

	_, err = session.Run(ctx, "SHOW CONSTRAINTS", nil)
	if err != nil {
		status.DatabaseExists = false
		status.Error = fmt.Sprintf("database access failed: %v", err)
		status.Latency = time.Since(start)
		return status
	}

	status.DatabaseExists = true
	status.Latency = time.Since(start)

	return status
}

func (n *Neo4jClient) GetDriver() neo4j.DriverWithContext {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.driver
}

func (n *Neo4jClient) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.driver != nil {
		return n.driver.Close(context.Background())
	}
	return nil
}

func (n *Neo4jClient) Run(ctx context.Context, cypher string, params map[string]interface{}) (neo4j.ResultWithContext, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	return session.Run(ctx, cypher, params)
}

func (n *Neo4jClient) RunWrite(ctx context.Context, cypher string, params map[string]interface{}) (neo4j.ResultWithContext, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	return session.Run(ctx, cypher, params)
}

func (n *Neo4jClient) RunRead(ctx context.Context, cypher string, params map[string]interface{}) (neo4j.ResultWithContext, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	return session.Run(ctx, cypher, params)
}

// Node operations
func (n *Neo4jClient) CreateNode(ctx context.Context, labels []string, properties map[string]interface{}) (string, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	labelStr := ""
	for i, label := range labels {
		if i > 0 {
			labelStr += ":"
		}
		labelStr += ":" + label
	}

	cypher := fmt.Sprintf("CREATE (n%s) SET n = $props RETURN elementId(n) as id", labelStr)

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, cypher, map[string]interface{}{"props": properties})
	if err != nil {
		return "", fmt.Errorf("failed to create node: %w", err)
	}

	if result.Next(ctx) {
		record := result.Record()
		if id, ok := record.Get("id"); ok {
			return id.(string), nil
		}
	}

	return "", fmt.Errorf("failed to get created node ID")
}

func (n *Neo4jClient) GetNode(ctx context.Context, id string) (map[string]interface{}, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	cypher := "MATCH (n) WHERE elementId(n) = $id RETURN properties(n) as props"

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, cypher, map[string]interface{}{"id": id})
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	if result.Next(ctx) {
		record := result.Record()
		if props, ok := record.Get("props"); ok {
			if propsMap, ok := props.(map[string]interface{}); ok {
				return propsMap, nil
			}
		}
	}

	return nil, fmt.Errorf("node not found")
}

func (n *Neo4jClient) UpdateNode(ctx context.Context, id string, properties map[string]interface{}) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	cypher := "MATCH (n) WHERE elementId(n) = $id SET n += $props RETURN n"

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, cypher, map[string]interface{}{
		"id":    id,
		"props": properties,
	})
	if err != nil {
		return fmt.Errorf("failed to update node: %w", err)
	}

	if !result.Next(ctx) {
		return fmt.Errorf("node not found")
	}

	return nil
}

func (n *Neo4jClient) DeleteNode(ctx context.Context, id string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	cypher := "MATCH (n) WHERE elementId(n) = $id DETACH DELETE n"

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	_, err := session.Run(ctx, cypher, map[string]interface{}{"id": id})
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return nil
}

// Relationship operations
func (n *Neo4jClient) CreateRelationship(ctx context.Context, fromID, toID, relType string, properties map[string]interface{}) (string, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	cypher := fmt.Sprintf("MATCH (a), (b) WHERE elementId(a) = $fromId AND elementId(b) = $toId CREATE (a)-[r:%s]->(b) SET r = $props RETURN elementId(r) as id", relType)

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, cypher, map[string]interface{}{
		"fromId": fromID,
		"toId":   toID,
		"props":  properties,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create relationship: %w", err)
	}

	if result.Next(ctx) {
		record := result.Record()
		if id, ok := record.Get("id"); ok {
			return id.(string), nil
		}
	}

	return "", fmt.Errorf("failed to get created relationship ID")
}

func (n *Neo4jClient) DeleteRelationship(ctx context.Context, id string) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	cypher := "MATCH ()-[r]-() WHERE elementId(r) = $id DELETE r"

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
		AccessMode:   neo4j.AccessModeWrite,
	})
	defer session.Close(ctx)

	_, err := session.Run(ctx, cypher, map[string]interface{}{"id": id})
	if err != nil {
		return fmt.Errorf("failed to delete relationship: %w", err)
	}

	return nil
}

func (n *Neo4jClient) FindNodes(ctx context.Context, label string, properties map[string]interface{}) ([]map[string]interface{}, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	cypher := fmt.Sprintf("MATCH (n:%s) WHERE ", label)
	params := make(map[string]interface{})

	whereConditions := make([]string, 0, len(properties))
	for key, value := range properties {
		paramKey := "prop_" + key
		whereConditions = append(whereConditions, fmt.Sprintf("n.%s = $%s", key, paramKey))
		params[paramKey] = value
	}

	if len(whereConditions) > 0 {
		cypher += fmt.Sprintf("%s RETURN properties(n) as props, elementId(n) as id", whereConditions[0])
		for i := 1; i < len(whereConditions); i++ {
			cypher = cypher[:len(cypher)-len(" RETURN properties(n) as props, elementId(n) as id")] +
				fmt.Sprintf(" AND %s RETURN properties(n) as props, elementId(n) as id", whereConditions[i])
		}
	} else {
		cypher += "true RETURN properties(n) as props, elementId(n) as id"
	}

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find nodes: %w", err)
	}

	var nodes []map[string]interface{}
	for result.Next(ctx) {
		record := result.Record()
		nodeData := make(map[string]interface{})

		if id, ok := record.Get("id"); ok {
			nodeData["_id"] = id
		}

		if props, ok := record.Get("props"); ok {
			if propsMap, ok := props.(map[string]interface{}); ok {
				for k, v := range propsMap {
					nodeData[k] = v
				}
			}
		}

		nodes = append(nodes, nodeData)
	}

	return nodes, nil
}

func (n *Neo4jClient) FindPath(ctx context.Context, fromID, toID string, maxDepth int) ([]map[string]interface{}, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	cypher := fmt.Sprintf("MATCH path = shortestPath((a)-[*1..%d]-(b)) WHERE elementId(a) = $fromId AND elementId(b) = $toId RETURN path", maxDepth)

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
		AccessMode:   neo4j.AccessModeRead,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, cypher, map[string]interface{}{
		"fromId": fromID,
		"toId":   toID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find path: %w", err)
	}

	var paths []map[string]interface{}
	for result.Next(ctx) {
		record := result.Record()
		if path, ok := record.Get("path"); ok {
			pathData := map[string]interface{}{
				"path": path,
			}
			paths = append(paths, pathData)
		}
	}

	return paths, nil
}

func (n *Neo4jClient) BeginTransaction(ctx context.Context) (neo4j.ExplicitTransaction, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: n.config.Database,
		AccessMode:   neo4j.AccessModeWrite,
	})

	return session.BeginTransaction(ctx)
}
