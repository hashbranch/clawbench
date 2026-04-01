package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	gaiaAPIURL = "https://datasets-server.huggingface.co/rows?dataset=gaia-benchmark%2FGAIA&config=2023_level1&split=validation&offset=0&length=100"
)

// gaiaAPIResponse is the top-level response from the HuggingFace datasets-server API.
type gaiaAPIResponse struct {
	Rows []gaiaAPIRow `json:"rows"`
}

// gaiaAPIRow wraps each row returned by the API.
type gaiaAPIRow struct {
	RowIdx int        `json:"row_idx"`
	Row    gaiaRow    `json:"row"`
}

// gaiaRow is the actual question data inside each API row.
type gaiaRow struct {
	TaskID      string `json:"task_id"`
	Question    string `json:"Question"`
	FinalAnswer string `json:"Final answer"`
	Level       int    `json:"Level"`
	FileName    string `json:"file_name"`
}

// FetchGAIAQuestions downloads Level 1 validation questions from HuggingFace
// and returns them as benchmark Tasks. Questions requiring file attachments
// or multimedia (youtube, video, audio, image) are filtered out since
// ClawBench only supports text-based tool use.
//
// Requires a valid HuggingFace token with access to the gated
// gaia-benchmark/GAIA dataset.
func FetchGAIAQuestions(hfToken string) ([]Task, error) {
	req, err := http.NewRequest("GET", gaiaAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+hfToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching GAIA dataset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("HuggingFace auth failed (HTTP %d). Ensure your token has access to gaia-benchmark/GAIA", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HuggingFace API returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var apiResp gaiaAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decoding API response: %w", err)
	}

	var tasks []Task
	for _, row := range apiResp.Rows {
		if !isTextOnlyGAIAQuestion(row.Row) {
			continue
		}
		task := Task{
			ID:         fmt.Sprintf("gaia_l1_real_%03d", row.RowIdx),
			Name:       fmt.Sprintf("GAIA L1 Official #%d", row.RowIdx),
			Category:   "gaia_l1",
			Tags:       []string{"gaia", "level1", "official"},
			Prompt:     row.Row.Question,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{row.Row.FinalAnswer}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// isTextOnlyGAIAQuestion returns true if the question can be answered
// with text-based tools only (no file attachments, no multimedia).
func isTextOnlyGAIAQuestion(row gaiaRow) bool {
	// Skip questions that require file attachments
	if row.FileName != "" {
		return false
	}

	// Skip questions referencing multimedia content
	lower := strings.ToLower(row.Question)
	multimediaKeywords := []string{
		"youtube", "video", "audio", "image", "photo",
		"picture", "screenshot", "recording", "mp3", "mp4",
		"wav", "jpg", "jpeg", "png", "gif", "pdf",
		"spreadsheet", "excel", ".xlsx", ".csv", ".zip",
		"attached", "the file", "this file",
	}
	for _, kw := range multimediaKeywords {
		if strings.Contains(lower, kw) {
			return false
		}
	}

	// Skip questions with empty answers (data quality)
	if strings.TrimSpace(row.FinalAnswer) == "" {
		return false
	}

	return true
}
