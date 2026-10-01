package messaging

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

const employeeCreatedBody = `{
	"id": "evt-1",
	"type": "empleado.creado",
	"version": 1,
	"occurredAt": "2026-03-01T10:00:00.000Z",
	"producer": "empleados-service",
	"data": {"id": "E001"}
}`

func TestRoute_DespachaAlHandlerRegistrado(t *testing.T) {
	consumer := NewConsumer("amqp://localhost", "rhm.events", "test.queue", testLogger(), 1, 0)

	var received EventEnvelope
	consumer.On("empleado.creado", func(_ context.Context, envelope EventEnvelope) error {
		received = envelope
		return nil
	})

	if err := consumer.route(context.Background(), "empleado.creado", []byte(employeeCreatedBody)); err != nil {
		t.Fatalf("route() error = %v, want nil", err)
	}
	if received.ID != "evt-1" {
		t.Errorf("received.ID = %q, want evt-1", received.ID)
	}
	if received.StringField("id") != "E001" {
		t.Errorf("received.Data[id] = %q, want E001", received.StringField("id"))
	}
}

func TestRoute_IgnoraRoutingKeysSinHandler(t *testing.T) {
	consumer := NewConsumer("amqp://localhost", "rhm.events", "test.queue", testLogger(), 1, 0)
	consumer.On("empleado.creado", func(context.Context, EventEnvelope) error {
		t.Fatal("no debería llamarse")
		return nil
	})

	if err := consumer.route(context.Background(), "empleado.retirado", []byte(employeeCreatedBody)); err != nil {
		t.Fatalf("route() error = %v, want nil", err)
	}
}

func TestRoute_PropagaMensajesMalformados(t *testing.T) {
	consumer := NewConsumer("amqp://localhost", "rhm.events", "test.queue", testLogger(), 1, 0)
	consumer.On("empleado.creado", func(context.Context, EventEnvelope) error {
		t.Fatal("no debería llamarse")
		return nil
	})

	err := consumer.route(context.Background(), "empleado.creado", []byte("esto no es JSON"))
	if err == nil {
		t.Fatal("route() error = nil, want error por JSON malformado")
	}
}

func TestRoute_PropagaElErrorDelHandler(t *testing.T) {
	consumer := NewConsumer("amqp://localhost", "rhm.events", "test.queue", testLogger(), 1, 0)
	wantErr := errors.New("fallo del handler")
	consumer.On("empleado.creado", func(context.Context, EventEnvelope) error {
		return wantErr
	})

	err := consumer.route(context.Background(), "empleado.creado", []byte(employeeCreatedBody))
	if !errors.Is(err, wantErr) {
		t.Fatalf("route() error = %v, want %v", err, wantErr)
	}
}
