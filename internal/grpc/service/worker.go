package service

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"taskqueue/internal/grpc/proto"
	"taskqueue/internal/models"
	"taskqueue/internal/storage"
)

// WorkerService implements the proto.WorkerServiceServer scaffold.
type WorkerService struct {
	proto.UnimplementedWorkerServiceServer
	store storage.Storage
}

// NewWorkerService creates a WorkerService backed by storage.
func NewWorkerService(store storage.Storage) *WorkerService {
	return &WorkerService{store: store}
}

func (s *WorkerService) RegisterWorker(ctx context.Context, req *proto.RegisterWorkerRequest) (*proto.RegisterWorkerResponse, error) {
	return nil, status.Error(codes.Unimplemented, "RegisterWorker not implemented")
}

func (s *WorkerService) Heartbeat(ctx context.Context, req *proto.HeartbeatRequest) (*proto.HeartbeatResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Heartbeat not implemented")
}

func (s *WorkerService) GetWorkerStatus(ctx context.Context, req *proto.GetWorkerStatusRequest) (*proto.Worker, error) {
	workers, err := s.store.ListWorkers(ctx)
	if err != nil {
		return nil, err
	}
	for _, w := range workers {
		if w.ID == req.GetWorkerId() {
			return convertWorker(w), nil
		}
	}
	return nil, status.Error(codes.NotFound, "worker not found")
}

func (s *WorkerService) ListWorkers(ctx context.Context, req *proto.ListWorkersRequest) (*proto.ListWorkersResponse, error) {
	workers, err := s.store.ListWorkers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*proto.Worker, 0, len(workers))
	for _, worker := range workers {
		out = append(out, convertWorker(worker))
	}
	return &proto.ListWorkersResponse{
		Workers: out,
		Total:   int32(len(out)),
	}, nil
}

func (s *WorkerService) DeregisterWorker(ctx context.Context, req *proto.DeregisterWorkerRequest) (*proto.DeregisterWorkerResponse, error) {
	return nil, status.Error(codes.Unimplemented, "DeregisterWorker not implemented")
}

func convertWorker(info models.WorkerInfo) *proto.Worker {
	return &proto.Worker{
		Id:            info.ID,
		Hostname:      info.Hostname,
		Status:        toProtoWorkerStatus(info.Status),
		PoolSize:      int32(info.PoolSize),
		ActiveJobs:    int32(info.ActiveJobs),
		TotalProcessed: int64(info.TotalJobs),
		StartedAt:     timestamppb.New(info.StartedAt),
		LastHeartbeat: timestamppb.New(info.LastHeartbeat),
	}
}

func toProtoWorkerStatus(status string) proto.WorkerStatus {
	switch strings.ToLower(status) {
	case "active", "busy":
		return proto.WorkerStatus_WORKER_STATUS_ACTIVE
	case "idle":
		return proto.WorkerStatus_WORKER_STATUS_IDLE
	case "draining":
		return proto.WorkerStatus_WORKER_STATUS_DRAINING
	default:
		return proto.WorkerStatus_WORKER_STATUS_UNSPECIFIED
	}
}

