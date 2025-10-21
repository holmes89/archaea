package base

import (
	"context"
	"log"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

type GenericGRPCService[T Entity] struct {
	svc       Service[T]
	publisher Producer[T]
}

func (s *GenericGRPCService[T]) Get(ctx context.Context, rawreq connect.AnyRequest, req GetRequest[T]) (GetResponse[T], error) {
	log.Printf("Get called - Protocol: %s", rawreq.Header().Get("Connect-Protocol-Version"))
	var t T
	log.Printf("Get %T request: %+v", t, req)

	return s.svc.Get(ctx, req)
}

func (s *GenericGRPCService[T]) List(ctx context.Context, rawreq connect.AnyRequest, req ListRequest) (ListResponse[T], error) {
	log.Printf("List called - Protocol: %s", rawreq.Header().Get("Connect-Protocol-Version"))
	var t T
	log.Printf("List %T content: %+v", t, req)

	count := req.GetCount()
	if count != req.GetCount() {
		log.Printf("Request count: %d", req.GetCount())
	} else {
		log.Printf("Request count is nil")
	}

	resp, err := s.svc.List(ctx, req)
	if err != nil {
		log.Printf("Error from service: %v", err)
		return nil, err
	}

	log.Printf("Service returned %d %T", len(resp.GetData()), t)

	return resp, nil
}

func (s *GenericGRPCService[T]) Create(ctx context.Context, rawreq connect.AnyRequest, req CreateRequest[T]) (string, error) {
	log.Printf("CreateBook called - Protocol: %s", rawreq.Header().Get("Connect-Protocol-Version"))
	var t T
	log.Printf("Create %T request: %+v", t, req)

	e := req.GetData()
	id := e.GetUuid()
	if id == "" {
		id = uuid.NewString()
	}
	log.Printf("publish create %T - %s", t, id)

	err := s.publisher.Publish(ctx, e, id, time.Now())
	if err != nil {
		log.Printf("Error publishing event: %v", err)
		return "", err
	}

	return id, nil
}

func NewGenericGRPCService[T Entity](svc Service[T], publisher Producer[T]) *GenericGRPCService[T] {
	return &GenericGRPCService[T]{
		svc:       svc,
		publisher: publisher,
	}
}
