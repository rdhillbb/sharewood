package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/hashicorp/consul/api"
	"github.com/joho/godotenv"
	"github.com/rdhillbb/sharewood/sharewoodapi" // Import the sharewoodapi package
)

var consulClient *api.Client

func loadConfig() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found. Using environment variables.")
	}
}

func main() {
	loadConfig()
	var err error
	consulClient, err = initConsulClient()
	if err != nil {
		log.Fatalf("Error initializing Consul client: %v", err)
	}

	r := gin.Default()
	r.Use(corsMiddleware())
	
	// Public endpoints
	r.GET("/health", healthCheck)

	// API group secured with authentication middleware
	api := r.Group("/api/v1")
	api.Use(authMiddleware())
	{
		// Agent endpoints
		agents := api.Group("/agents")
		{
			agents.GET("", listAgents)
			agents.GET("/:name", getAgent)
			agents.POST("", authorize("admin", "agent-publisher"), registerAgent)
			agents.DELETE("/:name", authorize("admin", "agent-publisher"), unregisterAgent)
			agents.PUT("/:name/health", authorize("admin", "agent-publisher"), updateAgentHealth)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// Middleware functions
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For development/testing, you can bypass auth
		if os.Getenv("DEV_MODE") == "true" {
			c.Set("role", "admin")
			c.Next()
			return
		}

		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			role, valid := validateAPIKey(apiKey)
			if valid {
				c.Set("role", role)
				c.Next()
				return
			}
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, valid := validateJWT(tokenString)
			if valid {
				c.Set("user_id", claims.UserID)
				c.Set("role", claims.Role)
				c.Next()
				return
			}
		}

		c.JSON(http.StatusUnauthorized, sharewoodapi.ErrorResponse{
			Error:   "Authentication required",
			Details: "Provide a valid API key or Bearer token",
		})
		c.Abort()
	}
}

func authorize(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, sharewoodapi.ErrorResponse{
				Error: "Role information missing",
			})
			c.Abort()
			return
		}
		roleStr := role.(string)
		for _, allowedRole := range allowedRoles {
			if roleStr == allowedRole || roleStr == "admin" {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, sharewoodapi.ErrorResponse{
			Error: "Insufficient permissions",
		})
		c.Abort()
	}
}

// Authentication functions
func validateAPIKey(apiKey string) (string, bool) {
	// In production, implement secure API key validation
	if apiKey == "test-api-key" {
		return "agent-publisher", true
	}
	return "", false
}

type JWTClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.StandardClaims
}

func validateJWT(tokenString string) (*JWTClaims, bool) {
	secret := os.Getenv("JWT_SECRET")
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, false
	}
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, true
	}
	return nil, false
}

// Consul client initialization
func initConsulClient() (*api.Client, error) {
	config := api.DefaultConfig()
	consulAddr := os.Getenv("CONSUL_ADDR")
	if consulAddr != "" {
		config.Address = consulAddr
	}
	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %w", err)
	}
	return client, nil
}

// API endpoints
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Helper function to encode arrays to string for Consul metadata
func encodeArrayToString(arr []string) string {
	if len(arr) == 0 {
		return ""
	}
	return strings.Join(arr, ",")
}

// Helper function to decode string to array from Consul metadata
func decodeStringToArray(str string) []string {
	if str == "" {
		return []string{}
	}
	return strings.Split(str, ",")
}

// Helper function to check if an agent with the given name already exists
func agentExists(name string) (bool, error) {
	services, err := consulClient.Agent().Services()
	if err != nil {
		return false, fmt.Errorf("failed to check if agent exists: %w", err)
	}

	for _, service := range services {
		if service.Service == name {
			return true, nil
		}
	}
	
	return false, nil
}

// Agent Registration endpoint - Updated to use sharewoodapi.Agent
func registerAgent(c *gin.Context) {
	var agent sharewoodapi.Agent
	if err := c.ShouldBindJSON(&agent); err != nil {
		c.JSON(http.StatusBadRequest, sharewoodapi.ErrorResponse{
			Error:   "Invalid request body", 
			Details: err.Error(),
		})
		return
	}

	// Validate required fields
	if agent.Name == "" || agent.Description == "" || agent.BaseURL == "" || agent.HowToUse == "" {
		c.JSON(http.StatusBadRequest, sharewoodapi.ErrorResponse{
			Error:   "Missing required fields",
			Details: "name, description, baseurl, and howtouse are required",
		})
		return
	}
	
	// Check if an agent with this name already exists
	exists, err := agentExists(agent.Name)
	if err != nil {
		log.Printf("Error checking existing agents: %v", err)
		c.JSON(http.StatusInternalServerError, sharewoodapi.ErrorResponse{
			Error:   "Failed to check if agent already exists",
			Details: err.Error(),
		})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, sharewoodapi.ErrorResponse{
			Error:   "Agent already exists",
			Details: fmt.Sprintf("An agent with the name '%s' is already registered", agent.Name),
		})
		return
	}
	
	// Create metadata map with essential fields only
	metadata := map[string]string{
		"Description": agent.Description,
		"howtouse":    agent.HowToUse,
		"baseurl":     agent.BaseURL,
	}
	
	// Add expiration if present
	if !agent.Expiration.IsZero() {
		metadata["expiration"] = agent.Expiration.Format(time.RFC3339)
	}
	
	// Add release if present
	if agent.Release != "" {
		metadata["release"] = agent.Release
	}
	
	// Store OpenAPI spec
	if agent.OpenAPI != "" {
		metadata["openapi"] = agent.OpenAPI
	}
	
	// Store tags in metadata for easier retrieval
	if len(agent.Tags) > 0 {
		metadata["tags"] = encodeArrayToString(agent.Tags)
	}

	// Prepare service registration
	registration := &api.AgentServiceRegistration{
		Name: agent.Name,
		Tags: append([]string{"ai-agent"}, agent.Tags...),
		Meta: metadata,
	}

	// Handle TTL
	if agent.TTL > 0 {
		ttlDuration := time.Duration(agent.TTL) * time.Second
		registration.Check = &api.AgentServiceCheck{
			TTL:   ttlDuration.String(),
			Notes: "TTL for the AI agent service",
		}
	}

	if err := consulClient.Agent().ServiceRegister(registration); err != nil {
		log.Printf("Error registering agent: %v", err)
		c.JSON(http.StatusInternalServerError, sharewoodapi.ErrorResponse{
			Error:   "Failed to register agent",
			Details: err.Error(),
		})
		return
	}

	// Return the response in the expected format
	c.JSON(http.StatusCreated, sharewoodapi.AgentRegistrationResponse{
		Agent:   agent,
		Message: "Agent registered successfully",
	})
}

// List Agents endpoint - Updated to return format expected by client
func listAgents(c *gin.Context) {
	services, err := consulClient.Agent().Services()
	if err != nil {
		log.Printf("Error listing agents: %v", err)
		c.JSON(http.StatusInternalServerError, sharewoodapi.ErrorResponse{
			Error:   "Failed to list agents",
			Details: err.Error(),
		})
		return
	}

	agents := make([]sharewoodapi.Agent, 0)
	for _, service := range services {
		// Filter for AI agents only
		isAIAgent := false
		for _, tag := range service.Tags {
			if tag == "ai-agent" {
				isAIAgent = true
				break
			}
		}

		if isAIAgent {
			// Build sharewoodapi.Agent object
			agent := sharewoodapi.Agent{
				Name:        service.Service,
				Description: service.Meta["Description"],
				BaseURL:     service.Meta["baseurl"],
				HowToUse:    service.Meta["howtouse"],
			}
			
			// Add release if available
			if val, ok := service.Meta["release"]; ok && val != "" {
				agent.Release = val
			}
			
			// Add OpenAPI if available
			if val, ok := service.Meta["openapi"]; ok && val != "" {
				agent.OpenAPI = val
			}
			
			// Add expiration if available
			if val, ok := service.Meta["expiration"]; ok && val != "" {
				if t, err := time.Parse(time.RFC3339, val); err == nil {
					agent.Expiration = t
				}
			}
			
			// Add tags
			agent.Tags = make([]string, 0)
			// First add tags from meta if present
			if val, ok := service.Meta["tags"]; ok && val != "" {
				agent.Tags = append(agent.Tags, decodeStringToArray(val)...)
			}
			// Then add any tags from service that aren't the "ai-agent" tag
			for _, tag := range service.Tags {
				if tag != "ai-agent" {
					// Check if tag is already in the list
					found := false
					for _, existingTag := range agent.Tags {
						if existingTag == tag {
							found = true
							break
						}
					}
					if !found {
						agent.Tags = append(agent.Tags, tag)
					}
				}
			}
			
			agents = append(agents, agent)
		}
	}

	// Return the agents array directly to match client expectations
	c.JSON(http.StatusOK, agents)
}

// Get Agent endpoint - Updated to return format expected by client
func getAgent(c *gin.Context) {
	name := c.Param("name")
	
	// Check if the agent exists first
	exists, err := agentExists(name)
	if err != nil {
		log.Printf("Error checking agent existence: %v", err)
		c.JSON(http.StatusInternalServerError, sharewoodapi.ErrorResponse{
			Error:   "Failed to check agent existence",
			Details: err.Error(),
		})
		return
	}
	
	if !exists {
		c.JSON(http.StatusNotFound, sharewoodapi.ErrorResponse{
			Error: "Agent not found",
		})
		return
	}
	
	// If we get here, the agent exists, so we can fetch its details
	services, err := consulClient.Agent().Services()
	if err != nil {
		log.Printf("Error getting agent: %v", err)
		c.JSON(http.StatusInternalServerError, sharewoodapi.ErrorResponse{
			Error:   "Failed to get agent",
			Details: err.Error(),
		})
		return
	}

	for _, service := range services {
		if service.Service == name {
			// Check if it's an AI agent
			isAIAgent := false
			for _, tag := range service.Tags {
				if tag == "ai-agent" {
					isAIAgent = true
					break
				}
			}

			if isAIAgent {
				// Build agent with proper sharewoodapi.Agent type
				agent := sharewoodapi.Agent{
					Name:        service.Service,
					Description: service.Meta["Description"],
					HowToUse:    service.Meta["howtouse"],
					BaseURL:     service.Meta["baseurl"],
				}
				
				// Add release if it exists
				if val, ok := service.Meta["release"]; ok && val != "" {
					agent.Release = val
				}
				
				// Use consistent field name for OpenAPI
				if val, ok := service.Meta["openapi"]; ok && val != "" {
					agent.OpenAPI = val
				}
				
				// Add expiration if available
				if val, ok := service.Meta["expiration"]; ok && val != "" {
					if t, err := time.Parse(time.RFC3339, val); err == nil {
						agent.Expiration = t
					}
				}
				
				// Process tags
				agent.Tags = make([]string, 0)
				// First add tags from meta if present
				if val, ok := service.Meta["tags"]; ok && val != "" {
					agent.Tags = append(agent.Tags, decodeStringToArray(val)...)
				}
				// Then add any tags from service that aren't the "ai-agent" tag
				for _, tag := range service.Tags {
					if tag != "ai-agent" {
						// Check if tag is already in the list
						found := false
						for _, existingTag := range agent.Tags {
							if existingTag == tag {
								found = true
								break
							}
						}
						if !found {
							agent.Tags = append(agent.Tags, tag)
						}
					}
				}
				
				// Return in expected AgentResponse format
				c.JSON(http.StatusOK, sharewoodapi.AgentResponse{
					Agent: agent,
				})
				return
			}
		}
	}

	c.JSON(http.StatusNotFound, sharewoodapi.ErrorResponse{
		Error: "Agent not found",
	})
}

// Unregister Agent endpoint - Updated to use standard error responses
func unregisterAgent(c *gin.Context) {
	name := c.Param("name")
	
	// Verify the agent exists before attempting to deregister
	exists, err := agentExists(name)
	if err != nil {
		log.Printf("Error checking agent existence: %v", err)
		c.JSON(http.StatusInternalServerError, sharewoodapi.ErrorResponse{
			Error:   "Failed to check agent existence",
			Details: err.Error(),
		})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, sharewoodapi.ErrorResponse{
			Error:   "Agent not found",
			Details: fmt.Sprintf("No agent with the name '%s' was found", name),
		})
		return
	}

	if err := consulClient.Agent().ServiceDeregister(name); err != nil {
		log.Printf("Error unregistering agent: %v", err)
		c.JSON(http.StatusInternalServerError, sharewoodapi.ErrorResponse{
			Error:   "Failed to unregister agent",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent unregistered successfully"})
}

// Update Agent Health endpoint - Updated to use standard error responses
func updateAgentHealth(c *gin.Context) {
	name := c.Param("name")
	status := c.Query("status")

	// Validate status
	if status != "passing" && status != "warning" && status != "critical" {
		c.JSON(http.StatusBadRequest, sharewoodapi.ErrorResponse{
			Error: "Invalid status. Must be 'passing', 'warning', or 'critical'",
		})
		return
	}
	
	// Check if the agent exists
	exists, err := agentExists(name)
	if err != nil {
		log.Printf("Error checking agent existence: %v", err)
		c.JSON(http.StatusInternalServerError, sharewoodapi.ErrorResponse{
			Error:   "Failed to check agent existence",
			Details: err.Error(),
		})
		return
	}
	
	if !exists {
		c.JSON(http.StatusNotFound, sharewoodapi.ErrorResponse{
			Error: "Agent not found",
		})
		return
	}

	checkID := "service:" + name
	if err := consulClient.Agent().UpdateTTL(checkID, "", status); err != nil {
		log.Printf("Error updating agent health: %v", err)
		c.JSON(http.StatusInternalServerError, sharewoodapi.ErrorResponse{
			Error:   "Failed to update agent health",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent health updated successfully"})
}
