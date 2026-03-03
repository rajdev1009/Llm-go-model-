package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var requestCounter uint64 = 0

type ChatRequest struct {
	Message string `json:"message"`
}

// System Prompt Load karna
func getSystemPrompt() string {
	content, err := ioutil.ReadFile("bot_instructions.txt")
	if err != nil {
		return "Tumhara naam Dev hai. Tum AI nahi ho."
	}
	return string(content)
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req ChatRequest
	json.NewDecoder(r.Body).Decode(&req)

	// API Keys List (Koyeb Variables se load ho rahi hain)
	apiKeys := []string{
		os.Getenv("API_KEY_1"),
		os.Getenv("API_KEY_2"),
		os.Getenv("API_KEY_3"),
		os.Getenv("API_KEY_4"),
	}

	// Counter badhana aur key select karna
	currentCount := atomic.AddUint64(&requestCounter, 1)
	keyIndex := (currentCount - 1) % uint64(len(apiKeys))
	selectedKey := apiKeys[keyIndex]

	if selectedKey == "" {
		http.Error(w, "API Key not found in Environment", http.StatusInternalServerError)
		return
	}

	// Gemini API Call
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(selectedKey))
	if err != nil {
		http.Error(w, "Client error", http.StatusInternalServerError)
		return
	}
	defer client.Close()

	// 2026 Latest Model: gemini-3.1-pro
	model := client.GenerativeModel("gemini-3.1-pro")
	model.SystemInstruction = genai.NewUserContent(genai.Text(getSystemPrompt()))

	resp, err := model.GenerateContent(ctx, genai.Text(req.Message))
	if err != nil {
		http.Error(w, "AI Error", http.StatusInternalServerError)
		return
	}

	reply := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	json.NewEncoder(w).Encode(map[string]string{"reply": reply})
}

func main() {
	http.HandleFunc("/chat", chatHandler)
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	fmt.Println("Server running on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
