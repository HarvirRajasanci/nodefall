package main

import (
	"context"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/google/uuid"

	"nodefall/shared/genproto"
)

// MinPlayersPerMatch is how many queued players are needed before a
// match is formed. Kept small and unexported here rather than in
// shared/config, since it's specific to matchmaker's own logic, not
// something other services need to know about.
const MinPlayersPerMatch = 2

// QueueEntry tracks one player waiting for a match, and — once a match
// is found — the details they need to connect.
type QueueEntry struct {
	PlayerID   string
	MatchID    string // empty until matched
	ServerAddr string // empty until matched
}

// Queue holds players waiting for a match and periodically groups them
// together, calling the game service's StartMatch over gRPC once
// enough are waiting.
type Queue struct {
	mu      sync.Mutex
	waiting []string               // player IDs waiting, in join order
	matched map[string]*QueueEntry // playerID -> match details, once formed

	gameClient genproto.GameServiceClient
}

// NewQueue dials the game service's gRPC address and returns a Queue
// ready to use.
func NewQueue(gameGRPCAddr string) (*Queue, error) {
	conn, err := grpc.NewClient(gameGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Queue{
		matched:    make(map[string]*QueueEntry),
		gameClient: genproto.NewGameServiceClient(conn),
	}, nil
}

// Join adds playerID to the waiting queue, unless they're already
// waiting or already matched.
func (q *Queue) Join(playerID string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, alreadyMatched := q.matched[playerID]; alreadyMatched {
		return
	}
	for _, id := range q.waiting {
		if id == playerID {
			return
		}
	}

	q.waiting = append(q.waiting, playerID)
}

// Leave removes playerID from the waiting queue. Has no effect if
// they've already been matched — once a match is formed, leaving the
// queue doesn't cancel it.
func (q *Queue) Leave(playerID string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, id := range q.waiting {
		if id == playerID {
			q.waiting = append(q.waiting[:i], q.waiting[i+1:]...)
			return
		}
	}
}

// Status returns playerID's current queue state: "waiting" if still
// in the queue, "matched" with match details if a match has been
// formed for them, or "not_queued" if neither.
func (q *Queue) Status(playerID string) (status string, entry *QueueEntry) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if e, ok := q.matched[playerID]; ok {
		return "matched", e
	}
	for _, id := range q.waiting {
		if id == playerID {
			return "waiting", nil
		}
	}
	return "not_queued", nil
}

// Run periodically checks the waiting queue and forms matches once
// enough players are waiting. Blocks until ctx is cancelled — intended
// to run on its own goroutine.
func (q *Queue) Run(ctx context.Context, tick <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			q.tryFormMatch(ctx)
		}
	}
}

func (q *Queue) tryFormMatch(ctx context.Context) {
	q.mu.Lock()
	if len(q.waiting) < MinPlayersPerMatch {
		q.mu.Unlock()
		return
	}

	players := append([]string(nil), q.waiting[:MinPlayersPerMatch]...)
	q.waiting = q.waiting[MinPlayersPerMatch:]
	q.mu.Unlock()

	matchID := uuid.New().String()

	resp, err := q.gameClient.StartMatch(ctx, &genproto.MatchRequest{
		MatchId:   matchID,
		PlayerIds: players,
	})
	if err != nil || !resp.Accepted {
		log.Printf("StartMatch failed for %v: %v", players, err)
		// Put the players back at the front of the queue to retry.
		q.mu.Lock()
		q.waiting = append(players, q.waiting...)
		q.mu.Unlock()
		return
	}

	q.mu.Lock()
	for _, id := range players {
		q.matched[id] = &QueueEntry{
			PlayerID:   id,
			MatchID:    matchID,
			ServerAddr: resp.ServerAddr,
		}
	}
	q.mu.Unlock()

	log.Printf("match %s formed with players %v on %s", matchID, players, resp.ServerAddr)
}
