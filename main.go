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
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var requestCounter uint64 = 0

type ChatRequest struct {
	Message string `json:"message"`
}

func getSystemPrompt() string {
	content, err := ioutil.ReadFile("bot_instructions.txt")
	if err != nil {
		return "Tumhara naam Dev hai. Tum AstraToonix ke creator Raj Dev ke assistant ho. Hindi mein jawab do."
	}
	return string(content)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	content, err := ioutil.ReadFile("index.html")
	if err != nil {
		fmt.Fprintf(w, "HTML file missing on server")
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(content)
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	apiKeys := []string{
		os.Getenv("API_KEY_1"),
		os.Getenv("API_KEY_2"),
		os.Getenv("API_KEY_3"),
		os.Getenv("API_KEY_4"),
	}
	currentCount := atomic.AddUint64(&requestCounter, 1)
	keyIndex := (currentCount - 1) % uint64(len(apiKeys))
	selectedKey := apiKeys[keyIndex]

	if selectedKey == "" {
		jsonError(w, "API key nahi mila. Koyeb mein API_KEY_1 set karo!", http.StatusInternalServerError)
		return
	}

	log.Printf("🔑 Using API key index %d | Message: %s", keyIndex, req.Message)

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(selectedKey))
	if err != nil {
		log.Printf("❌ Client error: %v", err)
		jsonError(w, "Gemini client error", http.StatusInternalServerError)
		return
	}
	defer client.Close()

	// ✅ March 2026 ka current best fast model
	model := client.GenerativeModel("gemini-2.5-flash")

	chat := model.StartChat()
	chat.History = []*genai.Content{
		{Role: "user", Parts: []genai.Part{genai.Text(getSystemPrompt())}},
		{Role: "model", Parts: []genai.Part{genai.Text("Ok! Main taiyaar hoon.")}},
	}

	// Retry logic for any transient errors
	var resp *genai.GenerateContentResponse
	var sendErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, sendErr = chat.SendMessage(ctx, genai.Text(req.Message))
		if sendErr == nil {
			break
		}
		if attempt < 2 {
			log.Printf("⚠️ Attempt %d failed (%v), retrying in 2s...", attempt+1, sendErr)
			time.Sleep(2 * time.Second)
		}
	}

	if sendErr != nil {
		log.Printf("❌ FINAL SendMessage error: %v", sendErr)  // full error ab logs mein dikhega
		jsonError(w, "Gemini busy hai, 10 sec baad try karo", http.StatusTooManyRequests)
		return
	}

	log.Printf("✅ Gemini response OK | Candidates: %d", len(resp.Candidates))

	if len(resp.Candidates) == 0 {
		jsonError(w, "Gemini ne koi reply nahi diya", http.StatusInternalServerError)
		return
	}

	var reply string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			reply += string(text)
		}
	}

	if reply == "" {
		jsonError(w, "Empty reply from Gemini", http.StatusInternalServerError)
		return
	}

	log.Printf("📤 Reply sent: %d chars", len(reply))
	json.NewEncoder(w).Encode(map[string]string{"reply": reply})
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/chat", chatHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🚀 AstraToonix Dev Chat Server running on port %s (gemini-2.5-flash)", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
