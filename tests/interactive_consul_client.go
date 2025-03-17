package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	serverURL = "http://localhost:3000/api/v1"
	apiKey    = "test-api-key"
	debugMode = false // Set to true to show debug information
)

// Agent represents an AI agent in the registry
// Changed the JSON tag from "version" to "release" to match the server's metadata key.
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

func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n=== Consul AI Agent Registry Client ===")
		fmt.Println("1. List all agents")
		fmt.Println("2. View agent details")
		fmt.Println("3. Create Geography agent")
		fmt.Println("4. Create custom agent")
		fmt.Println("5. Delete an agent")
		fmt.Println("0. Exit")
		fmt.Print("Enter your choice: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "0":
			fmt.Println("Exiting program.")
			return
		case "1":
			agents, err := getAllAgents()
			if err != nil {
				displayError("Failed to list agents", err)
				continue
			}
			displayAgentList(agents)
		case "2":
			fmt.Print("Enter the name of the agent to view: ")
			name, _ := reader.ReadString('\n')
			name = strings.TrimSpace(name)

			agent, err := getAgent(name)
			if err != nil {
				displayError("Failed to get agent details", err)
				continue
			}

			displayAgentDetails(agent)
		case "3":
			fmt.Println("Attempting to register Geography agent...")
			if err := createGeographyAgent(); err != nil {
				displayError("Failed to create Geography agent", err)
				fmt.Println("\nPossible solutions:")
				fmt.Println("- Check if an agent with this name already exists")
				fmt.Println("- Delete the existing Geography agent first (option 5)")
			} else {
				displaySuccess("Geography agent created successfully!")
			}
		case "4":
			if err := createCustomAgent(reader); err != nil {
				displayError("Failed to create custom agent", err)
			}
		case "5":
			agents, err := getAllAgents()
			if err != nil {
				displayError("Failed to list agents", err)
				continue
			}
			displayAgentList(agents)

			fmt.Print("Enter the number of the agent to delete: ")
			numStr, _ := reader.ReadString('\n')
			numStr = strings.TrimSpace(numStr)
			num, err := strconv.Atoi(numStr)
			if err != nil || num < 1 || num > len(agents) {
				displayError("Invalid selection", nil)
				continue
			}

			agent := agents[num-1]
			agentName := agent["name"].(string)
			fmt.Printf("Attempting to delete agent '%s'...\n", agentName)
			if err := deleteAgent(agentName); err != nil {
				displayError("Failed to delete agent", err)
			} else {
				displaySuccess(fmt.Sprintf("Agent '%s' deleted successfully!", agentName))
			}
		default:
			fmt.Println("Invalid choice. Please try again.")
		}
	}
}
func getAllAgents() ([]map[string]interface{}, error) {
	req, err := http.NewRequest("GET", serverURL+"/agents", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("X-API-Key", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if debugMode {
		fmt.Println("DEBUG - Server response:", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, extractErrorFromResponse(resp.StatusCode, body)
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

	var agentMaps []map[string]interface{}

	if jsonType == "array" {
		// Direct array format
		var agents []interface{}
		if err := json.Unmarshal(body, &agents); err != nil {
			return nil, fmt.Errorf("failed to parse JSON array response: %w", err)
		}
		
		agentMaps = make([]map[string]interface{}, 0, len(agents))
		for _, agentData := range agents {
			agent, ok := agentData.(map[string]interface{})
			if !ok {
				continue
			}
			agentMaps = append(agentMaps, agent)
		}
	} else if jsonType == "object" {
		// Object with agents field
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON object response: %w", err)
		}

		agents, ok := result["agents"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("unexpected response format: agents field not found or not an array")
		}

		agentMaps = make([]map[string]interface{}, 0, len(agents))
		for _, agentData := range agents {
			agent, ok := agentData.(map[string]interface{})
			if !ok {
				continue
			}
			agentMaps = append(agentMaps, agent)
		}
	} else {
		return nil, fmt.Errorf("unexpected JSON format in response")
	}

	return agentMaps, nil
}

func ZgetAllAgents() ([]map[string]interface{}, error) {
	req, err := http.NewRequest("GET", serverURL+"/agents", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("X-API-Key", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if debugMode {
		fmt.Println("DEBUG - Server response:", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, extractErrorFromResponse(resp.StatusCode, body)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	agents, ok := result["agents"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format: agents field not found or not an array")
	}

	agentMaps := make([]map[string]interface{}, 0, len(agents))
	for _, agentData := range agents {
		agent, ok := agentData.(map[string]interface{})
		if !ok {
			continue
		}
		agentMaps = append(agentMaps, agent)
	}

	return agentMaps, nil
}

func getAgent(name string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", serverURL+"/agents/"+name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("X-API-Key", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if debugMode {
		fmt.Println("DEBUG - Server response:", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, extractErrorFromResponse(resp.StatusCode, body)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	agent, ok := result["agent"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format: agent field not found or not a map")
	}

	return agent, nil
}

func displayAgentList(agents []map[string]interface{}) {
	fmt.Printf("\nFound %d agents:\n", len(agents))
	fmt.Println("------------------------------------------------------------------------------------------------")
	fmt.Printf("%-3s | %-15s | %-20s | %-15s | %-30s\n", "#", "NAME", "DESCRIPTION", "RELEASE", "HOW TO USE")
	fmt.Println("------------------------------------------------------------------------------------------------")
	for i, agent := range agents {
		name := truncateString(fmt.Sprintf("%v", agent["name"]), 15)
		desc := truncateString(fmt.Sprintf("%v", agent["description"]), 20)
		
		// Handle optional release field
		releaseStr := "<not specified>"
		if release, ok := agent["release"]; ok && release != nil && release != "" {
			releaseStr = truncateString(fmt.Sprintf("%v", release), 15)
		}
		
		// Handle how to use field
		howToUseStr := "<not specified>"
		if howToUse, ok := agent["howtouse"]; ok && howToUse != nil && howToUse != "" {
			howToUseStr = truncateString(fmt.Sprintf("%v", howToUse), 30)
		}
		
		fmt.Printf("%-3d | %-15s | %-20s | %-15s | %-30s\n", 
			i+1, name, desc, releaseStr, howToUseStr)
	}
	fmt.Println("------------------------------------------------------------------------------------------------")
}
func DdisplayAgentList(agents []map[string]interface{}) {
	fmt.Printf("\nFound %d agents:\n", len(agents))
	fmt.Println("---------------------------------------------------")
	fmt.Printf("%-3s | %-20s | %s\n", "#", "NAME", "DESCRIPTION")
	fmt.Println("---------------------------------------------------")
	for i, agent := range agents {
		name := truncateString(fmt.Sprintf("%v", agent["name"]), 20)
		desc := truncateString(fmt.Sprintf("%v", agent["description"]), 50)
		fmt.Printf("%-3d | %-20s | %s\n", i+1, name, desc)
	}
	fmt.Println("---------------------------------------------------")
}

func displayAgentDetails(agent map[string]interface{}) {
	fmt.Println("\n=== Agent Details ===")
	fmt.Println("---------------------------------------------------")

	fmt.Printf("Name: %v\n", agent["name"])
	fmt.Printf("Description: %v\n", agent["description"])

	// Check for release field
	var releaseValue interface{}
	if val, ok := agent["release"]; ok && val != nil && val != "" {
		releaseValue = val
	}
	if releaseValue != nil {
		fmt.Printf("Release: %v\n", releaseValue)
	} else {
		fmt.Println("Release: <not returned by server>")
	}

	fmt.Println("\nAccess Information:")
	fmt.Printf("Base URL: %v\n", agent["baseurl"])

	var openAPIValue interface{}
	for _, key := range []string{"openapi", "openAPI", "OpenAPI"} {
		if val, ok := agent[key]; ok && val != nil {
			openAPIValue = val
			break
		}
	}
	if openAPIValue != nil {
		fmt.Printf("OpenAPI: %v\n", openAPIValue)
	} else {
		fmt.Println("OpenAPI: <not specified>")
	}

	fmt.Println("\nDocumentation:")
	if agent["howtouse"] != nil {
		fmt.Printf("How To Use: %v\n", agent["howtouse"])
	}

	fmt.Println("\nOperational Details:")
	if agent["expiration"] != nil {
		fmt.Printf("Expiration: %v\n", agent["expiration"])
	}

	fmt.Println("\nClassification:")
	fmt.Printf("Tags: %v\n", formatArray(agent["tags"]))

	fmt.Println("---------------------------------------------------")
}

func formatArray(value interface{}) string {
	if value == nil {
		return "<none>"
	}

	switch v := value.(type) {
	case []interface{}:
		items := make([]string, 0, len(v))
		for _, item := range v {
			items = append(items, fmt.Sprintf("%v", item))
		}
		return strings.Join(items, ", ")
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func createGeographyAgent() error {
	agent := Agent{
		Name:        "Geography",
		Description: "Provide information on historical places, countries, cities, or places of interest",
		Release:     "1.0.0",
		BaseURL:     "https://api.example.com/geography",
		OpenAPI:     "https://example.com/geography/openapi.json",
		HowToUse:    "Send GET requests to the API with location parameters",
		Expiration:  time.Now().AddDate(1, 0, 0),
		TTL:         300,
		Tags:        []string{"geography", "locations", "travel"},
	}

	return registerAgent(agent)
}

func createCustomAgent(reader *bufio.Reader) error {
	agent := Agent{}

	fmt.Println("\n=== Create Custom Agent ===")

	fmt.Print("Name: ")
	agent.Name = readString(reader)
	if agent.Name == "" {
		return fmt.Errorf("name is required")
	}

	fmt.Print("Description: ")
	agent.Description = readString(reader)
	if agent.Description == "" {
		return fmt.Errorf("description is required")
	}

	fmt.Print("Release: ")
	agent.Release = readString(reader)

	fmt.Print("Base URL: ")
	agent.BaseURL = readString(reader)
	if agent.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	fmt.Print("How To Use: ")
	agent.HowToUse = readString(reader)
	if agent.HowToUse == "" {
		return fmt.Errorf("how to use is required")
	}

	fmt.Print("OpenAPI URL (optional): ")
	agent.OpenAPI = readString(reader)

	fmt.Print("Tags (comma-separated): ")
	tags := readString(reader)
	if tags != "" {
		agent.Tags = strings.Split(tags, ",")
		for i, tag := range agent.Tags {
			agent.Tags[i] = strings.TrimSpace(tag)
		}
	}

	fmt.Print("TTL in seconds (e.g., 300 for 5 minutes): ")
	ttlStr := readString(reader)
	if ttlStr != "" {
		seconds, err := strconv.ParseInt(ttlStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid TTL format (must be a number of seconds): %v", err)
		}
		agent.TTL = seconds
	}

	agent.Expiration = time.Now().AddDate(1, 0, 0)

	fmt.Println("Attempting to register custom agent...")
	return registerAgent(agent)
}

func registerAgent(agent Agent) error {
	jsonData, err := json.Marshal(agent)
	if err != nil {
		return fmt.Errorf("failed to marshal agent to JSON: %w", err)
	}

	if debugMode {
		fmt.Println("DEBUG - Sending agent data:", string(jsonData))
	}

	req, err := http.NewRequest("POST", serverURL+"/agents", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("X-API-Key", apiKey)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if debugMode {
		fmt.Println("DEBUG - Server response:", string(body))
	}

	if resp.StatusCode != http.StatusCreated {
		return extractErrorFromResponse(resp.StatusCode, body)
	}

	// Check for release field in response
	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err == nil {
		if agentData, ok := responseData["agent"].(map[string]interface{}); ok {
			if _, hasRelease := agentData["release"]; !hasRelease && debugMode {
				fmt.Println("WARNING: Server response does not include the release field")
			}
		}
	}

	return nil
}

func deleteAgent(name string) error {
	req, err := http.NewRequest("DELETE", serverURL+"/agents/"+name, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("X-API-Key", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if debugMode {
		fmt.Println("DEBUG - Server response:", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return extractErrorFromResponse(resp.StatusCode, body)
	}

	return nil
}

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

func displayError(context string, err error) {
	fmt.Println("\n❌ ERROR:", context)
	if err != nil {
		fmt.Printf("   %v\n", err)
	}
}

func displaySuccess(message string) {
	fmt.Println("\n✅", message)
}

func readString(reader *bufio.Reader) string {
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
