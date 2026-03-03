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
		jsonError(w, "API key nahi mila. DigitalOcean mein API_KEY_1 set karo!", http.StatusInternalServerError)
		return
	}

	log.Printf("🔑 Using API key index %d | Message: %s", keyIndex, req.Message)

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(selectedKey))
	if err != nil {
		log.Printf("❌ Client error: %v", err)
		jsonError(w, "Gemini client error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-3.1-pro-preview")

	chat := model.StartChat()
	chat.History = []*genai.Content{
		{Role: "user", Parts: []genai.Part{genai.Text(getSystemPrompt())}},
		{Role: "model", Parts: []genai.Part{genai.Text("Ok! Main taiyaar hoon.")}},
	}

	resp, err := chat.SendMessage(ctx, genai.Text(req.Message))
	if err != nil {
		log.Printf("❌ SendMessage error: %v", err)
		jsonError(w, "Gemini se reply nahi aaya: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Gemini raw response received | Candidates: %d", len(resp.Candidates))

	if len(resp.Candidates) == 0 {
		log.Println("❌ No candidates in response")
		jsonError(w, "Gemini ne koi reply nahi diya (safety block?)", http.StatusInternalServerError)
		return
	}

	// ✅ PROPER TEXT EXTRACTION (yeh "No reply" ka asli culprit tha)
	var reply string
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			reply += string(text)
		} else if textPtr, ok := part.(*genai.Text); ok { // pointer case bhi handle
			reply += string(*textPtr)
		}
	}

	log.Printf("📤 Final reply length: %d chars | First 100: %s...", len(reply), reply[:min(100, len(reply))])

	if reply == "" {
		jsonError(w, "Gemini ne empty reply diya (model issue ya safety filter)", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"reply": reply})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/chat", chatHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🚀 AstraToonix Dev Chat Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
