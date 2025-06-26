# 🎯 Directive Backend Test Cases

This document contains test scenarios to validate the backend API functionality.

## API Endpoint
`POST http://localhost:8080/api/tasks`

## Test Cases

### 1. Software Engineer - Recent Graduate
**Goal:** `become a software engineer`  
**Context:** `recent graduate with computer science degree`

### 2. Entrepreneur - No Experience
**Goal:** `start my own business`  
**Context:** `working full-time but want to start a side business`

### 3. Data Scientist - Career Change
**Goal:** `become a data scientist`  
**Context:** `currently working in marketing but have basic python knowledge`

### 4. Fitness - Beginner
**Goal:** `get fit and lose weight`  
**Context:** `haven't exercised in years, work desk job`

### 5. Language Learning
**Goal:** `learn Spanish fluently`  
**Context:** `complete beginner, planning to travel to Spain next year`

### 6. Creative Writing
**Goal:** `write and publish a novel`  
**Context:** `love reading but never written anything longer than school essays`

## Testing Methods

### Manual cURL Testing
```bash
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"goal":"become a software engineer","context":"recent graduate with computer science degree"}'
```

### Web Interface
Open `index.html` in your browser and test the scenarios above.

### Expected Response Format
```json
{
  "tasks": [
    "task 1 description",
    "task 2 description", 
    "task 3 description",
    "task 4 description",
    "task 5 description"
  ]
}
```

## Notes
- Each response should contain exactly 5 tasks
- Tasks should be actionable and completable in 15-30 minutes
- Tasks should be personalized based on the goal and context provided 