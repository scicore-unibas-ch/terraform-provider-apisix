package apisix

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

// TestClient_Integration tests the APISIX client against a real APISIX instance
// Run with: TF_ACC=1 go test ./internal/apisix -run TestClient_Integration -v
func TestClient_Integration(t *testing.T) {
	// Skip if not running acceptance tests
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set, skipping integration test")
	}

	baseURL := os.Getenv("APISIX_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:9180/apisix/admin"
	}

	adminKey := os.Getenv("APISIX_ADMIN_KEY")
	if adminKey == "" {
		adminKey = "test123456789"
	}

	client := NewClient(baseURL, adminKey, 30)
	ctx := context.Background()

	t.Run("CreateUpstream", func(t *testing.T) {
		upstream := Upstream{
			Name: "test-upstream",
			Type: "roundrobin",
			Nodes: []UpstreamNode{
				{Host: "127.0.0.1", Port: 8080, Weight: 100},
			},
		}

		// Create upstream
		err := client.Create(ctx, "upstreams", "test-1", upstream)
		if err != nil {
			t.Fatalf("Failed to create upstream: %v", err)
		}

		// Read upstream
		data, err := client.Read(ctx, "upstreams", "test-1")
		if err != nil {
			t.Fatalf("Failed to read upstream: %v", err)
		}

		// Verify response
		var resp APISIXResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Delete upstream
		err = client.Delete(ctx, "upstreams", "test-1", false)
		if err != nil {
			t.Fatalf("Failed to delete upstream: %v", err)
		}

		t.Log("✓ Create, Read, Delete upstream successful")
	})

	t.Run("UpdateUpstream", func(t *testing.T) {
		// First create
		upstream := Upstream{
			Name: "test-upstream-update",
			Type: "roundrobin",
			Nodes: []UpstreamNode{
				{Host: "127.0.0.1", Port: 8080, Weight: 100},
			},
		}
		err := client.Create(ctx, "upstreams", "test-2", upstream)
		if err != nil {
			t.Fatalf("Failed to create upstream: %v", err)
		}

		// Update with new node
		update := map[string]interface{}{
			"nodes": []UpstreamNode{
				{Host: "127.0.0.1", Port: 8080, Weight: 100},
				{Host: "127.0.0.1", Port: 8081, Weight: 50},
			},
		}
		err = client.Update(ctx, "upstreams", "test-2", update)
		if err != nil {
			t.Fatalf("Failed to update upstream: %v", err)
		}

		// Verify update
		data, err := client.Read(ctx, "upstreams", "test-2")
		if err != nil {
			t.Fatalf("Failed to read upstream: %v", err)
		}

		var resp APISIXResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Cleanup
		client.Delete(ctx, "upstreams", "test-2", false)

		t.Log("✓ Update upstream successful")
	})

	t.Run("ListUpstreams", func(t *testing.T) {
		data, err := client.List(ctx, "upstreams")
		if err != nil {
			t.Fatalf("Failed to list upstreams: %v", err)
		}

		var resp APISIXListResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			t.Fatalf("Failed to unmarshal list response: %v", err)
		}

		t.Logf("✓ List upstreams successful (total: %d)", resp.Total)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Try to read non-existent upstream
		_, err := client.Read(ctx, "upstreams", "non-existent")
		if err == nil {
			t.Fatal("Expected error for non-existent upstream, got nil")
		}

		t.Logf("✓ Error handling works: %v", err)
	})
}
