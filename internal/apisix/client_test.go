package apisix

import (
	"encoding/json"
	"testing"
)

func TestClient_buildURL(t *testing.T) {
	client := NewClient("http://localhost:9180/apisix/admin", "test123", 30)

	tests := []struct {
		name         string
		resourceType string
		id           string
		want         string
	}{
		{
			name:         "upstream with id",
			resourceType: "upstreams",
			id:           "1",
			want:         "http://localhost:9180/apisix/admin/upstreams/1",
		},
		{
			name:         "upstream without id",
			resourceType: "upstreams",
			id:           "",
			want:         "http://localhost:9180/apisix/admin/upstreams",
		},
		{
			name:         "route with id",
			resourceType: "routes",
			id:           "test-route",
			want:         "http://localhost:9180/apisix/admin/routes/test-route",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.buildURL(tt.resourceType, tt.id)
			if got != tt.want {
				t.Errorf("buildURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPISIXError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  APISIXError
		want string
	}{
		{
			name: "with ErrorMsg",
			err:  APISIXError{ErrorMsg: "resource not found"},
			want: "resource not found",
		},
		{
			name: "with Message",
			err:  APISIXError{Message: "invalid configuration"},
			want: "invalid configuration",
		},
		{
			name: "empty error",
			err:  APISIXError{},
			want: "unknown APISIX error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpstreamNode_MarshalJSON(t *testing.T) {
	node := UpstreamNode{
		Host:     "127.0.0.1",
		Port:     8080,
		Weight:   100,
		Priority: 0,
	}

	// Test that node serializes correctly
	data, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("Failed to marshal node: %v", err)
	}

	expected := `{"host":"127.0.0.1","port":8080,"weight":100}`
	if string(data) != expected {
		t.Errorf("MarshalJSON() = %s, want %s", string(data), expected)
	}
}
