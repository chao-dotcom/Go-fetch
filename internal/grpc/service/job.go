package service

import (
	"context"
	"encoding/json"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"taskqueue/internal/grpc/proto"
	"taskqueue/internal/models"
	"taskqueue/internal/storage"
)

// JobService implements proto.JobServiceServer using existing storage/broker logic.
type JobService struct {
	proto.UnimplementedJobServiceServer
	store storage.Storage
}

// NewJobService scaffolds a JobService with the given dependencies.
func NewJobService(store storage.Storage) *JobService {
	return &JobService{store: store}
}

func (s *JobService) SubmitJob(ctx context.Context, req *proto.SubmitJobRequest) (*proto.SubmitJobResponse, error) {
	return nil, status.Error(codes.Unimplemented, "SubmitJob not wired yet")
}

func (s *JobService) GetJobStatus(ctx context.Context, req *proto.GetJobStatusRequest) (*proto.Job, error) {
	job, err := s.store.GetJob(ctx, req.GetJobId())
	if err != nil {
		return nil, err
	}
	return convertJob(job), nil
}

func (s *JobService) CancelJob(ctx context.Context, req *proto.CancelJobRequest) (*proto.CancelJobResponse, error) {
	return nil, status.Error(codes.Unimplemented, "CancelJob not wired yet")
}

func (s *JobService) StreamJobUpdates(req *proto.StreamJobUpdatesRequest, stream proto.JobService_StreamJobUpdatesServer) error {
	return status.Error(codes.Unimplemented, "StreamJobUpdates not implemented")
}

func (s *JobService) ListJobs(ctx context.Context, req *proto.ListJobsRequest) (*proto.ListJobsResponse, error) {
	filter := storage.ListFilter{
		Status: fromProtoStatus(req.GetStatus()),
		Type:   req.GetType(),
		Limit:  int(req.GetLimit()),
		Offset: int(req.GetOffset()),
	}
	jobs, total, err := s.store.ListJobs(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]*proto.Job, 0, len(jobs))
	for _, job := range jobs {
		out = append(out, convertJob(job))
	}
	return &proto.ListJobsResponse{
		Jobs:   out,
		Total:  int32(total),
		Limit:  req.GetLimit(),
		Offset: req.GetOffset(),
	}, nil
}

func convertJob(job *models.Job) *proto.Job {
	if job == nil {
		return nil
	}
	payload := bytesToStruct(job.Payload)
	result := bytesToStruct(job.Result)
	return &proto.Job{
		Id:               job.ID,
		Type:             job.Type,
		Payload:          payload,
		Status:           toProtoStatus(job.Status),
		Priority:         int32(job.Priority),
		Attempts:         int32(job.Attempts),
		MaxAttempts:      int32(job.MaxAttempts),
		CreatedAt:        timestamppb.New(job.CreatedAt),
		UpdatedAt:        timestamppb.New(job.UpdatedAt),
		StartedAt:        timePtr(job.StartedAt),
		FinishedAt:       timePtr(job.FinishedAt),
		Result:           result,
		Error:            job.Error,
		WorkerId:         job.WorkerID,
		ProcessingTimeMs: processingDuration(job),
	}
}

func toProtoStatus(status models.JobStatus) proto.JobStatus {
	switch status {
	case models.JobStatusQueued:
		return proto.JobStatus_JOB_STATUS_QUEUED
	case models.JobStatusRunning:
		return proto.JobStatus_JOB_STATUS_RUNNING
	case models.JobStatusSucceeded:
		return proto.JobStatus_JOB_STATUS_SUCCEEDED
	case models.JobStatusFailed:
		return proto.JobStatus_JOB_STATUS_FAILED
	case models.JobStatusRetrying:
		return proto.JobStatus_JOB_STATUS_RETRYING
	default:
		return proto.JobStatus_JOB_STATUS_UNSPECIFIED
	}
}

func fromProtoStatus(status proto.JobStatus) models.JobStatus {
	switch status {
	case proto.JobStatus_JOB_STATUS_QUEUED:
		return models.JobStatusQueued
	case proto.JobStatus_JOB_STATUS_RUNNING:
		return models.JobStatusRunning
	case proto.JobStatus_JOB_STATUS_SUCCEEDED:
		return models.JobStatusSucceeded
	case proto.JobStatus_JOB_STATUS_FAILED:
		return models.JobStatusFailed
	case proto.JobStatus_JOB_STATUS_RETRYING:
		return models.JobStatusRetrying
	default:
		return ""
	}
}

func bytesToStruct(raw []byte) *structpb.Struct {
	if len(raw) == 0 {
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil
	}
	res, err := structpb.NewStruct(data)
	if err != nil {
		return nil
	}
	return res
}

func timePtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func processingDuration(job *models.Job) int64 {
	if job.StartedAt == nil || job.FinishedAt == nil {
		return 0
	}
	return job.FinishedAt.Sub(*job.StartedAt).Milliseconds()
}
	return &proto.Job{
		Id:        job.ID,
		Type:      job.Type,
		Status:    proto.JobStatus(proto.JobStatus_value["JOB_STATUS_"+job.Status]),
		Priority:  int32(job.Priority),
		Attempts:  int32(job.Attempts),
		MaxAttempts: int32(job.MaxAttempts),
		Error:     job.Error,
		WorkerId:  job.WorkerID,
	}
}

