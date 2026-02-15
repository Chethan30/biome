package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/biome/agent-core/packages/agent/orchestrators/agentic"
	"github.com/biome/agent-core/pkg/httpapi"
	"github.com/biome/agent-mind/openrouter"
)

func main() {
	fmt.Println("🌐 Agent-Core HTTP Server")
	fmt.Println("==========================")

	llmModel := "anthropic/claude-3-haiku"

	// Get API key (optional)
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	var llmProvider *openrouter.Provider
	if apiKey != "" {
		llmProvider = openrouter.NewProvider(apiKey, llmModel)
		fmt.Printf("✅ LLM Provider: %s\n", llmModel)
	} else {
		fmt.Println("⚠️  No OPENROUTER_API_KEY - using mock mode")
	}

	// Tools are supplied per request in POST /agent/prompt via the "tools" array (tool configs).
	fmt.Println("✅ Tools: pass per request in POST /agent/prompt (see \"tools\" field)")

	apiServer := httpapi.NewServer(llmProvider)

	// Setup routes
	http.HandleFunc("/agent/prompt", apiServer.CORSMiddleware(apiServer.PromptHandler))
	http.HandleFunc("/tools/register", apiServer.CORSMiddleware(apiServer.RegisterToolHandler))
	http.HandleFunc("/tools", apiServer.CORSMiddleware(apiServer.ListToolsHandler))
	http.HandleFunc("/health", apiServer.CORSMiddleware(apiServer.HealthHandler))

	// Start server
	port := "8080"
	fmt.Printf("\n🚀 Server starting on http://localhost:%s\n\n", port)
	fmt.Println("Endpoints:")
	fmt.Println("  POST   http://localhost:8080/agent/prompt")
	fmt.Println("  POST   http://localhost:8080/tools/register")
	fmt.Println("  GET    http://localhost:8080/tools")
	fmt.Println("  GET    http://localhost:8080/health")
	fmt.Println("\nExample (with optional tools in request):")
	fmt.Println("  curl -X POST http://localhost:8080/agent/prompt \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"message\": \"Hello!\", \"stream\": true, \"tools\": []}'")
	fmt.Println()

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
