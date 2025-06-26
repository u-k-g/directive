package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

type TaskRequest struct {
	Goal    string `json:"goal"`
	Context string `json:"context"`
}

type TaskResponse struct {
	Tasks []string `json:"tasks"`
}

func generateTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	tasks, err := getAITasks(req.Goal, req.Context)
	if err != nil {
		log.Printf("error generating tasks: %v", err)
		http.Error(w, "failed to generate tasks", http.StatusInternalServerError)
		return
	}

	response := TaskResponse{Tasks: tasks}
	json.NewEncoder(w).Encode(response)
}

func getAITasks(goal, userContext string) ([]string, error) {
	ctx := context.Background()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-flash-lite-preview-06-17")

	prompt := fmt.Sprintf(`Generate exactly 5 simple, actionable daily tasks for someone who wants to %s. 

Context: %s

Requirements:
- Each task should be completable in 15-30 minutes
- Tasks should be concrete and specific
- Focus on small, incremental progress
- Return only the tasks, one per line
- No numbering or bullet points
- Do not use any emojis or special characters

Example format:
research one specific skill needed for becoming a data scientist
identify one person in your network who works in data science
read one article about machine learning fundamentals
practice one Python coding exercise
reflect on what you learned today`, goal, userContext)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content generated")
	}

	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	lines := strings.Split(strings.TrimSpace(content), "\n")

	var tasks []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			tasks = append(tasks, line)
		}
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks extracted from response")
	}

	return tasks, nil
}

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Printf("warning: could not load .env file: %v", err)
	}

	http.HandleFunc("/api/tasks", generateTasks)

	fmt.Println("server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
