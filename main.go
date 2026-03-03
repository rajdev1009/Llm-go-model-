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

	// Google Generative AI ka official package
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Yahan apni 4 asli API keys daalna
var apiKeys = []string{
	"API_KEY_1",
	"API_KEY_2",
	"API_KEY_3",
	"API_KEY_4",
}

var requestCounter uint64 = 0

type ChatRequest struct {
	Message string `json:"message"`
}

func getSystemPrompt() string {
	content, err := ioutil.ReadFile("bot_instructions.txt")
	if err != nil {
		log.Println("Warning: bot_instructions.txt file nahi mili.")
		return "Tumhara naam Dev hai."
	}
	return string(content)
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	// CORS Headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Message == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// 1, 2, 3, 4 API key rotation
	currentCount := atomic.AddUint64(&requestCounter, 1)
	keyIndex := (currentCount - 1) % uint64(len(apiKeys))
	selectedAPIKey := apiKeys[keyIndex]

	botRules := getSystemPrompt()

	fmt.Printf("User Message: %s | API Key Number Used: %d\n", req.Message, keyIndex+1)

	// --- GOOGLE GEMINI API CALL LOGIC STARTS HERE ---
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(selectedAPIKey))
	if err != nil {
		log.Printf("Client error: %v", err)
		http.Error(w, "API Client Error", http.StatusInternalServerError)
		return
	}
	defer client.Close()

	// 2026 ka latest model use kar rahe hain: gemini-3.1-pro
	model := client.GenerativeModel("gemini-3.1-pro")
	
	// System Instructions (Tumhari details aur rules) AI ko pass karna
	model.SystemInstruction = genai.NewUserContent(genai.Text(botRules))

	// User ka message AI ko bhejna
	resp, err := model.GenerateContent(ctx, genai.Text(req.Message))
	if err != nil {
		log.Printf("Generation error: %v", err)
		http.Error(w, "AI Generation Error", http.StatusInternalServerError)
		return
	}

	// AI ka reply extract karna
	var replyText string
	if resp != nil && len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		replyText = fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	} else {
		replyText = "Bhai, server side se API ka response theek se nahi aaya."
	}
	// --- GOOGLE GEMINI API CALL LOGIC ENDS HERE ---

	response := map[string]string{
		"reply": replyText,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/chat", chatHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("Server port " + port + " par start ho raha hai...")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
