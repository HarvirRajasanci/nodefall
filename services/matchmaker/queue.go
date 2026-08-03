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
	"nodefall/shared/registry"
)

const MinPlayersPerMatch = 2
const startMatchTimeout = 2 * time.Second

type QueueEntry struct {
	PlayerID   string
	MatchID    string
	ServerAddr string
}

// Queue holds players waiting for a match and periodically groups them
// together, discovering currently-live game instances from a Redis-
// backed registry.Registry on every matching pass and trying each in
// order until one accepts the match — so newly-started instances are
// picked up automatically, and instances that have crashed (missed
// their heartbeat and expired from the registry) are never selected
// at all, with no explicit failure-detection code needed here.
type Queue struct {
	mu       sync.Mutex
	waiting  []string
	matched  map[string]*QueueEntry
	registry *registry.Registry
}

// NewQueue creates a Queue that discovers game instances via reg.
func NewQueue(reg *registry.Registry) *Queue {
	return &Queue{
		matched:  make(map[string]*QueueEntry),
		registry: reg,
	}
}

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

// tryFormMatch discovers currently-live instances from the registry
// and tries each in order until one accepts the match. If none accept
// (including if none are currently registered at all), the players
// are put back in the queue to retry on the next tick.
func (q *Queue) tryFormMatch(ctx context.Context) {
	q.mu.Lock()
	if len(q.waiting) < MinPlayersPerMatch {
		q.mu.Unlock()
		return
	}

	players := append([]string(nil), q.waiting[:MinPlayersPerMatch]...)
	q.waiting = q.waiting[MinPlayersPerMatch:]
	q.mu.Unlock()

	instances, err := q.registry.ListLive(ctx)
	if err != nil || len(instances) == 0 {
		log.Printf("no live game instances available for %v: %v", players, err)
		q.requeue(players)
		return
	}

	matchID := uuid.New().String()

	for _, inst := range instances {
		conn, err := grpc.NewClient(inst.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("dialing %s failed: %v", inst.GRPCAddr, err)
			continue
		}
		client := genproto.NewGameServiceClient(conn)

		callCtx, cancel := context.WithTimeout(ctx, startMatchTimeout)
		resp, err := client.StartMatch(callCtx, &genproto.MatchRequest{
			MatchId:   matchID,
			PlayerIds: players,
		})
		cancel()

		if err != nil || !resp.Accepted {
			log.Printf("StartMatch on %s failed for %v: %v", inst.ID, players, err)
			continue
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
		return
	}

	log.Printf("all %d live instances rejected match for %v", len(instances), players)
	q.requeue(players)
}

func (q *Queue) requeue(players []string) {
	q.mu.Lock()
	q.waiting = append(players, q.waiting...)
	q.mu.Unlock()
}
