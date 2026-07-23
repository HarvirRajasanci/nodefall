package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMatchReadyEvent_RoundTripsThroughJSON(t *testing.T) {
	original := MatchReadyEvent{
		MatchID:    "match-abc",
		PlayerIDs:  []string{"player-1", "player-2"},
		ServerAddr: "game-server-3:9000",
		StartedAt:  time.Now().UTC().Truncate(time.Second),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var decoded MatchReadyEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if decoded.MatchID != original.MatchID {
		t.Errorf("got MatchID %q, want %q", decoded.MatchID, original.MatchID)
	}
	if len(decoded.PlayerIDs) != 2 || decoded.PlayerIDs[0] != "player-1" {
		t.Errorf("got PlayerIDs %v, want %v", decoded.PlayerIDs, original.PlayerIDs)
	}
	if !decoded.StartedAt.Equal(original.StartedAt) {
		t.Errorf("got StartedAt %v, want %v", decoded.StartedAt, original.StartedAt)
	}
}

func TestMatchResultEvent_RoundTripsThroughJSON(t *testing.T) {
	original := MatchResultEvent{
		MatchID:   "match-abc",
		WinnerID:  "player-1",
		PlayerIDs: []string{"player-1", "player-2"},
		EndedAt:   time.Now().UTC().Truncate(time.Second),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	var decoded MatchResultEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if decoded.WinnerID != original.WinnerID {
		t.Errorf("got WinnerID %q, want %q", decoded.WinnerID, original.WinnerID)
	}
	if !decoded.EndedAt.Equal(original.EndedAt) {
		t.Errorf("got EndedAt %v, want %v", decoded.EndedAt, original.EndedAt)
	}
}
