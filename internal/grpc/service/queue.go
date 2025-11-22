package service

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"taskqueue/internal/broker"
	"taskqueue/internal/grpc/proto"
)

// QueueDepthReporter exposes queue metrics for the QueueService.
type QueueDepthReporter interface {
	Depth() int
}

// QueueService implements proto.QueueServiceServer.
type QueueService struct {
	proto.UnimplementedQueueServiceServer
	queueName string
	queue     broker.Broker
	depth     QueueDepthReporter
}

// NewQueueService scaffolds a queue service using the broker implementation.
func NewQueueService(name string, q broker.Broker) *QueueService {
	reporter, _ := q.(QueueDepthReporter)
	return &QueueService{
		queueName: name,
		queue:     q,
		depth:     reporter,
	}
}

func (s *QueueService) GetQueueStats(ctx context.Context, req *proto.GetQueueStatsRequest) (*proto.QueueStats, error) {
	return s.stats(), nil
}

func (s *QueueService) ListQueues(ctx context.Context, req *proto.ListQueuesRequest) (*proto.ListQueuesResponse, error) {
	return &proto.ListQueuesResponse{
		Queues: []*proto.QueueStats{s.stats()},
	}, nil
}

func (s *QueueService) PurgeQueue(ctx context.Context, req *proto.PurgeQueueRequest) (*proto.PurgeQueueResponse, error) {
	// Not implemented yet; respond with empty success flag.
	return &proto.PurgeQueueResponse{Success: false}, nil
}

func (s *QueueService) stats() *proto.QueueStats {
	depth := int64(-1)
	if s.depth != nil {
		depth = int64(s.depth.Depth())
	}
	return &proto.QueueStats{
		Name:           s.queueName,
		Depth:          depth,
		ProcessingRate: 0,
		AvgLatencyMs:   0,
		TotalProcessed: 0,
		TotalFailed:    0,
		LastUpdated:    timestamppb.New(time.Now()),
	}
}

