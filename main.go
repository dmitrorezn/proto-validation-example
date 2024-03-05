package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bufbuild/protovalidate-go"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/dmitrorezn/proto-validation-example/gen/apis/processor"
)

type Cfg struct {
	Port string
}

func NewCfg() *Cfg {
	return &Cfg{
		Port: "8080",
	}
}

func (c Cfg) WithPort(p string) Cfg {
	c.Port = p

	return c
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := NewCfg()

	service, err := newProcessorSvc(ctx)
	if err != nil {
		panic(err)
	}
	lis, err := net.Listen("tcp", net.JoinHostPort("", cfg.Port))
	if err != nil {
		panic(err)
	}
	gr, ctx := errgroup.WithContext(ctx)
	defer func() {
		if err = gr.Wait(); err != nil {
			panic(err)
		}
	}()

	server := grpc.NewServer()
	defer server.GracefulStop()
	processor.RegisterProcessorServiceServer(server, service)

	gr.Go(func() error {
		return server.Serve(lis)
	})

	slog.Info("STARTED")
	defer slog.Info("STOPPED")
	<-ctx.Done()
}

var _ processor.ProcessorServiceServer = (*processorSvc)(nil)

type processorSvc struct {
	processor.UnimplementedProcessorServiceServer

	ctx        context.Context
	mu         sync.RWMutex
	cache      *safehmap[uuid.UUID, string]
	validation *protovalidate.Validator
	replicator *replicator[uuid.UUID]
}

type replicator[T any] struct {
	sync.Mutex
	consumers []chan T
	c         chan T
}

func newReplicator[T any](ctx context.Context) *replicator[T] {
	r := &replicator[T]{
		consumers: make([]chan T, 0),
		c:         make(chan T),
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case item := <-r.c:
				for _, consumer := range r.consumers {
					select {
					case consumer <- item:
					case <-time.After(10 * time.Millisecond):
						continue
					}
				}
			}
		}
	}()

	return r
}

func (r *replicator[T]) consume() chan T {
	consumer := make(chan T)

	defer lockUnlock(r)
	r.consumers = append(r.consumers, consumer)

	return consumer
}

func (r *replicator[T]) produce(item T) chan struct{} {
	wait := make(chan struct{})
	go func() {
		defer close(wait)
		r.c <- item
	}()

	return wait
}

func (p *processorSvc) Consume(_ *processor.ConsumeRequest, server processor.ProcessorService_ConsumeServer) error {
	for {
		select {
		case id := <-p.replicator.consume():
			if err := server.Send(&processor.ConsumeResponse{
				Id: id.String(),
			}); err != nil {
				return err
			}
		case <-p.ctx.Done():
			return p.ctx.Err()
		}
	}
}

func newProcessorSvc(ctx context.Context) (*processorSvc, error) {
	validator, err := protovalidate.New(
		protovalidate.WithMessages(
			new(processor.ProcessRequest),
		),
	)
	if err != nil {
		return nil, err
	}
	return &processorSvc{
		ctx:        ctx,
		replicator: newReplicator[uuid.UUID](ctx),
		cache:      newSafe[uuid.UUID, string](),
		validation: validator,
	}, nil
}

func (p *processorSvc) ProcessStream(server processor.ProcessorService_ProcessStreamServer) error {
	ctx := server.Context()

	var response *processor.ProcessResponse
	for {
		request, err := server.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			slog.Error("ProcessStream -> Recv", err)

			continue
		}
		if response, err = p.Process(ctx, request); err != nil {
			return err
		}
		if err = server.Send(response); err != nil {
			return err
		}
	}
}

func (p *processorSvc) putName(ctx context.Context, name string) (uuid.UUID, error) {
	id := uuid.New()
	p.cache.Put(id, name)

	select {
	case <-ctx.Done():
		return id, ctx.Err()
	case <-p.replicator.produce(id):
	}

	return id, nil
}

func (p *processorSvc) Process(ctx context.Context, request *processor.ProcessRequest) (*processor.ProcessResponse, error) {
	if err := p.validation.Validate(request); err != nil {
		return nil, err
	}
	id, err := p.putName(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	return &processor.ProcessResponse{
		Id: id.String(),
	}, nil
}
