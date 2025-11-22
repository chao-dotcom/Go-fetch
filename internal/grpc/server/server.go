package server

import (
	"context"
	"net"

	"google.golang.org/grpc"

	"taskqueue/internal/broker"
	"taskqueue/internal/grpc/proto"
	"taskqueue/internal/storage"
)

// Config wires dependencies for the gRPC server bootstrap.
type Config struct {
	Addr    string
	Store   storage.Storage
	Broker  broker.Broker
	Server  proto.JobServiceServer
	Workers proto.WorkerServiceServer
	Queues  proto.QueueServiceServer
}

// Run starts a gRPC server that hosts the provided service implementations.
func Run(ctx context.Context, cfg Config) error {
	lis, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	if cfg.Server != nil {
		proto.RegisterJobServiceServer(grpcServer, cfg.Server)
	}
	if cfg.Workers != nil {
		proto.RegisterWorkerServiceServer(grpcServer, cfg.Workers)
	}
	if cfg.Queues != nil {
		proto.RegisterQueueServiceServer(grpcServer, cfg.Queues)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- grpcServer.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		grpcServer.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}

