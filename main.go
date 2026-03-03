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
		return "Tumhara naam Dev hai. Tum AstraToonix ke creator Raj Dev ke assistant ho."
	}
	return string(content)
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
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	json.NewDecoder(r.Body).Decode(&req)

	apiKeys := []string{os.Getenv("API_KEY_1"), os.Getenv("API_KEY_2"), os.Getenv("API_KEY_3"), os.Getenv("API_KEY_4")}
	currentCount := atomic.AddUint64(&requestCounter, 1)
	keyIndex := (currentCount - 1) % uint64(len(apiKeys))
	selectedKey := apiKeys[keyIndex]

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(selectedKey))
	if err != nil {
		http.Error(w, "Client error", http.StatusInternalServerError)
		return
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-3.1-pro")
	chat := model.StartChat()
	chat.History = []*genai.Content{
		{Role: "user", Parts: []genai.Part{genai.Text(getSystemPrompt())}},
		{Role: "model", Parts: []genai.Part{genai.Text("Ok! Main taiyaar hoon.")}},
	}

	resp, err := chat.SendMessage(ctx, genai.Text(req.Message))
	if err != nil {
		http.Error(w, "AI Error", http.StatusInternalServerError)
		return
	}

	reply := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	json.NewEncoder(w).Encode(map[string]string{"reply": reply})
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/chat", chatHandler)

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
