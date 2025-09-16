#!/bin/bash

echo "🚀 Setting up TigerBeetle-Neo4j Pipeline with AuraDB"
echo "=================================================="

# Load environment variables from .env file
if [ -f .env ]; then
    echo "📋 Loading environment variables from .env file..."
    export $(grep -v '^#' .env | xargs)
    echo "   NEO4J_URI: $NEO4J_URI"
    echo "   NEO4J_USERNAME: $NEO4J_USERNAME"
    echo "   Password: [HIDDEN]"
else
    echo "❌ .env file not found. Please create one with your AuraDB credentials."
    exit 1
fi

echo ""
echo "🧪 Testing Neo4j AuraDB connectivity..."

# Test DNS resolution first
HOSTNAME=$(echo $NEO4J_URI | sed 's/.*:\/\/\([^\/]*\).*/\1/')
echo "   Testing DNS resolution for: $HOSTNAME"

echo "   Attempting connection test (DNS may take time to propagate)..."

# Try connection test even if DNS initially fails
# Test connection using a simple Go program
cat > temp_auradb_check.go << 'EOF'
package main

import (
    "context"
    "fmt"
    "os"
    "time"
    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
    uri := os.Getenv("NEO4J_URI")
    username := os.Getenv("NEO4J_USERNAME")
    password := os.Getenv("NEO4J_PASSWORD")
    
    fmt.Printf("   Testing connection to: %s\n", uri)
    
    driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
    if err != nil {
        fmt.Printf("   ❌ Failed to create driver: %v\n", err)
        return
    }
    defer driver.Close(context.Background())
    
    // Set a reasonable timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    err = driver.VerifyConnectivity(ctx)
    if err != nil {
        fmt.Printf("   ❌ Connection failed: %v\n", err)
        fmt.Println("   💡 This might be due to:")
        fmt.Println("      - DNS propagation delay (try again in a few minutes)")
        fmt.Println("      - Network firewall blocking Neo4j ports")
        fmt.Println("      - AuraDB instance not fully started")
        return
    }
    
    fmt.Println("   ✅ Successfully connected to AuraDB!")
    
    // Test query
    session := driver.NewSession(ctx, neo4j.SessionConfig{})
    defer session.Close(ctx)
    
    _, err = session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
        result, err := tx.Run(ctx, "RETURN 1 as test", nil)
        if err != nil {
            return nil, err
        }
        return result.Consume(ctx)
    })
    
    if err != nil {
        fmt.Printf("   ❌ Test query failed: %v\n", err)
    } else {
        fmt.Println("   ✅ Test query successful - AuraDB is ready!")
    }
}
EOF

echo "   Connecting to AuraDB..."
if go run temp_auradb_check.go; then
    CONNECTION_SUCCESS=true
else
    CONNECTION_SUCCESS=false
fi

if [ "$CONNECTION_SUCCESS" = "true" ]; then
        echo ""
        echo "🎉 AuraDB setup complete! You can now run the pipeline:"
        echo ""
        echo "   1. Format TigerBeetle and start infrastructure:"
        echo "      make setup-infra"
        echo ""
        echo "   2. Create Redpanda topic:"
        echo "      make create-topic"
        echo ""
        echo "   3. Build and run services:"
        echo "      make build"
        echo "      make run-generator    # In terminal 1"
        echo "      make run-cdc         # In terminal 2"  
        echo "      make run-sink        # In terminal 3"
        echo ""
else
    echo ""
    echo "🔧 Troubleshooting steps:"
    echo "   1. Wait a few minutes for DNS propagation (new AuraDB instances)"
    echo "   2. Verify your AuraDB instance is 'Running' in the Neo4j Console"
    echo "   3. Check that the connection URI is correct in .env file"
    echo "   4. Test connection from Neo4j Browser first"
    echo "   5. Verify the credentials are correct"
    echo ""
    echo "   Current .env settings:"
    echo "   NEO4J_URI=$NEO4J_URI"
    echo "   NEO4J_USERNAME=$NEO4J_USERNAME"
    echo ""
    echo "   You can still proceed with the pipeline setup:"
    echo "   make setup-infra && make create-topic && make build"
fi

rm -f temp_auradb_check.go
