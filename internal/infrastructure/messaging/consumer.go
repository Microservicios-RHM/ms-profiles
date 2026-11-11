package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("rhm-messaging")

// Handler procesa un evento ya parseado. Un error indica que el mensaje debe descartarse (no
// reintentar) — igual que aio-pika en ms-notifications, la deduplicación aguas abajo (en el
// repositorio) es lo que garantiza que una redelivery nunca duplique el efecto, así que no hace
// falta requeue.
type Handler func(ctx context.Context, envelope EventEnvelope) error

// Consumer declara su propio exchange (idempotente, no depende del orden de arranque frente a
// empleados-service) y su propia cola, enlazada a los routing keys registrados con On(). Cada
// servicio consumidor tiene su cola propia — eso es lo que produce el fan-out.
type Consumer struct {
	url                string
	exchangeName       string
	queueName          string
	logger             *slog.Logger
	connectMaxAttempts int
	connectRetryDelay  time.Duration

	handlers map[string]Handler

	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewConsumer(url, exchangeName, queueName string, logger *slog.Logger, connectMaxAttempts int, connectRetryDelay time.Duration) *Consumer {
	return &Consumer{
		url:                url,
		exchangeName:       exchangeName,
		queueName:          queueName,
		logger:             logger,
		connectMaxAttempts: connectMaxAttempts,
		connectRetryDelay:  connectRetryDelay,
		handlers:           make(map[string]Handler),
	}
}

func (c *Consumer) On(routingKey string, handler Handler) {
	c.handlers[routingKey] = handler
}

func (c *Consumer) Start(ctx context.Context) error {
	var err error
	for attempt := 1; attempt <= c.connectMaxAttempts; attempt++ {
		c.conn, err = amqp.Dial(c.url)
		if err == nil {
			break
		}
		c.logger.Warn("RabbitMQ connection attempt failed", "attempt", attempt, "error", err.Error())
		if attempt < c.connectMaxAttempts {
			time.Sleep(c.connectRetryDelay * time.Duration(attempt))
		}
	}
	if err != nil {
		return fmt.Errorf("no fue posible conectar a RabbitMQ: %w", err)
	}

	c.channel, err = c.conn.Channel()
	if err != nil {
		return fmt.Errorf("no fue posible abrir el canal de RabbitMQ: %w", err)
	}
	if err := c.channel.Qos(10, 0, false); err != nil {
		return fmt.Errorf("no fue posible configurar QoS: %w", err)
	}
	if err := c.channel.ExchangeDeclare(c.exchangeName, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("no fue posible declarar el exchange: %w", err)
	}
	queue, err := c.channel.QueueDeclare(c.queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("no fue posible declarar la cola: %w", err)
	}

	bindings := make([]string, 0, len(c.handlers))
	for routingKey := range c.handlers {
		if err := c.channel.QueueBind(queue.Name, routingKey, c.exchangeName, false, nil); err != nil {
			return fmt.Errorf("no fue posible enlazar %q: %w", routingKey, err)
		}
		bindings = append(bindings, routingKey)
	}

	deliveries, err := c.channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("no fue posible iniciar el consumo: %w", err)
	}

	go func() {
		for delivery := range deliveries {
			c.dispatch(ctx, delivery)
		}
	}()

	c.logger.Info("RabbitMQ connection ready", "exchange", c.exchangeName, "queue", c.queueName, "bindings", bindings)
	return nil
}

func (c *Consumer) dispatch(ctx context.Context, delivery amqp.Delivery) {
	ctx, span := c.startConsumeSpan(ctx, delivery.RoutingKey, delivery.Headers)
	defer span.End()

	if err := c.route(ctx, delivery.RoutingKey, delivery.Body); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.logger.ErrorContext(ctx, "event discarded", "routingKey", delivery.RoutingKey, "error", err.Error())
		_ = delivery.Nack(false, false)
		return
	}
	_ = delivery.Ack(false)
}

// startConsumeSpan continúa la traza del productor (Reto 8). No existe instrumentación oficial de
// OpenTelemetry para amqp091-go, así que la propagación es manual: el `traceparent` (W3C Trace
// Context) que empleados-service dejó en las cabeceras del mensaje se extrae y el span CONSUMER
// cuelga del span PRODUCER. Sin esto, la creación del perfil aparecería en Zipkin como una traza
// suelta, separada del alta del empleado que la provocó.
func (c *Consumer) startConsumeSpan(ctx context.Context, routingKey string, headers amqp.Table) (context.Context, trace.Span) {
	carrier := propagation.MapCarrier{}
	for key, value := range headers {
		switch v := value.(type) {
		case string:
			carrier[key] = v
		case []byte:
			carrier[key] = string(v)
		}
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	return tracer.Start(ctx, routingKey+" process",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.operation.type", "process"),
			attribute.String("messaging.destination.name", c.queueName),
			attribute.String("messaging.rabbitmq.destination.routing_key", routingKey),
		),
	)
}

// IsConnected se usa en GET /health (check messageBroker).
func (c *Consumer) IsConnected() bool {
	return c.conn != nil && !c.conn.IsClosed()
}

// route contiene la lógica pura (parseo + despacho), separada de amqp.Delivery para poder
// probarla sin necesitar una conexión real.
func (c *Consumer) route(ctx context.Context, routingKey string, body []byte) error {
	handler, ok := c.handlers[routingKey]
	if !ok {
		return nil
	}
	var envelope EventEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("evento malformado: %w", err)
	}
	return handler(ctx, envelope)
}

func (c *Consumer) Close() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
