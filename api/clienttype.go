package sharewoodapi
 

import (
	"time"
)

// Agent represents an AI agent in the registry
type Agent struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Release     string    `json:"release,omitempty"`
	BaseURL     string    `json:"baseurl"`
	OpenAPI     string    `json:"openapi,omitempty"`
	HowToUse    string    `json:"howtouse"`
	Expiration  time.Time `json:"expiration"`
	TTL         int64     `json:"ttl,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
}

// ErrorResponse represents the standard error response from the server
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details"`
}

// AgentList represents a list of agents returned by the API
type AgentList struct {
	Agents []Agent `json:"agents"`
}

// AgentResponse represents a single agent response
type AgentResponse struct {
	Agent Agent `json:"agent"`
}

// AgentRegistrationResponse represents the server response when registering an agent
type AgentRegistrationResponse struct {
	Agent   Agent  `json:"agent"`
	Message string `json:"message,omitempty"`
}

// ClientOptions contains configuration options for the ConsulClient
type ClientOptions struct {
	ServerURL string
	APIKey    string
	Timeout   time.Duration
	Debug     bool
}
