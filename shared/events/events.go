// Package events defines the message shapes published to Redis Streams
// between Nodefall's services. Every producer and consumer of these
// events imports this package so they agree on the same wire format —
// none of them define their own ad hoc structs for this data.
package events

import "time"

// MatchReadyEvent is published by the matchmaker once it has assembled
// a match and successfully started it on a game server via gRPC.
// Consumers: currently none, but this is the seam a future stats/
// analytics service would subscribe to.
type MatchReadyEvent struct {
	MatchID    string    `json:"match_id"`
	PlayerIDs  []string  `json:"player_ids"`
	ServerAddr string    `json:"server_addr"`
	StartedAt  time.Time `json:"started_at"`
}

// MatchResultEvent is published by the game service when a match ends.
// Consumers: a future leaderboard service would subscribe to this to
// update player rankings.
type MatchResultEvent struct {
	MatchID  string    `json:"match_id"`
	WinnerID string    `json:"winner_id"`
	PlayerIDs []string `json:"player_ids"`
	EndedAt  time.Time `json:"ended_at"`
}
