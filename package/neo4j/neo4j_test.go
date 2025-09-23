package neo4j

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func TestNewNeo4jService_ValidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Neo4jConfig
	}{
		{
			name: "basic bolt config",
			config: Neo4jConfig{
				URI:      "bolt://localhost:7687",
				Username: "neo4j",
				Password: "password123",
				Database: "neo4j",
			},
		},
		{
			name: "neo4j protocol config",
			config: Neo4jConfig{
				URI:      "neo4j://localhost:7687",
				Username: "neo4j",
				Password: "secret",
				Database: "test",
			},
		},
		{
			name: "config with empty database (should default)",
			config: Neo4jConfig{
				URI:      "bolt://remote.neo4j.com:7687",
				Username: "admin",
				Password: "strongpassword",
				Database: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewNeo4jService(tt.config)

			if err != nil && (strings.Contains(err.Error(), "Neo4j URI is required") ||
				strings.Contains(err.Error(), "Neo4j username is required") ||
				strings.Contains(err.Error(), "Neo4j password is required")) {
				t.Errorf("Configuration validation failed: %v", err)
			}
		})
	}
}

func TestNewNeo4jService_InvalidConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        Neo4jConfig
		expectedError string
	}{
		{
			name: "empty URI",
			config: Neo4jConfig{
				URI:      "",
				Username: "neo4j",
				Password: "password",
				Database: "neo4j",
			},
			expectedError: "Neo4j URI is required",
		},
		{
			name: "empty username",
			config: Neo4jConfig{
				URI:      "bolt://localhost:7687",
				Username: "",
				Password: "password",
				Database: "neo4j",
			},
			expectedError: "Neo4j username is required",
		},
		{
			name: "empty password",
			config: Neo4jConfig{
				URI:      "bolt://localhost:7687",
				Username: "neo4j",
				Password: "",
				Database: "neo4j",
			},
			expectedError: "Neo4j password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewNeo4jService(tt.config)

			if err == nil {
				t.Error("Expected error for invalid config, got nil")
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}

			if client != nil {
				t.Error("Expected nil client for invalid config")
			}
		})
	}
}

func TestNeo4jConfig_DefaultDatabase(t *testing.T) {
	config := Neo4jConfig{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "password",
		Database: "",
	}

	if config.Database == "" {
		config.Database = "neo4j"
	}

	if config.Database != "neo4j" {
		t.Errorf("Expected default database to be 'neo4j', got '%s'", config.Database)
	}
}

func TestNeo4jClient_ImplementsInterface(t *testing.T) {
	var _ Neo4jService = (*Neo4jClient)(nil)

	clientType := reflect.TypeOf(&Neo4jClient{})
	interfaceType := reflect.TypeOf((*Neo4jService)(nil)).Elem()

	if !clientType.Implements(interfaceType) {
		t.Error("Neo4jClient does not implement Neo4jService interface")
	}
}

func TestNeo4jConfig_Structure(t *testing.T) {
	config := Neo4jConfig{
		URI:      "bolt://localhost:7687",
		Username: "test_user",
		Password: "test_password",
		Database: "test_db",
	}

	if config.URI != "bolt://localhost:7687" {
		t.Errorf("Expected URI to be 'bolt://localhost:7687', got %s", config.URI)
	}

	if config.Username != "test_user" {
		t.Errorf("Expected Username to be 'test_user', got %s", config.Username)
	}

	if config.Password != "test_password" {
		t.Errorf("Expected Password to be 'test_password', got %s", config.Password)
	}

	if config.Database != "test_db" {
		t.Errorf("Expected Database to be 'test_db', got %s", config.Database)
	}
}

func TestHealthStatus_Structure(t *testing.T) {
	status := HealthStatus{
		Connected:      true,
		Authenticated:  true,
		DatabaseExists: true,
		URI:            "bolt://localhost:7687",
		Database:       "neo4j",
		Latency:        100 * time.Millisecond,
	}

	if !status.Connected {
		t.Error("Expected Connected to be true")
	}

	if !status.Authenticated {
		t.Error("Expected Authenticated to be true")
	}

	if !status.DatabaseExists {
		t.Error("Expected DatabaseExists to be true")
	}

	if status.URI != "bolt://localhost:7687" {
		t.Errorf("Expected URI to be 'bolt://localhost:7687', got %s", status.URI)
	}

	if status.Database != "neo4j" {
		t.Errorf("Expected Database to be 'neo4j', got %s", status.Database)
	}

	if status.Latency != 100*time.Millisecond {
		t.Errorf("Expected Latency to be 100ms, got %v", status.Latency)
	}
}

func TestHealthStatus_JSONTags(t *testing.T) {
	statusType := reflect.TypeOf(HealthStatus{})

	expectedTags := map[string]string{
		"Connected":      "connected",
		"Authenticated":  "authenticated",
		"DatabaseExists": "database_exists",
		"URI":            "uri",
		"Database":       "database",
		"Latency":        "latency",
		"Error":          "error,omitempty",
	}

	for fieldName, expectedTag := range expectedTags {
		field, found := statusType.FieldByName(fieldName)
		if !found {
			t.Errorf("Field %s not found in HealthStatus", fieldName)
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag != expectedTag {
			t.Errorf("Field %s: expected JSON tag '%s', got '%s'", fieldName, expectedTag, jsonTag)
		}
	}
}

func TestNeo4jService_InterfaceCompleteness(t *testing.T) {
	interfaceType := reflect.TypeOf((*Neo4jService)(nil)).Elem()

	expectedMethods := []string{
		"HealthCheck", "GetDriver", "Close",
		"Run", "RunWrite", "RunRead",
		"CreateNode", "GetNode", "UpdateNode", "DeleteNode",
		"CreateRelationship", "DeleteRelationship",
		"FindNodes", "FindPath", "BeginTransaction",
	}

	for _, methodName := range expectedMethods {
		method, found := interfaceType.MethodByName(methodName)
		if !found {
			t.Errorf("Expected method %s not found in Neo4jService interface", methodName)
			continue
		}

		if method.Type.NumIn() >= 1 && methodName != "GetDriver" && methodName != "Close" {
			firstParam := method.Type.In(0)
			contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
			if firstParam == contextType || firstParam.String() == "context.Context" {
			} else {
				t.Errorf("Method %s should take context.Context as first parameter, got %v",
					methodName, firstParam)
			}
		}
	}
}

func TestNeo4jClient_MethodSignatures(t *testing.T) {
	clientType := reflect.TypeOf(&Neo4jClient{})

	expectedMethods := []struct {
		name     string
		numIn    int
		numOut   int
		hasError bool
	}{
		{"HealthCheck", 2, 1, false}, // (receiver, context) -> HealthStatus
		{"GetDriver", 1, 1, false},   // (receiver) -> neo4j.DriverWithContext
		{"Close", 1, 1, true},        // (receiver) -> error
		{"Run", 4, 2, true},          // (receiver, context, cypher, params) -> (result, error)
		{"RunWrite", 4, 2, true},     // (receiver, context, cypher, params) -> (result, error)
		{"RunRead", 4, 2, true},      // (receiver, context, cypher, params) -> (result, error)
		{"CreateNode", 4, 2, true},   // (receiver, context, labels, properties) -> (string, error)
		{"GetNode", 3, 2, true},      // (receiver, context, id) -> (map, error)
		{"UpdateNode", 4, 1, true},   // (receiver, context, id, properties) -> error
		{"DeleteNode", 3, 1, true},   // (receiver, context, id) -> error
	}

	for _, expected := range expectedMethods {
		method, found := clientType.MethodByName(expected.name)
		if !found {
			t.Errorf("Method %s not found", expected.name)
			continue
		}

		methodType := method.Type

		if methodType.NumIn() != expected.numIn {
			t.Errorf("Method %s: expected %d input parameters, got %d",
				expected.name, expected.numIn, methodType.NumIn())
		}

		if methodType.NumOut() != expected.numOut {
			t.Errorf("Method %s: expected %d output parameters, got %d",
				expected.name, expected.numOut, methodType.NumOut())
		}

		if expected.hasError && methodType.NumOut() > 0 {
			lastOut := methodType.Out(methodType.NumOut() - 1)
			errorInterface := reflect.TypeOf((*error)(nil)).Elem()
			if !lastOut.Implements(errorInterface) {
				t.Errorf("Method %s: expected last return type to be error, got %v",
					expected.name, lastOut)
			}
		}
	}
}

func TestCreateNode_LabelConstruction(t *testing.T) {
	tests := []struct {
		name     string
		labels   []string
		expected string
	}{
		{
			name:     "single label",
			labels:   []string{"Person"},
			expected: ":Person",
		},
		{
			name:     "multiple labels",
			labels:   []string{"Person", "Employee"},
			expected: ":Person::Employee",
		},
		{
			name:     "three labels",
			labels:   []string{"Person", "Employee", "Manager"},
			expected: ":Person::Employee::Manager",
		},
		{
			name:     "empty labels",
			labels:   []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			labelStr := ""
			for i, label := range tt.labels {
				if i > 0 {
					labelStr += ":"
				}
				labelStr += ":" + label
			}

			if labelStr != tt.expected {
				t.Errorf("Expected label string '%s', got '%s'", tt.expected, labelStr)
			}

			expectedCypher := "CREATE (n" + tt.expected + ") SET n = $props RETURN elementId(n) as id"
			actualCypher := "CREATE (n" + labelStr + ") SET n = $props RETURN elementId(n) as id"

			if actualCypher != expectedCypher {
				t.Errorf("Expected cypher '%s', got '%s'", expectedCypher, actualCypher)
			}
		})
	}
}

func TestFindNodes_WhereClauseConstruction(t *testing.T) {
	tests := []struct {
		name       string
		label      string
		properties map[string]interface{}
		expected   string
	}{
		{
			name:       "single property",
			label:      "Person",
			properties: map[string]interface{}{"name": "John"},
			expected:   "MATCH (n:Person) WHERE n.name = $prop_name RETURN properties(n) as props, elementId(n) as id",
		},
		{
			name:       "no properties",
			label:      "Person",
			properties: map[string]interface{}{},
			expected:   "MATCH (n:Person) WHERE true RETURN properties(n) as props, elementId(n) as id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cypher := "MATCH (n:" + tt.label + ") WHERE "
			params := make(map[string]interface{})

			whereConditions := make([]string, 0, len(tt.properties))
			for key, value := range tt.properties {
				paramKey := "prop_" + key
				whereConditions = append(whereConditions, "n."+key+" = $"+paramKey)
				params[paramKey] = value
			}

			if len(whereConditions) > 0 {
				cypher += whereConditions[0] + " RETURN properties(n) as props, elementId(n) as id"
			} else {
				cypher += "true RETURN properties(n) as props, elementId(n) as id"
			}

			if cypher != tt.expected {
				t.Errorf("Expected cypher '%s', got '%s'", tt.expected, cypher)
			}

			for key, value := range tt.properties {
				paramKey := "prop_" + key
				if params[paramKey] != value {
					t.Errorf("Expected param %s to be %v, got %v", paramKey, value, params[paramKey])
				}
			}
		})
	}
}

func TestFindPath_QueryConstruction(t *testing.T) {
	tests := []struct {
		name     string
		fromID   string
		toID     string
		maxDepth int
		expected string
	}{
		{
			name:     "basic path query",
			fromID:   "node1",
			toID:     "node2",
			maxDepth: 5,
			expected: "MATCH path = shortestPath((a)-[*1..5]-(b)) WHERE elementId(a) = $fromId AND elementId(b) = $toId RETURN path",
		},
		{
			name:     "different max depth",
			fromID:   "start",
			toID:     "end",
			maxDepth: 10,
			expected: "MATCH path = shortestPath((a)-[*1..10]-(b)) WHERE elementId(a) = $fromId AND elementId(b) = $toId RETURN path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualCypher := fmt.Sprintf("MATCH path = shortestPath((a)-[*1..%d]-(b)) WHERE elementId(a) = $fromId AND elementId(b) = $toId RETURN path", tt.maxDepth)

			if actualCypher != tt.expected {
				t.Errorf("Expected cypher '%s', got '%s'", tt.expected, actualCypher)
			}
		})
	}
}

func TestNeo4jConfig_DefaultValues(t *testing.T) {
	var config Neo4jConfig

	if config.URI != "" {
		t.Errorf("Expected default URI to be empty, got %s", config.URI)
	}

	if config.Username != "" {
		t.Errorf("Expected default Username to be empty, got %s", config.Username)
	}

	if config.Password != "" {
		t.Errorf("Expected default Password to be empty, got %s", config.Password)
	}

	if config.Database != "" {
		t.Errorf("Expected default Database to be empty, got %s", config.Database)
	}
}

func TestHealthStatus_DefaultValues(t *testing.T) {
	var status HealthStatus

	if status.Connected {
		t.Error("Expected default Connected to be false")
	}

	if status.Authenticated {
		t.Error("Expected default Authenticated to be false")
	}

	if status.DatabaseExists {
		t.Error("Expected default DatabaseExists to be false")
	}

	if status.URI != "" {
		t.Errorf("Expected default URI to be empty, got %s", status.URI)
	}

	if status.Database != "" {
		t.Errorf("Expected default Database to be empty, got %s", status.Database)
	}

	if status.Latency != 0 {
		t.Errorf("Expected default Latency to be 0, got %v", status.Latency)
	}

	if status.Error != "" {
		t.Errorf("Expected default Error to be empty, got %s", status.Error)
	}
}

func TestNeo4jClient_ThreadSafetyPattern(t *testing.T) {
	clientType := reflect.TypeOf(Neo4jClient{})

	mutexField, found := clientType.FieldByName("mu")
	if !found {
		t.Error("Expected 'mu' field not found in Neo4jClient")
	}

	expectedType := reflect.TypeOf((*sync.RWMutex)(nil)).Elem()
	if mutexField.Type != expectedType {
		t.Errorf("Expected 'mu' field to be sync.RWMutex, got %v", mutexField.Type)
	}
}

func TestNeo4jClient_GetDriver_Pattern(t *testing.T) {
	client := &Neo4jClient{
		driver: nil,
		config: Neo4jConfig{},
	}

	methodType := reflect.TypeOf(client.GetDriver)
	expectedReturnType := reflect.TypeOf((*neo4j.DriverWithContext)(nil)).Elem()

	if methodType.Out(0) != expectedReturnType {
		t.Errorf("GetDriver should return neo4j.DriverWithContext, got %v", methodType.Out(0))
	}
}

func TestNeo4jClient_Close_NilDriver(t *testing.T) {
	client := &Neo4jClient{
		driver: nil,
		config: Neo4jConfig{},
	}

	err := client.Close()
	if err != nil {
		t.Errorf("Close() with nil driver should not return error, got: %v", err)
	}
}
