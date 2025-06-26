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
	Type      string   `json:"type"`
	Message   string   `json:"message,omitempty"`
	Questions []string `json:"questions,omitempty"`
	Tasks     []string `json:"tasks,omitempty"`
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

	response, err := getAITasks(req.Goal, req.Context)
	if err != nil {
		log.Printf("error generating tasks: %v", err)
		http.Error(w, "failed to generate tasks", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response)
}

func getAITasks(goal, userContext string) (*TaskResponse, error) {
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

	model := client.GenerativeModel("gemini-2.5-flash")

	prompt := fmt.Sprintf(`You are helping someone achieve their goal: "%s"

Their current context: "%s"

Analyze if you have enough specific information to create 5 meaningful, actionable daily tasks. Consider:
- Is the goal specific enough?
- Is there enough context about their current situation?
- Are there important choices or preferences that would significantly change the approach?

If you need more information, respond with "QUESTIONS:" followed by 2-4 specific questions that would help you create better tasks.

If you have enough information, respond with "TASKS:" followed by exactly 5 simple, actionable daily tasks.
DO NOT REPEAT QUESTIONS.
IMPORTANT: If the user answered "I don't know" to any question that is core to achieving their goal, create tasks specifically aimed at helping them discover the answer to that question. For example, if they don't know which programming language to learn for software development, the tasks should focus on researching and comparing different programming languages.

Requirements for tasks:
- Each task should be completable in 15-30 minutes
- Tasks should be concrete and specific
- Focus on small, incremental progress
- Do not use any emojis or special characters
- One task per line
- If user said "I don't know" to core questions, prioritize tasks that help them find those answers

Requirements for questions:
- Ask about specific details that would change your task recommendations
- Focus on the most important missing information
- Keep questions clear and direct
- Do not use any emojis or special characters

Example response formats:

QUESTIONS:
Which dialect of Arabic are you most interested in learning?
Do you have any specific goals for using Arabic?
Are you planning to travel to an Arabic-speaking country?

OR

TASKS:
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
	content = strings.TrimSpace(content)

	if strings.HasPrefix(content, "QUESTIONS:") {
		questionText := strings.TrimPrefix(content, "QUESTIONS:")
		questionText = strings.TrimSpace(questionText)
		lines := strings.Split(questionText, "\n")

		var questions []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				questions = append(questions, line)
			}
		}

		if len(questions) == 0 {
			return nil, fmt.Errorf("no questions extracted from response")
		}

		return &TaskResponse{
			Type:      "questions",
			Message:   "I need more information to create the best tasks for you:",
			Questions: questions,
		}, nil
	}

	if strings.HasPrefix(content, "TASKS:") {
		taskText := strings.TrimPrefix(content, "TASKS:")
		taskText = strings.TrimSpace(taskText)
		lines := strings.Split(taskText, "\n")

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

		return &TaskResponse{
			Type:  "tasks",
			Tasks: tasks,
		}, nil
	}

	return nil, fmt.Errorf("unexpected response format from AI")
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
