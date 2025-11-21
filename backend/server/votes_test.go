package server

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/skarlso/kube_adventures/voting/backend/parser"
)

func TestNewVoteManager(t *testing.T) {
	vm := NewVoteManager()

	if vm == nil {
		t.Fatal("expected non-nil VoteManager")
	}

	if vm.votes == nil {
		t.Error("votes map should be initialized")
	}

	if vm.voters == nil {
		t.Error("voters map should be initialized")
	}

	if vm.clients == nil {
		t.Error("clients map should be initialized")
	}

	if vm.broadcast == nil {
		t.Error("broadcast channel should be initialized")
	}
}

func TestStartVoting(t *testing.T) {
	vm := NewVoteManager()
	go vm.Run()

	questionID := "test-question"
	choices := []string{"choice-a", "choice-b", "choice-c"}
	duration := 100 * time.Millisecond

	completed := false
	vm.StartVoting(questionID, choices, duration, func(results map[string]int, winner string) {
		completed = true
	})

	if !vm.IsVotingActive() {
		t.Error("voting should be active")
	}

	vm.mu.RLock()
	if vm.currentQuestion != questionID {
		t.Errorf("currentQuestion = %q, want %q", vm.currentQuestion, questionID)
	}
	vm.mu.RUnlock()

	// Verify vote counts initialized
	results := vm.GetResults(questionID)
	if len(results) != len(choices) {
		t.Errorf("got %d choices, want %d", len(results), len(choices))
	}

	for _, choice := range choices {
		if results[choice] != 0 {
			t.Errorf("choice %q count = %d, want 0", choice, results[choice])
		}
	}

	// Wait for timer to complete
	time.Sleep(150 * time.Millisecond)

	if !completed {
		t.Error("onComplete callback should have been called")
	}

	if vm.IsVotingActive() {
		t.Error("voting should be inactive after timer")
	}
}

func TestStartVotingWithChoices(t *testing.T) {
	vm := NewVoteManager()
	go vm.Run()

	questionID := "test-decision"
	choiceIDs := []string{"opt-a", "opt-b"}
	choiceObjects := []parser.Choice{
		{ID: "opt-a", Label: "Option A", Next: "path-a", Risk: "low"},
		{ID: "opt-b", Label: "Option B", Next: "path-b", Risk: "high"},
	}
	duration := 100 * time.Millisecond

	vm.StartVotingWithChoices(questionID, choiceIDs, choiceObjects, "What should we do?", duration, nil)

	if !vm.IsVotingActive() {
		t.Error("voting should be active")
	}

	results := vm.GetResults(questionID)
	if len(results) != 2 {
		t.Errorf("got %d choices, want 2", len(results))
	}

	vm.EndVoting() // Stop timer to prevent it firing after test
}

func TestSubmitVote(t *testing.T) {
	vm := NewVoteManager()
	go vm.Run()
	defer close(vm.broadcast)

	questionID := "test-question"
	choices := []string{"choice-a", "choice-b"}
	duration := 1 * time.Second

	vm.StartVoting(questionID, choices, duration, nil)

	// Submit votes
	tests := []struct {
		voterID  string
		choiceID string
	}{
		{"voter-1", "choice-a"},
		{"voter-2", "choice-a"},
		{"voter-3", "choice-b"},
	}

	for _, tt := range tests {
		if err := vm.SubmitVote(tt.voterID, tt.choiceID); err != nil {
			t.Errorf("SubmitVote failed: %v", err)
		}
	}

	results := vm.GetResults(questionID)
	if results["choice-a"] != 2 {
		t.Errorf("choice-a votes = %d, want 2", results["choice-a"])
	}
	if results["choice-b"] != 1 {
		t.Errorf("choice-b votes = %d, want 1", results["choice-b"])
	}
}

func TestSubmitVote_ChangeVote(t *testing.T) {
	vm := NewVoteManager()
	go vm.Run()
	defer close(vm.broadcast)

	questionID := "test-question"
	choices := []string{"choice-a", "choice-b"}
	duration := 1 * time.Second

	vm.StartVoting(questionID, choices, duration, nil)

	// Voter changes their vote
	voterID := "voter-1"

	vm.SubmitVote(voterID, "choice-a")
	results := vm.GetResults(questionID)
	if results["choice-a"] != 1 {
		t.Errorf("choice-a votes = %d, want 1", results["choice-a"])
	}

	vm.SubmitVote(voterID, "choice-b")
	results = vm.GetResults(questionID)
	if results["choice-a"] != 0 {
		t.Errorf("choice-a votes = %d, want 0 (vote changed)", results["choice-a"])
	}
	if results["choice-b"] != 1 {
		t.Errorf("choice-b votes = %d, want 1", results["choice-b"])
	}
}

func TestSubmitVote_WhenInactive(t *testing.T) {
	vm := NewVoteManager()

	// Try to vote without starting voting
	err := vm.SubmitVote("voter-1", "choice-a")
	if err != nil {
		t.Errorf("SubmitVote should not return error when inactive: %v", err)
	}

	// Verify no votes were recorded
	results := vm.GetResults("any-question")
	if len(results) > 0 {
		t.Error("no votes should be recorded when voting is inactive")
	}
}

func TestEndVoting(t *testing.T) {
	vm := NewVoteManager()
	go vm.Run()
	defer close(vm.broadcast)

	questionID := "test-question"
	choices := []string{"choice-a", "choice-b"}
	duration := 10 * time.Second // Long duration, we'll end manually

	var resultsMu sync.Mutex
	var finalResults map[string]int
	var finalWinner string

	vm.StartVoting(questionID, choices, duration, func(results map[string]int, winner string) {
		resultsMu.Lock()
		finalResults = results
		finalWinner = winner
		resultsMu.Unlock()
	})

	// Submit votes
	vm.SubmitVote("voter-1", "choice-a")
	vm.SubmitVote("voter-2", "choice-a")
	vm.SubmitVote("voter-3", "choice-b")

	// End voting manually
	vm.EndVoting()

	// Wait briefly for callback
	time.Sleep(10 * time.Millisecond)

	if vm.IsVotingActive() {
		t.Error("voting should be inactive after EndVoting")
	}

	resultsMu.Lock()
	if finalWinner != "choice-a" {
		t.Errorf("winner = %q, want %q", finalWinner, "choice-a")
	}
	if finalResults["choice-a"] != 2 {
		t.Errorf("choice-a votes = %d, want 2", finalResults["choice-a"])
	}
	resultsMu.Unlock()
}

func TestDetermineWinner(t *testing.T) {
	vm := NewVoteManager()

	tests := []struct {
		name       string
		results    map[string]int
		wantWinner string
	}{
		{
			name:       "clear winner",
			results:    map[string]int{"a": 5, "b": 2, "c": 1},
			wantWinner: "a",
		},
		{
			name:       "tie - first in map wins",
			results:    map[string]int{"a": 3, "b": 3},
			wantWinner: "", // Could be either, depends on map iteration
		},
		{
			name:       "no votes",
			results:    map[string]int{"a": 0, "b": 0},
			wantWinner: "",
		},
		{
			name:       "single choice",
			results:    map[string]int{"only": 10},
			wantWinner: "only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			winner := vm.determineWinner(tt.results)
			if tt.name == "tie - first in map wins" {
				// For ties, just verify it's one of the tied choices
				if winner != "a" && winner != "b" && winner != "" {
					t.Errorf("winner = %q, want 'a', 'b', or ''", winner)
				}
			} else if winner != tt.wantWinner {
				t.Errorf("winner = %q, want %q", winner, tt.wantWinner)
			}
		})
	}
}

func TestGetResults(t *testing.T) {
	vm := NewVoteManager()
	go vm.Run()
	defer close(vm.broadcast)

	questionID := "test-question"
	choices := []string{"choice-a", "choice-b"}

	vm.StartVoting(questionID, choices, 1*time.Second, nil)
	vm.SubmitVote("voter-1", "choice-a")
	vm.SubmitVote("voter-2", "choice-a")

	results := vm.GetResults(questionID)

	if results["choice-a"] != 2 {
		t.Errorf("choice-a = %d, want 2", results["choice-a"])
	}

	// Results should be a copy (mutations shouldn't affect internal state)
	results["choice-a"] = 100
	newResults := vm.GetResults(questionID)
	if newResults["choice-a"] != 2 {
		t.Error("GetResults should return a copy, not the original map")
	}
}

func TestResetVoting(t *testing.T) {
	vm := NewVoteManager()
	go vm.Run()
	defer close(vm.broadcast)

	// Start voting and submit some votes
	vm.StartVoting("q1", []string{"a", "b"}, 1*time.Second, nil)
	vm.SubmitVote("voter-1", "a")

	// Reset
	vm.ResetVoting()

	if vm.IsVotingActive() {
		t.Error("voting should be inactive after reset")
	}

	vm.mu.RLock()
	if vm.currentQuestion != "" {
		t.Errorf("currentQuestion = %q, want empty", vm.currentQuestion)
	}
	if len(vm.voters) != 0 {
		t.Errorf("voters map should be empty, got %d entries", len(vm.voters))
	}
	if len(vm.votes) != 0 {
		t.Errorf("votes map should be empty, got %d entries", len(vm.votes))
	}
	vm.mu.RUnlock()
}

func TestClearQuestionVotes(t *testing.T) {
	vm := NewVoteManager()
	go vm.Run()
	defer close(vm.broadcast)

	// Create votes for multiple questions
	vm.StartVoting("q1", []string{"a", "b"}, 1*time.Second, nil)
	vm.SubmitVote("voter-1", "a")
	vm.EndVoting()

	vm.StartVoting("q2", []string{"c", "d"}, 1*time.Second, nil)
	vm.SubmitVote("voter-2", "c")
	vm.EndVoting()

	// Clear only q1
	vm.ClearQuestionVotes("q1")

	vm.mu.RLock()
	if _, exists := vm.votes["q1"]; exists {
		t.Error("q1 votes should be cleared")
	}
	if _, exists := vm.votes["q2"]; !exists {
		t.Error("q2 votes should still exist")
	}
	vm.mu.RUnlock()
}

func TestHandleVoteMessage(t *testing.T) {
	vm := NewVoteManager()
	go vm.Run()
	defer close(vm.broadcast)

	vm.StartVoting("test-q", []string{"a", "b"}, 1*time.Second, nil)

	tests := []struct {
		name    string
		message string
		wantErr bool
	}{
		{
			name:    "valid vote message",
			message: `{"type":"vote","voter_id":"voter-1","choice_id":"a"}`,
			wantErr: false,
		},
		{
			name:    "invalid json",
			message: `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "non-vote message",
			message: `{"type":"other","data":"something"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := vm.HandleVoteMessage([]byte(tt.message))
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleVoteMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Verify vote was recorded
	results := vm.GetResults("test-q")
	if results["a"] != 1 {
		t.Errorf("vote should have been recorded for choice 'a', got %d", results["a"])
	}
}

func TestConcurrentVoting(t *testing.T) {
	vm := NewVoteManager()
	go vm.Run()
	defer close(vm.broadcast)

	questionID := "concurrent-test"
	choices := []string{"a", "b", "c"}
	vm.StartVoting(questionID, choices, 2*time.Second, nil)

	// Submit votes concurrently
	var wg sync.WaitGroup
	numVoters := 100

	for i := 0; i < numVoters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			voterID := string(rune('0' + (id % 10)))
			choiceID := choices[id%len(choices)]
			vm.SubmitVote(voterID, choiceID)
		}(i)
	}

	wg.Wait()

	// Verify results
	results := vm.GetResults(questionID)
	totalVotes := 0
	for _, count := range results {
		totalVotes += count
	}

	// Each unique voter ID should have exactly one vote
	// We have 10 unique voter IDs (0-9)
	if totalVotes != 10 {
		t.Errorf("total votes = %d, want 10 (one per unique voter)", totalVotes)
	}
}

func TestBroadcastMessage(t *testing.T) {
	vm := NewVoteManager()
	go vm.Run()

	// Create a mock client channel
	received := make(chan *Message, 1)

	// Send a broadcast
	go func() {
		msg := <-vm.broadcast
		received <- msg
	}()

	vm.BroadcastMessage("test_event", map[string]any{
		"key": "value",
	})

	select {
	case msg := <-received:
		if msg.Type != "test_event" {
			t.Errorf("message type = %q, want %q", msg.Type, "test_event")
		}
		if msg.Payload["key"] != "value" {
			t.Errorf("payload[key] = %v, want %q", msg.Payload["key"], "value")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for broadcast message")
	}
}

func TestMessageSerialization(t *testing.T) {
	msg := &Message{
		Type: "test",
		Payload: map[string]any{
			"string": "value",
			"number": 42,
			"bool":   true,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal message: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	if decoded.Type != msg.Type {
		t.Errorf("type = %q, want %q", decoded.Type, msg.Type)
	}

	if decoded.Payload["string"] != "value" {
		t.Error("payload not correctly serialized/deserialized")
	}
}
