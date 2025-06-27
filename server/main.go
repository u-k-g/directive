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
	Step      string   `json:"step"`
	Goal      string   `json:"goal"`
	Context   string   `json:"context"`
	Questions []string `json:"questions,omitempty"`
	Answers   []string `json:"answers,omitempty"`
	Roadmap   []string `json:"roadmap,omitempty"`
}

type TaskResponse struct {
	Type      string   `json:"type"`
	Message   string   `json:"message,omitempty"`
	Questions []string `json:"questions,omitempty"`
	Roadmap   []string `json:"roadmap,omitempty"`
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

	var response *TaskResponse
	var err error

	switch req.Step {
	case "analyze_goal":
		response, err = analyzeGoal(req.Goal, req.Context)
	case "create_roadmap":
		response, err = createRoadmap(req.Goal, req.Context, req.Questions, req.Answers)
	case "generate_tasks":
		response, err = generateDailyTasks(req.Goal, req.Context, req.Roadmap)
	default:
		http.Error(w, "invalid step", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("error in step %s: %v", req.Step, err)
		http.Error(w, "failed to process request", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(response)
}

func analyzeGoal(goal, userContext string) (*TaskResponse, error) {
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

Your task is to ask 3-4 simple, direct questions to understand their current state and aspirations. These questions will help you create a personalized roadmap.

Focus on direct questions about:
- Their current experience and skills (e.g., "Have you written code before?")
- Any past projects or practical experience (e.g., "Describe any software you've built")
- Their specific career or end-state goals (e.g., "What kind of company do you want to work at?")

Respond with "QUESTIONS:" followed by your questions, one per line.

Requirements:
- Ask simple, direct questions that are easy to answer
- Avoid academic or high-school related questions
- Focus on practical experience and goals
- Do not use any emojis or special characters

Example input:
Goal: "become a software engineer"
Context: "just graduated high school"

Example response format:

QUESTIONS:
What experience do you have in software engineering?
If you've built any software, describe it.
What kind of company do you hope to engineer at?
Any other details to give me:`, goal, userContext)

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
			Message:   "I need to understand your specific situation to create the best roadmap:",
			Questions: questions,
		}, nil
	}

	return nil, fmt.Errorf("unexpected response format from AI")
}

func createRoadmap(goal, userContext string, questions, answers []string) (*TaskResponse, error) {
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

	var qaBuilder strings.Builder
	for i, q := range questions {
		if i < len(answers) {
			qaBuilder.WriteString(fmt.Sprintf("Question: %s\nAnswer: %s\n\n", q, answers[i]))
		}
	}
	qaText := qaBuilder.String()

	prompt := fmt.Sprintf(`Based on this goal: "%s"

Context: "%s"

And this Q&A:
%s

Create a high-level roadmap with major milestones for achieving this goal. Each milestone should represent a significant, concrete achievement.

Respond with "ROADMAP:" followed by the milestones, one per line.

Requirements:
- Each milestone should be a real achievement (e.g., "Build a full-stack web application")
- Avoid vague, abstract concepts (e.g., "Master core principles")
- Order them logically from beginning to end
- Make them specific but not overly specific. do not make them vague or hard to answer
- Focus on key achievements or shifts in focus
- Do not use any emojis or special characters

Example input:
Goal: "become a software engineer"
Context: "recent graduate with computer science degree, looking for entry-level position"
Q&A:
Question: What experience do you have in software engineering?
Answer: None, just some college projects.
Question: If you've built any software, describe it.
Answer: A small web app for a class.
Question: What kind of company do you hope to engineer at?
Answer: A fast-paced startup.

Example response format:

ROADMAP:
Build a personal portfolio website from scratch
Contribute to an open-source project
Create a full-stack web application with a database
Build a mobile application for iOS or Android
Complete a data structures and algorithms course
Prepare for and attend technical interviews`, goal, userContext, qaText)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content generated")
	}

	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	content = strings.TrimSpace(content)

	if strings.HasPrefix(content, "ROADMAP:") {
		roadmapText := strings.TrimPrefix(content, "ROADMAP:")
		roadmapText = strings.TrimSpace(roadmapText)
		lines := strings.Split(roadmapText, "\n")

		var roadmap []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				roadmap = append(roadmap, line)
			}
		}

		if len(roadmap) == 0 {
			return nil, fmt.Errorf("no roadmap extracted from response")
		}

		return &TaskResponse{
			Type:    "roadmap",
			Message: "Here's your personalized roadmap. You can edit any milestone before we create your daily tasks:",
			Roadmap: roadmap,
		}, nil
	}

	return nil, fmt.Errorf("unexpected response format from AI")
}

func generateDailyTasks(goal, userContext string, roadmap []string) (*TaskResponse, error) {
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

	roadmapText := strings.Join(roadmap, "\n")

	prompt := fmt.Sprintf(`Based on this goal: "%s"

Context: "%s"

And this confirmed roadmap:
%s

Generate 5 specific daily tasks that will help them progress toward the FIRST milestone in their roadmap. Focus on concrete, actionable steps they can take today to start their journey.

Respond with "TASKS:" followed by exactly 5 tasks, one per line.

Requirements:
- Each task should be completable in a day
- Tasks should be concrete and specific with no vagueness
- Focus on the first milestone/phase of the roadmap
- Tasks should be actionable today
- Do not use any emojis or special characters

Example input:
Goal: "become a software engineer"
Context: "recent graduate with computer science degree, looking for entry-level position"
Roadmap: "Master basic programming fundamentals and syntax", "Build first simple projects", etc.

Example response format:

TASKS:
think of something you want to build or research beginner projects
follow a programming tutorial all the way through or make signficant progess
brainstorm even more complex programming projects that you want to build`, goal, userContext, roadmapText)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content generated")
	}

	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	content = strings.TrimSpace(content)

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
