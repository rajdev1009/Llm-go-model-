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

// System Prompt (bot_instructions.txt) load karne ka function
func getSystemPrompt() string {
	content, err := ioutil.ReadFile("bot_instructions.txt")
	if err != nil {
		log.Println("Warning: bot_instructions.txt nahi mili, default use kar rahe hain.")
		return "Tumhara naam Dev hai. Tum AstraToonix ke creator Raj Dev ke assistant ho."
	}
	return string(content)
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	// CORS Settings taaki website se connect ho sake
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid Request", http.StatusBadRequest)
		return
	}

	// Environment Variables se API Keys uthana
	apiKeys := []string{
		os.Getenv("API_KEY_1"),
		os.Getenv("API_KEY_2"),
		os.Getenv("API_KEY_3"),
		os.Getenv("API_KEY_4"),
	}

	// Round-robin logic keys rotate karne ke liye
	currentCount := atomic.AddUint64(&requestCounter, 1)
	keyIndex := (currentCount - 1) % uint64(len(apiKeys))
	selectedKey := apiKeys[keyIndex]

	if selectedKey == "" {
		log.Printf("Error: API_KEY_%d missing hai", keyIndex+1)
		http.Error(w, "API Key Not Found", http.StatusInternalServerError)
		return
	}

	// Gemini API Setup
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(selectedKey))
	if err != nil {
		log.Printf("Client error: %v", err)
		http.Error(w, "Server Error", http.StatusInternalServerError)
		return
	}
	defer client.Close()

	// 2026 Latest Model: gemini-3.1-pro
	model := client.GenerativeModel("gemini-3.1-pro")
	
	// Chat session start karna instructions ke saath
	chat := model.StartChat()
	chat.History = []*genai.Content{
		{
			Role: "user",
			Parts: []genai.Part{genai.Text(getSystemPrompt())},
		},
		{
			Role: "model",
			Parts: []genai.Part{genai.Text("Theek hai, main samajh gaya. Main Dev hoon, AstraToonix se. Chaliye shuru karte hain!")},
		},
	}

	// User ka message bhejna
	resp, err := chat.SendMessage(ctx, genai.Text(req.Message))
	if err != nil {
		log.Printf("Gemini Error: %v", err)
		http.Error(w, "AI Response Error", http.StatusInternalServerError)
		return
	}

	// Result nikalna
	var replyText string
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		replyText = fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	} else {
		replyText = "Bhai, kuch error aa gaya response mein."
	}

	// Frontend ko JSON response bhejna
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"reply": replyText,
	})
}

func main() {
	// API Endpoint
	http.HandleFunc("/chat", chatHandler)

	// Port configuration (Koyeb ke liye)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Dev Bot Server running on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
