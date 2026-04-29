package resources

import (
	"fmt"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/apisix"
)

// getClient retrieves the APISIX client from provider meta
func getClient(meta interface{}) (*apisix.Client, error) {
	if meta == nil {
		return nil, fmt.Errorf("provider meta is nil")
	}
	client, ok := meta.(*apisix.Client)
	if !ok {
		return nil, fmt.Errorf("failed to convert provider meta to APISIX client")
	}
	return client, nil
}

// expandStringList converts interface list to string list
func expandStringList(list []interface{}) []string {
	result := make([]string, 0, len(list))
	for _, v := range list {
		if v != nil {
			result = append(result, v.(string))
		}
	}
	return result
}

// expandInlineUpstream converts inline upstream config
func expandInlineUpstream(upstream []interface{}) map[string]interface{} {
	if len(upstream) == 0 || upstream[0] == nil {
		return nil
	}
	u := upstream[0].(map[string]interface{})
	result := make(map[string]interface{})
	if v, ok := u["type"].(string); ok {
		result["type"] = v
	}
	if v, ok := u["nodes"]; ok {
		result["nodes"] = expandInlineNodes(v.([]interface{}))
	}
	return result
}

// expandInlineNodes converts inline nodes config
func expandInlineNodes(nodes []interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(nodes))
	for _, node := range nodes {
		if n, ok := node.(map[string]interface{}); ok {
			nodeMap := make(map[string]interface{})
			if v, ok := n["host"].(string); ok {
				nodeMap["host"] = v
			}
			if v, ok := n["port"].(int); ok {
				nodeMap["port"] = v
			}
			if v, ok := n["weight"].(int); ok {
				nodeMap["weight"] = v
			}
			result = append(result, nodeMap)
		}
	}
	return result
}
