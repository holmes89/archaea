package base

import (
	"context"
	"log"

	"connectrpc.com/connect"
)

// GenericGRPCService provides a base implementation for gRPC services
// that wraps a business logic Service and optionally publishes events
type GenericGRPCService[T Entity] struct {
	Service   Service[T]
	Publisher Producer[T]
}

// NewGenericGRPCService creates a new generic gRPC service
func NewGenericGRPCService[T Entity](svc Service[T], publisher Producer[T]) *GenericGRPCService[T] {
	return &GenericGRPCService[T]{
		Service:   svc,
		Publisher: publisher,
	}
}

// List handles list requests by delegating to the service
func (g *GenericGRPCService[T]) List(ctx context.Context, req *connect.Request[any], listReq ListRequest) (ListResponse[T], error) {
	log.Printf("Listing entities with cursor: %s, count: %d", listReq.GetCursor(), listReq.GetCount())

	resp, err := g.Service.List(ctx, listReq)
	if err != nil {
		log.Printf("Error listing entities: %v", err)
		return nil, err
	}

	log.Printf("Successfully listed %d entities", len(resp.GetData()))
	return resp, nil
}

// Get handles get requests by delegating to the service
func (g *GenericGRPCService[T]) Get(ctx context.Context, req *connect.Request[any], getReq GetRequest[T]) (GetResponse[T], error) {
	log.Printf("Getting entity with ID: %s", getReq.GetUuid())

	resp, err := g.Service.Get(ctx, getReq)
	if err != nil {
		log.Printf("Error getting entity: %v", err)
		return nil, err
	}

	log.Printf("Successfully retrieved entity")
	return resp, nil
}

// Create handles create requests by delegating to the service and publishing events
func (g *GenericGRPCService[T]) Create(ctx context.Context, req *connect.Request[any], createReq CreateRequest[T]) (string, error) {
	log.Printf("Creating new entity")

	resp, err := g.Service.Create(ctx, createReq)
	if err != nil {
		log.Printf("Error creating entity: %v", err)
		return "", err
	}

	entity := resp.GetData()

	// Publish event if publisher is configured
	if g.Publisher != nil {
		if err := g.Publisher.Publish(ctx, entity); err != nil {
			log.Printf("Warning: failed to publish create event: %v", err)
			// Don't fail the request if publishing fails
		}
	}

	log.Printf("Successfully created entity with ID: %s", entity.GetUuid())
	return entity.GetUuid(), nil
}

// Update handles update requests by delegating to the service and publishing events
func (g *GenericGRPCService[T]) Update(ctx context.Context, req *connect.Request[any], updateReq UpdateRequest[T]) error {
	log.Printf("Updating entity")

	entity := updateReq.GetData()
	if _, err := g.Service.Update(ctx, updateReq); err != nil {
		log.Printf("Error updating entity: %v", err)
		return err
	}

	// Publish event if publisher is configured
	if g.Publisher != nil {
		if err := g.Publisher.Publish(ctx, entity); err != nil {
			log.Printf("Warning: failed to publish update event: %v", err)
		}
	}

	log.Printf("Successfully updated entity")
	return nil
}

// Delete handles delete requests by delegating to the service
func (g *GenericGRPCService[T]) Delete(ctx context.Context, req *connect.Request[any], deleteReq DeleteRequest) error {
	log.Printf("Deleting entity with ID: %s", deleteReq.GetUuid())

	if err := g.Service.Delete(ctx, deleteReq.GetUuid()); err != nil {
		log.Printf("Error deleting entity: %v", err)
		return err
	}

	log.Printf("Successfully deleted entity")
	return nil
}
