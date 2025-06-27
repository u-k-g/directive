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
	Step    string   `json:"step"`
	Goal    string   `json:"goal"`
	Context string   `json:"context"`
	Answers []string `json:"answers,omitempty"`
	Roadmap []string `json:"roadmap,omitempty"`
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
		response, err = createRoadmap(req.Goal, req.Context, req.Answers)
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

Your task is to ask 3-5 strategic questions that will help you create a comprehensive roadmap for achieving this goal.

Focus on questions about:
- Specific preferences or choices that affect the approach
- Timeline and constraints
- Current skill level or experience
- Resources available
- End goals or specific outcomes desired

Respond with "QUESTIONS:" followed by your questions, one per line.

Requirements:
- Ask strategic questions that will shape the roadmap
- Keep questions clear and direct
- Do not use any emojis or special characters
- Focus on information that significantly impacts the learning/achievement path

Example response format:

QUESTIONS:
What is your target timeline for achieving this goal?
Do you prefer self-study or structured courses?
What is your current experience level with this topic?
What specific outcome are you hoping to achieve?`, goal, userContext)

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

func createRoadmap(goal, userContext string, answers []string) (*TaskResponse, error) {
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

	answersText := strings.Join(answers, "\n")

	prompt := fmt.Sprintf(`Based on this goal: "%s"

Context: "%s"

And these answers to strategic questions:
%s

Create a high-level roadmap with 5-8 major milestones for achieving this goal. Each milestone should represent a significant step or phase in the journey.

Respond with "ROADMAP:" followed by the milestones, one per line.

Requirements:
- Each milestone should be a major achievement or phase
- Order them logically from beginning to end
- Make them specific but not overly detailed
- Focus on key learning phases or skill development stages
- Do not use any emojis or special characters

Example response format:

ROADMAP:
Master basic programming fundamentals and syntax
Build first simple projects and understand core concepts
Learn advanced programming patterns and best practices
Develop portfolio projects showcasing different skills
Apply for entry-level positions and practice interviews`, goal, userContext, answersText)

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
- Each task should be completable in 15-30 minutes
- Tasks should be concrete and specific
- Focus on the first milestone/phase of the roadmap
- Tasks should be beginner-friendly and actionable today
- Do not use any emojis or special characters

Example response format:

TASKS:
research three online programming courses for beginners
install a code editor and set up your development environment
complete one simple programming tutorial or exercise
join one programming community or forum
write down three specific programming projects you'd like to build`, goal, userContext, roadmapText)

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
