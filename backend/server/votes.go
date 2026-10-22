package server

import (
	"encoding/json"
	"log"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/skarlso/kube_adventures/voting/backend/parser"
)

// VoteManager handles vote aggregation and broadcasting.
//
// Lock order is mu before clientsMu. State changes broadcast while holding mu,
// so a client registered under mu can never miss or double-see a message.
type VoteManager struct {
	mu              sync.RWMutex
	currentQuestion string
	choiceOrder     []string                  // declaration order of the current question's choices
	votes           map[string]map[string]int // questionID -> choiceID -> count
	voters          map[string]string         // voterID -> choiceID (for current question)
	clientsMu       sync.Mutex
	clients         map[*client]struct{}
	timer           *time.Timer
	timerDuration   time.Duration
	votingActive    bool
	onVoteComplete  func(results map[string]int, winner string)
}

// Message represents a WebSocket message.
type Message struct {
	Type    string         `json:"type"` // vote, results, state, timer, etc.
	Payload map[string]any `json:"payload"`
}

// NewVoteManager creates a new vote manager.
func NewVoteManager() *VoteManager {
	return &VoteManager{
		votes:   make(map[string]map[string]int),
		voters:  make(map[string]string),
		clients: make(map[*client]struct{}),
	}
}

// broadcast queues a message for every client without blocking. A client whose
// queue is full is dropped instead of stalling everyone else; it reconnects.
func (vm *VoteManager) broadcast(message *Message) {
	vm.clientsMu.Lock()
	defer vm.clientsMu.Unlock()

	for c := range vm.clients {
		select {
		case c.send <- message:
		default:
			log.Printf("Dropping websocket client: send queue full")
			delete(vm.clients, c)
			close(c.send)
		}
	}
}

// StartVoting begins a new voting session.
func (vm *VoteManager) StartVoting(questionID string, choices []string, duration time.Duration, onComplete func(map[string]int, string)) {
	vm.StartVotingWithChoices(questionID, choices, nil, "", duration, onComplete)
}

// StartVotingWithChoices begins a new voting session with full choice metadata.
func (vm *VoteManager) StartVotingWithChoices(questionID string, choiceIDs []string, choiceObjects []parser.Choice, question string, duration time.Duration, onComplete func(map[string]int, string)) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	// reset state
	vm.currentQuestion = questionID
	vm.choiceOrder = slices.Clone(choiceIDs)
	vm.voters = make(map[string]string)
	vm.votingActive = true
	vm.timerDuration = duration
	vm.onVoteComplete = onComplete

	vm.votes[questionID] = make(map[string]int)
	for _, choice := range choiceIDs {
		vm.votes[questionID][choice] = 0
	}

	if vm.timer != nil {
		vm.timer.Stop()
	}

	vm.timer = time.AfterFunc(duration, func() {
		vm.EndVoting()
	})

	payload := map[string]any{
		"question_id": questionID,
		"duration":    duration.Seconds(),
	}

	if question != "" {
		payload["question"] = question
	}

	if len(choiceObjects) > 0 {
		payload["choices"] = choiceObjects
	} else {
		payload["choices"] = choiceIDs
	}

	vm.broadcast(&Message{
		Type:    "voting_started",
		Payload: payload,
	})
}

// SubmitVote records a vote from a user.
func (vm *VoteManager) SubmitVote(voterID, choiceID string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.votingActive {
		return nil
	}

	if previousChoice, hasVoted := vm.voters[voterID]; hasVoted {
		if vm.votes[vm.currentQuestion] != nil {
			vm.votes[vm.currentQuestion][previousChoice]--
		}
	}

	vm.voters[voterID] = choiceID
	if vm.votes[vm.currentQuestion] == nil {
		vm.votes[vm.currentQuestion] = make(map[string]int)
	}

	vm.votes[vm.currentQuestion][choiceID]++

	vm.broadcastResults()

	return nil
}

// EndVoting stops the current voting session and determines the winner.
func (vm *VoteManager) EndVoting() {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.votingActive {
		return
	}

	vm.votingActive = false

	if vm.timer != nil {
		vm.timer.Stop()
	}

	results := vm.votes[vm.currentQuestion]
	winner := vm.determineWinner(results)

	vm.broadcast(&Message{
		Type: "voting_ended",
		Payload: map[string]any{
			"question_id": vm.currentQuestion,
			"results":     results,
			"winner":      winner,
		},
	})

	if vm.onVoteComplete != nil {
		go vm.onVoteComplete(results, winner)
	}
}

// StopVoting disarms an in-flight voting session without clearing the tally or
// broadcasting. Used when the presenter advances manually before the timer
// expires, so the pending AfterFunc cannot fire "voting_ended" for the previous
// question on top of the next chapter.
func (vm *VoteManager) StopVoting() {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.timer != nil {
		vm.timer.Stop()
		vm.timer = nil
	}

	vm.votingActive = false
	vm.choiceOrder = nil
	vm.onVoteComplete = nil
}

// determineWinner finds the choice with the most votes. Iteration follows the
// order the choices were declared in the chapter so a tie always resolves to
// the same path, and so choice IDs a client invented are never picked. Falls
// back to sorted keys when no question is in flight.
func (vm *VoteManager) determineWinner(results map[string]int) string {
	order := vm.choiceOrder
	if len(order) == 0 {
		order = slices.Sorted(maps.Keys(results))
	}

	maxVotes := 0
	winner := ""

	for _, choiceID := range order {
		if count := results[choiceID]; count > maxVotes {
			maxVotes = count
			winner = choiceID
		}
	}

	return winner
}

// broadcastResults sends current vote counts to all clients.
func (vm *VoteManager) broadcastResults() {
	results := make(map[string]int)

	if vm.votes[vm.currentQuestion] != nil {
		maps.Copy(results, vm.votes[vm.currentQuestion])
	}

	vm.broadcast(&Message{
		Type: "vote_update",
		Payload: map[string]any{
			"question_id": vm.currentQuestion,
			"results":     results,
			"total":       len(vm.voters),
		},
	})
}

// stateMessage snapshots the voting state for a newly connected client.
// Callers must hold vm.mu.
func (vm *VoteManager) stateMessage() *Message {
	state := map[string]any{
		"voting_active": vm.votingActive,
		"question_id":   vm.currentQuestion,
	}

	// cloned because the writer serializes it later, outside the lock
	if vm.votingActive && vm.votes[vm.currentQuestion] != nil {
		state["results"] = maps.Clone(vm.votes[vm.currentQuestion])
		state["total"] = len(vm.voters)
	}

	return &Message{
		Type:    "state",
		Payload: state,
	}
}

// GetResults returns the current vote counts.
func (vm *VoteManager) GetResults(questionID string) map[string]int {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	results := make(map[string]int)

	if vm.votes[questionID] != nil {
		maps.Copy(results, vm.votes[questionID])
	}

	return results
}

// RegisterClient adds a WebSocket client and queues the current state for it.
func (vm *VoteManager) RegisterClient(conn *websocket.Conn) *client {
	c := newClient(conn)

	vm.mu.RLock()
	defer vm.mu.RUnlock()

	vm.clientsMu.Lock()
	defer vm.clientsMu.Unlock()

	c.send <- vm.stateMessage()

	vm.clients[c] = struct{}{}

	return c
}

// UnregisterClient removes a WebSocket client. Safe to call after the client
// was already dropped by broadcast.
func (vm *VoteManager) UnregisterClient(c *client) {
	vm.clientsMu.Lock()
	defer vm.clientsMu.Unlock()

	if _, ok := vm.clients[c]; ok {
		delete(vm.clients, c)
		close(c.send)
	}
}

// BroadcastMessage sends a custom message to all clients.
func (vm *VoteManager) BroadcastMessage(msgType string, payload map[string]any) {
	vm.broadcast(&Message{
		Type:    msgType,
		Payload: payload,
	})
}

// IsVotingActive returns whether voting is currently active.
func (vm *VoteManager) IsVotingActive() bool {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	return vm.votingActive
}

// VoteMessage represents an incoming vote.
type VoteMessage struct {
	Type     string `json:"type"`
	VoterID  string `json:"voter_id"`
	ChoiceID string `json:"choice_id"`
}

// HandleVoteMessage processes incoming vote messages.
func (vm *VoteManager) HandleVoteMessage(data []byte) error {
	var msg VoteMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	if msg.Type == "vote" {
		return vm.SubmitVote(msg.VoterID, msg.ChoiceID)
	}

	return nil
}

// ResetVoting clears all voting state.
func (vm *VoteManager) ResetVoting() {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.timer != nil {
		vm.timer.Stop()
		vm.timer = nil
	}

	vm.votingActive = false
	vm.currentQuestion = ""
	vm.choiceOrder = nil
	vm.voters = make(map[string]string)
	// clear the history
	vm.votes = make(map[string]map[string]int)
	vm.onVoteComplete = nil

	vm.broadcast(&Message{
		Type: "voting_reset",
		Payload: map[string]any{
			"status": "reset",
		},
	})
}

// ClearQuestionVotes clears votes for a specific question only.
func (vm *VoteManager) ClearQuestionVotes(questionID string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.timer != nil {
		vm.timer.Stop()
		vm.timer = nil
	}

	vm.votingActive = false
	vm.currentQuestion = ""
	vm.choiceOrder = nil
	vm.voters = make(map[string]string)

	if questionID != "" {
		delete(vm.votes, questionID)
	}

	vm.onVoteComplete = nil

	vm.broadcast(&Message{
		Type: "voting_reset",
		Payload: map[string]any{
			"status": "reset",
		},
	})
}
