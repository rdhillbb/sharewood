package sharwoodapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// ConsulClient is the client for interacting with the Consul AI Agent Registry API
type ConsulClient struct {
	serverURL string
	apiKey    string
	client    *http.Client
	debug     bool
}

// DefaultOptions returns the default client options
func DefaultOptions() ClientOptions {
	return ClientOptions{
		ServerURL: "http://localhost:3000/api/v1",
		APIKey:    "test-api-key",
		Timeout:   10 * time.Second,
		Debug:     false,
	}
}

// NewClient creates a new ConsulClient with the specified options
func NewClient(options ClientOptions) *ConsulClient {
	return &ConsulClient{
		serverURL: options.ServerURL,
		apiKey:    options.APIKey,
		client: &http.Client{
			Timeout: options.Timeout,
		},
		debug: options.Debug,
	}
}

// ListAgents retrieves all agents from the registry
func (c *ConsulClient) ListAgents() ([]Agent, error) {
	req, err := http.NewRequest("GET", c.serverURL+"/agents", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("X-API-Key", c.apiKey)

	body, statusCode, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, extractErrorFromResponse(statusCode, body)
	}

	// Check the first non-whitespace character to determine the JSON type
	jsonType := "unknown"
	for i := 0; i < len(body); i++ {
		if body[i] == ' ' || body[i] == '\n' || body[i] == '\r' || body[i] == '\t' {
			continue
		}
		if body[i] == '[' {
			jsonType = "array"
		} else if body[i] == '{' {
			jsonType = "object"
		}
		break
	}

	var agents []Agent

	if jsonType == "array" {
		// Direct array format
		var agentArray []Agent
		if err := json.Unmarshal(body, &agentArray); err != nil {
			return nil, fmt.Errorf("failed to parse JSON array response: %w", err)
		}
		agents = agentArray
	} else if jsonType == "object" {
		// Object with agents field
		var result AgentList
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON object response: %w", err)
		}
		agents = result.Agents
	} else {
		return nil, fmt.Errorf("unexpected JSON format in response")
	}

	return agents, nil
}

// GetAgent retrieves a specific agent by name
func (c *ConsulClient) GetAgent(name string) (*Agent, error) {
	if name == "" {
		return nil, fmt.Errorf("agent name cannot be empty")
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/agents/%s", c.serverURL, name), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("X-API-Key", c.apiKey)

	body, statusCode, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, extractErrorFromResponse(statusCode, body)
	}

	var result AgentResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &result.Agent, nil
}

// RegisterAgent registers a new agent with the registry
func (c *ConsulClient) RegisterAgent(agent Agent) (*Agent, error) {
	// Validate required fields
	if agent.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if agent.Description == "" {
		return nil, fmt.Errorf("agent description is required")
	}
	if agent.BaseURL == "" {
		return nil, fmt.Errorf("agent base URL is required")
	}
	if agent.HowToUse == "" {
		return nil, fmt.Errorf("agent how-to-use is required")
	}

	jsonData, err := json.Marshal(agent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agent to JSON: %w", err)
	}

	if c.debug {
		log.Printf("DEBUG - Sending agent data: %s", string(jsonData))
	}

	req, err := http.NewRequest("POST", c.serverURL+"/agents", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("X-API-Key", c.apiKey)
	req.Header.Add("Content-Type", "application/json")

	body, statusCode, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusCreated {
		return nil, extractErrorFromResponse(statusCode, body)
	}

	var response AgentRegistrationResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response.Agent, nil
}

// DeregisterAgent removes an agent from the registry
func (c *ConsulClient) DeregisterAgent(name string) error {
	if name == "" {
		return fmt.Errorf("agent name cannot be empty")
	}

	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/agents/%s", c.serverURL, name), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("X-API-Key", c.apiKey)

	body, statusCode, err := c.doRequest(req)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK {
		return extractErrorFromResponse(statusCode, body)
	}

	return nil
}

// doRequest performs an HTTP request and returns the response body and status code
func (c *ConsulClient) doRequest(req *http.Request) ([]byte, int, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.debug {
		log.Printf("DEBUG - Server response: %s", string(body))
	}

	return body, resp.StatusCode, nil
}

// extractErrorFromResponse parses error information from the response body
func extractErrorFromResponse(statusCode int, body []byte) error {
	// Try to parse as JSON error response
	var errorResp ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err == nil && (errorResp.Error != "" || errorResp.Details != "") {
		if errorResp.Details != "" {
			return fmt.Errorf("%s: %s (Status: %d)", errorResp.Error, errorResp.Details, statusCode)
		}
		return fmt.Errorf("%s (Status: %d)", errorResp.Error, statusCode)
	}
	
	// Fallback for non-standard error responses
	return fmt.Errorf("request failed with status %d: %s", statusCode, string(body))
}
