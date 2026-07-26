package server

import (
	"context"
	"log"

	"nodefall/services/game/matchmanager"
	"nodefall/shared/genproto"
)

// GRPCServer implements genproto.GameServiceServer, letting the
// matchmaker request that this game server host a match for a given
// set of players.
type GRPCServer struct {
	genproto.UnimplementedGameServiceServer
	matches *matchmanager.Manager
	addr    string // this server's own WebSocket address, returned to the matchmaker
}

// NewGRPCServer creates a GRPCServer backed by the given match manager,
// reporting addr (this server's WebSocket address) in every response.
func NewGRPCServer(matches *matchmanager.Manager, addr string) *GRPCServer {
	return &GRPCServer{matches: matches, addr: addr}
}

// StartMatch creates a new, isolated match restricted to the given
// player IDs, registered under the match ID the caller supplied.
func (g *GRPCServer) StartMatch(ctx context.Context, req *genproto.MatchRequest) (*genproto.MatchResponse, error) {
	log.Printf("StartMatch: match %s with %d players", req.MatchId, len(req.PlayerIds))

	g.matches.CreateMatch(req.MatchId, req.PlayerIds)

	return &genproto.MatchResponse{
		Accepted:   true,
		ServerAddr: g.addr,
	}, nil
}
