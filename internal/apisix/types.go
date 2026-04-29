package apisix

// APISIXResponse represents the standard response from APISIX Admin API
type APISIXResponse struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// APISIXListResponse represents a list response from APISIX Admin API
type APISIXListResponse struct {
	List  []APISIXResponse `json:"list"`
	Total int              `json:"total"`
}

// UpstreamNode represents a single node in an upstream
type UpstreamNode struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Weight   int    `json:"weight"`
	Priority int    `json:"priority,omitempty"`
}

// UpstreamTimeout represents timeout configuration for an upstream
type UpstreamTimeout struct {
	Connect int `json:"connect,omitempty"`
	Send    int `json:"send,omitempty"`
	Read    int `json:"read,omitempty"`
}

// Upstream represents an APISIX upstream configuration
type Upstream struct {
	ID            string                 `json:"id,omitempty"`
	Name          string                 `json:"name,omitempty"`
	Desc          string                 `json:"desc,omitempty"`
	Type          string                 `json:"type"`
	Nodes         []UpstreamNode         `json:"nodes,omitempty"`
	HealthCheck   map[string]interface{} `json:"checks,omitempty"`
	Timeout       *UpstreamTimeout       `json:"timeout,omitempty"`
	Retries       int                    `json:"retries,omitempty"`
	RetryTimeout  int                    `json:"retry_timeout,omitempty"`
	Scheme        string                 `json:"scheme,omitempty"`
	Labels        map[string]string      `json:"labels,omitempty"`
	ServiceName   string                 `json:"service_name,omitempty"`
	DiscoveryType string                 `json:"discovery_type,omitempty"`
	DiscoveryArgs map[string]string      `json:"discovery_args,omitempty"`
	HashOn        string                 `json:"hash_on,omitempty"`
	Key           string                 `json:"key,omitempty"`
	PassHost      string                 `json:"pass_host,omitempty"`
	UpstreamHost  string                 `json:"upstream_host,omitempty"`
	KeepalivePool map[string]interface{} `json:"keepalive_pool,omitempty"`
	TLS           map[string]interface{} `json:"tls,omitempty"`
}

// APISIXError represents an error response from APISIX
type APISIXError struct {
	Code     int    `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
	ErrorMsg string `json:"error_msg,omitempty"`
	Data     string `json:"data,omitempty"`
}

// Error implements the error interface for APISIXError
func (e *APISIXError) Error() string {
	if e.ErrorMsg != "" {
		return e.ErrorMsg
	}
	if e.Message != "" {
		return e.Message
	}
	return "unknown APISIX error"
}
