// Package observability agrupa la instrumentación del Reto 8: trazas OpenTelemetry hacia Zipkin,
// métricas Prometheus y el health check con verificación real de dependencias.
package observability

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const DefaultServiceName = "perfiles-service"

// ServiceName es el service.name que aparece en Zipkin y en el campo `service` de los logs.
func ServiceName() string {
	if name := os.Getenv("OTEL_SERVICE_NAME"); name != "" {
		return name
	}
	return DefaultServiceName
}

// SetupTracing configura el SDK de OpenTelemetry:
//   - exportador Zipkin hacia OTEL_EXPORTER_ZIPKIN_ENDPOINT (nombre de red Docker `zipkin`);
//   - propagación W3C Trace Context + Baggage, el mismo estándar que usan los servicios en Node,
//     Python, Ruby y PHP: un traceparent creado en el Gateway (Node) continúa aquí.
//
// Sin OTEL_EXPORTER_ZIPKIN_ENDPOINT (pruebas, ejecución local sin Zipkin) solo se registra el
// propagador: el proveedor global sigue siendo el no-op de la API y no se exporta nada.
// Devuelve la función que vacía los spans pendientes al apagar el servicio.
func SetupTracing(ctx context.Context) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	endpoint := os.Getenv("OTEL_EXPORTER_ZIPKIN_ENDPOINT")
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := zipkin.New(endpoint)
	if err != nil {
		return nil, err
	}
	// Sin esquema propio: resource.Default() ya trae la versión de semconv del SDK y fusionar dos
	// esquemas distintos es un error ("conflicting Schema URL").
	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(semconv.ServiceName(ServiceName())))
	if err != nil {
		return nil, err
	}
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	return provider.Shutdown, nil
}
