package gcs

import (
	"context"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/projector"
	semruntime "github.com/c360studio/semlink/internal/semstreams"
)

func TestCommandServiceSubmitWritesBoundCommandProjection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	runtime, err := semruntime.StartRuntime(ctx, semruntime.RuntimeOptions{Embedded: true})
	if err != nil {
		t.Fatalf("StartRuntime: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		if err := runtime.Stop(stopCtx); err != nil {
			t.Errorf("Runtime.Stop: %v", err)
		}
	})

	store := NewStore(runtime.NATSURL, true)
	service := NewCommandService(runtime.Graph, store)
	command, err := service.Submit(ctx, projector.VehicleEntityID(7), "request_autopilot_version")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if command.GraphRevision == 0 || command.Verb != "request-autopilot-version" {
		t.Fatalf("command = %#v", command)
	}
	exact, err := runtime.Graph.QueryExactEntity(ctx, command.EntityID)
	if err != nil {
		t.Fatalf("QueryExactEntity: %v", err)
	}
	if exact.KVRevision != command.GraphRevision || exact.Entity.GetTriple(projector.PredicateCommandVerb) == nil {
		t.Fatalf("exact command = %#v", exact)
	}
}
