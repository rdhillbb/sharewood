package main

import (
	"fmt"
	"log"
	"strings"
	"time"
       shwood "github.com/rdhillbb/sharewood/sharewoodapi"
)

func main() {
	// Initialize client with default options
	options := shwood.DefaultOptions()
	// Disable debug mode for cleaner output
	options.Debug = false
	
	client := shwood.NewClient(options)

	// Step 1: List all agents
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║                   LISTING ALL AGENTS                     ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	
	agents, err := client.ListAgents()
	if err != nil {
		log.Fatalf("Failed to list agents: %v", err)
	}
	
	fmt.Printf("Found %d agents\n", len(agents))
	fmt.Println("┌────┬────────────────────┬──────────────────────────────────────────┐")
	fmt.Println("│ #  │ Name               │ Description                              │")
	fmt.Println("├────┼────────────────────┼──────────────────────────────────────────┤")
	
	for i, agent := range agents {
		name := padOrTruncate(agent.Name, 18)
		desc := padOrTruncate(agent.Description, 40)
		fmt.Printf("│ %-2d │ %-18s │ %-40s │\n", i+1, name, desc)
	}
	
	fmt.Println("└────┴────────────────────┴──────────────────────────────────────────┘")

	// Step 2: Get detailed information for each agent
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║              DETAILED AGENT INFORMATION                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	
	for i, agent := range agents {
		fmt.Printf("\n[Agent %d/%d] %s\n", i+1, len(agents), agent.Name)
		fmt.Println("┌──────────────────────────────────────────────────────────────┐")
		
		agentDetails, err := client.GetAgent(agent.Name)
		if err != nil {
			fmt.Printf("│ ERROR: Failed to get agent details: %v\n", err)
			fmt.Println("└──────────────────────────────────────────────────────────────┘")
			continue
		}
		
		fmt.Printf("│ Name:        %-48s │\n", agentDetails.Name)
		fmt.Printf("│ Description: %-48s │\n", truncateString(agentDetails.Description, 48))
		fmt.Printf("│ Base URL:    %-48s │\n", truncateString(agentDetails.BaseURL, 48))
		
		if agentDetails.Release != "" {
			fmt.Printf("│ Release:     %-48s │\n", agentDetails.Release)
		}
		
		if agentDetails.OpenAPI != "" {
			fmt.Printf("│ OpenAPI:     %-48s │\n", truncateString(agentDetails.OpenAPI, 48))
		}
		
		if agentDetails.HowToUse != "" {
			fmt.Printf("│ How To Use:  %-48s │\n", truncateString(agentDetails.HowToUse, 48))
		}
		
		if !agentDetails.Expiration.IsZero() {
			fmt.Printf("│ Expires:     %-48s │\n", agentDetails.Expiration.Format("2006-01-02 15:04:05"))
		}
		
		if len(agentDetails.Tags) > 0 {
			fmt.Printf("│ Tags:        %-48s │\n", truncateString(formatTags(agentDetails.Tags), 48))
		}
		
		fmt.Println("└──────────────────────────────────────────────────────────────┘")
	}
	
	// Step 3: Deregister all agents
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║                 DEREGISTERING ALL AGENTS                  ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	
	fmt.Println("┌────────────────────────┬─────────────────────────────────────┐")
	fmt.Println("│ Agent Name             │ Status                              │")
	fmt.Println("├────────────────────────┼─────────────────────────────────────┤")
	
	for _, agent := range agents {
		name := padOrTruncate(agent.Name, 20)
		err := client.DeregisterAgent(agent.Name)
		if err != nil {
			fmt.Printf("│ %-20s │ ❌ Failed: %-26s │\n", name, truncateString(err.Error(), 26))
		} else {
			fmt.Printf("│ %-20s │ ✅ Successfully deregistered            │\n", name)
		}
	}
	
	fmt.Println("└────────────────────────┴─────────────────────────────────────┘")
	
	// Step 4: Verify all agents are gone
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║               VERIFYING DEREGISTRATION                    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	
	verifyAgents, err := client.ListAgents()
	if err != nil {
		log.Fatalf("Failed to list agents: %v", err)
	}
	
	if len(verifyAgents) > 0 {
		fmt.Printf("⚠️  Found %d remaining agents:\n", len(verifyAgents))
		for i, agent := range verifyAgents {
			fmt.Printf("   %d. %s\n", i+1, agent.Name)
		}
	} else {
		fmt.Println("✅ All agents were successfully removed!")
	}
	
	// Step 5: Register a new Geography agent
	fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║              REGISTERING GEOGRAPHY AGENT                  ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	
	newAgent := shwood.Agent{
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

	fmt.Println("Registering agent with properties:")
	fmt.Println("┌─────────────┬─────────────────────────────────────────────────┐")
	fmt.Printf("│ Name        │ %-47s │\n", newAgent.Name)
	fmt.Printf("│ Description │ %-47s │\n", truncateString(newAgent.Description, 47))
	fmt.Printf("│ Release     │ %-47s │\n", newAgent.Release)
	fmt.Printf("│ Tags        │ %-47s │\n", formatTags(newAgent.Tags))
	fmt.Println("└─────────────┴─────────────────────────────────────────────────┘")

	registeredAgent, err := client.RegisterAgent(newAgent)
	if err != nil {
		fmt.Printf("❌ Failed to register agent: %v\n", err)
	} else {
		fmt.Println("✅ Agent registered successfully!")
		fmt.Printf("   Name: %s\n", registeredAgent.Name)
		fmt.Printf("   Expiration: %s\n", registeredAgent.Expiration.Format("2006-01-02 15:04:05"))
	}
	
	fmt.Println("\n✨ All operations completed!")
}

// Helper functions for formatting
func padOrTruncate(s string, length int) string {
	if len(s) > length {
		return s[:length-3] + "..."
	}
	return s + strings.Repeat(" ", length-len(s))
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatTags(tags []string) string {
	return strings.Join(tags, ", ")
}
