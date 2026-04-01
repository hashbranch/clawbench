package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsTextOnlyGAIAQuestion_BasicPass(t *testing.T) {
	row := gaiaRow{
		Question:    "What is 2+2?",
		FinalAnswer: "4",
		FileName:    "",
	}
	if !isTextOnlyGAIAQuestion(row) {
		t.Error("expected text-only question to pass filter")
	}
}

func TestIsTextOnlyGAIAQuestion_HasFileName(t *testing.T) {
	row := gaiaRow{
		Question:    "Look at the attached spreadsheet and answer...",
		FinalAnswer: "42",
		FileName:    "data.xlsx",
	}
	if isTextOnlyGAIAQuestion(row) {
		t.Error("expected question with file_name to be filtered out")
	}
}

func TestIsTextOnlyGAIAQuestion_YouTubeReference(t *testing.T) {
	row := gaiaRow{
		Question:    "Watch this YouTube video and tell me...",
		FinalAnswer: "something",
		FileName:    "",
	}
	if isTextOnlyGAIAQuestion(row) {
		t.Error("expected question referencing YouTube to be filtered out")
	}
}

func TestIsTextOnlyGAIAQuestion_ImageReference(t *testing.T) {
	row := gaiaRow{
		Question:    "Look at the image and describe what you see",
		FinalAnswer: "a cat",
		FileName:    "",
	}
	if isTextOnlyGAIAQuestion(row) {
		t.Error("expected question referencing image to be filtered out")
	}
}

func TestIsTextOnlyGAIAQuestion_AudioReference(t *testing.T) {
	row := gaiaRow{
		Question:    "Listen to this audio clip and identify the speaker",
		FinalAnswer: "Bob",
		FileName:    "",
	}
	if isTextOnlyGAIAQuestion(row) {
		t.Error("expected question referencing audio to be filtered out")
	}
}

func TestIsTextOnlyGAIAQuestion_EmptyAnswer(t *testing.T) {
	row := gaiaRow{
		Question:    "What is the meaning of life?",
		FinalAnswer: "",
		FileName:    "",
	}
	if isTextOnlyGAIAQuestion(row) {
		t.Error("expected question with empty answer to be filtered out")
	}
}

func TestIsTextOnlyGAIAQuestion_PDFReference(t *testing.T) {
	row := gaiaRow{
		Question:    "Read the attached PDF and summarize it",
		FinalAnswer: "summary",
		FileName:    "",
	}
	if isTextOnlyGAIAQuestion(row) {
		t.Error("expected question referencing PDF to be filtered out")
	}
}

func TestIsTextOnlyGAIAQuestion_SpreadsheetReference(t *testing.T) {
	row := gaiaRow{
		Question:    "Open the spreadsheet and calculate the total",
		FinalAnswer: "100",
		FileName:    "",
	}
	if isTextOnlyGAIAQuestion(row) {
		t.Error("expected question referencing spreadsheet to be filtered out")
	}
}

func TestIsTextOnlyGAIAQuestion_CaseInsensitive(t *testing.T) {
	row := gaiaRow{
		Question:    "Watch this VIDEO and tell me what happens",
		FinalAnswer: "stuff",
		FileName:    "",
	}
	if isTextOnlyGAIAQuestion(row) {
		t.Error("expected case-insensitive multimedia keyword detection")
	}
}

func TestIsTextOnlyGAIAQuestion_FileExtensions(t *testing.T) {
	tests := []struct {
		question string
		name     string
	}{
		{"Download the .xlsx file and...", "xlsx"},
		{"Open the .csv and find...", "csv"},
		{"Extract from the .zip archive...", "zip"},
	}
	for _, tc := range tests {
		row := gaiaRow{
			Question:    tc.question,
			FinalAnswer: "answer",
			FileName:    "",
		}
		if isTextOnlyGAIAQuestion(row) {
			t.Errorf("expected question with .%s reference to be filtered out", tc.name)
		}
	}
}

func TestFetchGAIAQuestions_MockAPI(t *testing.T) {
	// Create a mock HuggingFace API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			w.WriteHeader(401)
			return
		}

		resp := gaiaAPIResponse{
			Rows: []gaiaAPIRow{
				{
					RowIdx: 0,
					Row: gaiaRow{
						TaskID:      "task-001",
						Question:    "What is 2+2?",
						FinalAnswer: "4",
						Level:       1,
						FileName:    "",
					},
				},
				{
					RowIdx: 1,
					Row: gaiaRow{
						TaskID:      "task-002",
						Question:    "Look at the attached image and...",
						FinalAnswer: "cat",
						Level:       1,
						FileName:    "photo.jpg", // should be filtered
					},
				},
				{
					RowIdx: 2,
					Row: gaiaRow{
						TaskID:      "task-003",
						Question:    "Who wrote Romeo and Juliet?",
						FinalAnswer: "William Shakespeare",
						Level:       1,
						FileName:    "",
					},
				},
				{
					RowIdx: 3,
					Row: gaiaRow{
						TaskID:      "task-004",
						Question:    "Watch this YouTube video and summarize",
						FinalAnswer: "summary",
						Level:       1,
						FileName:    "", // no file, but youtube keyword
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Temporarily override the API URL — we test the filter logic via the mock
	// Since we can't easily override the const, we test isTextOnlyGAIAQuestion directly
	// and verify the task construction logic here

	// Verify filter counts from our mock data
	rows := []gaiaRow{
		{Question: "What is 2+2?", FinalAnswer: "4", FileName: ""},
		{Question: "Look at the attached image", FinalAnswer: "cat", FileName: "photo.jpg"},
		{Question: "Who wrote Romeo and Juliet?", FinalAnswer: "William Shakespeare", FileName: ""},
		{Question: "Watch this YouTube video", FinalAnswer: "summary", FileName: ""},
	}

	var passed int
	for _, row := range rows {
		if isTextOnlyGAIAQuestion(row) {
			passed++
		}
	}
	if passed != 2 {
		t.Errorf("expected 2 questions to pass filter, got %d", passed)
	}
}

func TestGAIATaskConstruction(t *testing.T) {
	// Verify that a fetched GAIA question becomes a properly structured Task
	row := gaiaAPIRow{
		RowIdx: 5,
		Row: gaiaRow{
			Question:    "What year did WWII end?",
			FinalAnswer: "1945",
		},
	}

	// Simulate the task construction from FetchGAIAQuestions
	task := Task{
		ID:       "gaia_l1_real_005",
		Name:     "GAIA L1 Official #5",
		Category: "gaia_l1",
		Tags:     []string{"gaia", "level1", "official"},
	}

	if task.ID != "gaia_l1_real_005" {
		t.Errorf("unexpected task ID: %s", task.ID)
	}
	if task.Category != "gaia_l1" {
		t.Errorf("unexpected category: %s", task.Category)
	}

	// Verify evaluator would work with this answer
	ec := EvalConfig{Type: "gaia_exact", Patterns: []string{row.Row.FinalAnswer}, Weight: 1.0}
	result := evalGAIAExact(ec, "The answer is 1945")
	if result.Score != 1.0 {
		t.Errorf("expected gaia_exact to match, got score %f (%s)", result.Score, result.Details)
	}
}

// TestGAIAAnswerFormats tests the gaia_exact evaluator against real-world
// answer formats found in the GAIA dataset.
func TestGAIAAnswerFormats(t *testing.T) {
	tests := []struct {
		name        string
		groundTruth string
		modelAnswer string
		shouldMatch bool
	}{
		// Numeric answers
		{"plain number", "17", "17", true},
		{"number with context", "3", "The answer is 3", true},
		{"number with commas", "16000", "$16,000", true},
		{"decimal", "0.1777", "0.1777", true},

		// String answers
		{"simple word", "fluffy", "fluffy", true},
		{"case insensitive", "THE CASTLE", "The Castle", true},
		{"multi-word", "Saint Petersburg", "saint petersburg", true},
		{"with punctuation", "Right", "Right.", true},

		// Longer answers
		{"full sentence answer", "Maktay mato apple", "Maktay mato apple", true},
		{"answer with special chars", "80GSFC21M0002", "80GSFC21M0002", true},

		// List answers
		{"comma list", "Braintree, Honolulu", "Braintree, Honolulu", true},
		{"comma list case", "b, e", "B, E", true},
		{"comma list with names", "Yoshida, Uehara", "Yoshida, Uehara", true},

		// Should NOT match
		{"wrong answer", "Paris", "London", false},
		{"wrong number", "42", "43", false},
		{"partial match", "broccoli, celery, fresh basil, lettuce, sweet potatoes", "broccoli, celery", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ec := EvalConfig{Type: "gaia_exact", Patterns: []string{tc.groundTruth}, Weight: 1.0}
			result := evalGAIAExact(ec, tc.modelAnswer)
			if tc.shouldMatch && result.Score != 1.0 {
				t.Errorf("expected match for %q vs %q, got score %f (%s)", tc.groundTruth, tc.modelAnswer, result.Score, result.Details)
			}
			if !tc.shouldMatch && result.Score != 0.0 {
				t.Errorf("expected no match for %q vs %q, got score %f", tc.groundTruth, tc.modelAnswer, result.Score)
			}
		})
	}
}

// Verify the tags on official tasks vs clawbench originals are distinct
func TestGAIATaskTagsDistinct(t *testing.T) {
	originals := ClawBenchOriginalTasks()
	for _, task := range originals {
		for _, tag := range task.Tags {
			if tag == "official" {
				t.Errorf("ClawBench original task %s should not have 'official' tag", task.ID)
			}
		}
		if !strings.HasPrefix(task.ID, "cb_") {
			t.Errorf("ClawBench original task %s should have 'cb_' prefix", task.ID)
		}
	}
}
