package grpc

import (
	"context"

	"github.com/Fnuworsu/vektor/internal/cgobridge"
	"github.com/Fnuworsu/vektor/internal/coordinator/policy"
	"github.com/Fnuworsu/vektor/internal/coordinator/tracker"
	pb "github.com/Fnuworsu/vektor/proto"
	grpc "google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedControlPlaneServer
	tracker *tracker.Tracker
	policy  *policy.Engine
	engine  cgobridge.Engine
}

func RegisterServer(s *grpc.Server, t *tracker.Tracker, p *policy.Engine, e cgobridge.Engine) {
	pb.RegisterControlPlaneServer(s, &server{tracker: t, policy: p, engine: e})
}

func (s *server) GetStats(ctx context.Context, req *pb.GetStatsRequest) (*pb.GetStatsResponse, error) {
	snap := s.tracker.Snapshot()
	return &pb.GetStatsResponse{
		PrefetchIssued:   snap.PrefetchIssued,
		PrefetchHit:      snap.PrefetchHit,
		PrefetchMiss:     snap.PrefetchMiss,
		PrefetchDropped:  snap.PrefetchDropped,
		TotalGetsProxied: snap.TotalGetsProxied,
	}, nil
}

func (s *server) SetPolicy(ctx context.Context, req *pb.SetPolicyRequest) (*pb.SetPolicyResponse, error) {
	s.policy.UpdateThreshold(req.Threshold)
	return &pb.SetPolicyResponse{Success: true}, nil
}

func (s *server) GetModelState(ctx context.Context, req *pb.GetModelStateRequest) (*pb.GetModelStateResponse, error) {
	count := s.engine.GetModelState()
	return &pb.GetModelStateResponse{TrackedKeysCount: count}, nil
}
