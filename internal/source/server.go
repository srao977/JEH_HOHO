// File: server.go
// Date: 2026-09-17
// Version/Status: 1.0 / Slice 2 implementation
// Product/Component: JEH-HOHO / AcceptedBarSourceService
// Purpose: Deliver the engine's bounded accepted stream to exactly one JEH consumer.
// Inputs/Outputs: Accepted engine observations in; ordered gRPC responses out.
// Invariants: A second concurrent consumer is rejected; sends block; no replay exists.
// Failure behavior: Cancellation ends only the subscription; closure is UNAVAILABLE.
// Non-responsibilities: Source lifecycle, buffering policy, sequencing, or JEH admission.

package source

import (
	"sync"

	jehhohov1 "jeh-hoho/gen/go/jeh_hoho/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server exposes one Engine accepted queue through the product contract.
type Server struct {
	jehhohov1.UnimplementedAcceptedBarSourceServiceServer

	accepted <-chan *jehhohov1.AcceptedBarObservation
	mu       sync.Mutex
	active   bool
}

// NewServer creates a service over one engine-owned accepted queue.
func NewServer(accepted <-chan *jehhohov1.AcceptedBarObservation) *Server {
	return &Server{accepted: accepted}
}

// StreamAcceptedBars leases the accepted queue to exactly one active JEH consumer.
func (s *Server) StreamAcceptedBars(request *jehhohov1.StreamAcceptedBarsRequest, stream jehhohov1.AcceptedBarSourceService_StreamAcceptedBarsServer) error {
	if request.GetConsumerId() == "" {
		return status.Error(codes.InvalidArgument, "consumer_id is required")
	}
	if !s.acquire() {
		return status.Error(codes.AlreadyExists, "accepted Bar consumer is already connected")
	}
	defer s.release()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case observation, ok := <-s.accepted:
			if !ok {
				return status.Error(codes.Unavailable, "accepted Bar source stopped")
			}
			if err := stream.Send(&jehhohov1.StreamAcceptedBarsResponse{Observation: observation}); err != nil {
				return err
			}
		}
	}
}

func (s *Server) acquire() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		return false
	}
	s.active = true
	return true
}

func (s *Server) release() {
	s.mu.Lock()
	s.active = false
	s.mu.Unlock()
}
