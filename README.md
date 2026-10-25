# Microservicio de perfiles

Servicio construido en Go (biblioteca estándar `net/http`, sin framework), con arquitectura
hexagonal equivalente a `ms-employees` (Node/TS) y `ms-notifications` (Python/FastAPI). Combina
ambos estilos: reactivo (consume eventos de RabbitMQ) y REST (expone lectura/escritura del perfil).

Forma parte del ecosistema RHM (Reto 4). El contrato de eventos está en `docs/event-catalog.md` del
repositorio [rhm-database-infrastructure](https://github.com/Microservicios-RHM/rhm-database-infrastructure).

## Estado

- ✅ Consume `empleado.creado` → crea el perfil por defecto (`telefono`/`direccion`/`ciudad`/
  `biografia` vacíos, como pide el enunciado).
- ✅ Consume `empleado.actualizado` → sincroniza `nombre`/`email` en el perfil existente.
- ✅ Consume `empleado.retirado` → archiva el perfil (`archivado = true`), nunca lo borra.
- ✅ Deduplicación por `id` de mensaje en los tres handlers, con el mismo patrón
  (`INSERT ... ON CONFLICT DO NOTHING RETURNING id` + efecto, una sola transacción) que
  `ms-employees` y `ms-notifications`.
- ✅ `GET /perfiles`, `GET /perfiles/{empleadoId}` (404 si no existe), `PUT /perfiles/{empleadoId}`
  (actualización parcial de `telefono`/`direccion`/`ciudad`/`biografia`), documentados en
  OpenAPI 3.1 (`/perfiles/openapi.json`, Swagger UI en `/perfiles/docs`).
- ✅ Registrado en el API Gateway: `http://localhost:8080/perfiles`, **incluida la documentación
  interactiva** en `http://localhost:8080/perfiles/docs`.

## Requisitos

- Go 1.26 (el `Dockerfile` usa `golang:1.26-alpine` para build y `alpine:3.21` para runtime).
- Docker y Docker Compose para ejecutar el servicio real — igual que el resto del ecosistema, la
  infraestructura no publica puertos de bases de datos ni de RabbitMQ al host.

## Configuración local (tooling, no ejecución)

```bash
go build ./...
go vet ./...
go test ./...
cp .env.example .env
```

Los tests usan dobles de prueba (repositorio en memoria, sin conexión real a PostgreSQL ni a
RabbitMQ). Para correr el servicio contra las dependencias reales, el único flujo soportado es
Docker Compose desde `rhm-database-infrastructure`:

```bash
cd ../rhm-database-infrastructure
docker compose up --build
```

Dentro de Docker, PostgreSQL se resuelve como `database-perfiles:5432` y RabbitMQ como
`message-broker:5672`; ninguno de los dos publica su puerto al host para este servicio.

## Variables de entorno

`.env.example` documenta todas las variables. Las relevantes:

```dotenv
DB_HOST=database-perfiles
DB_PORT=5432
DB_NAME=profiles_db
DB_USER=profiles_service
DB_PASSWORD=change_profiles_password

BROKER_URL=amqp://admin:admin@message-broker:5672
BROKER_EXCHANGE=rhm.events
BROKER_QUEUE=perfiles.queue
```

La aplicación valida la configuración al arrancar (`internal/infrastructure/config`) y falla
inmediatamente si falta una variable requerida (`DB_HOST`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`,
`BROKER_URL`) o si `BROKER_URL` no tiene el esquema `amqp://`/`amqps://`.

## API

Envelope estándar del ecosistema, JSON en camelCase:

```bash
curl -i http://localhost:8080/perfiles
curl -i http://localhost:8080/perfiles/E001
curl -i -X PUT http://localhost:8080/perfiles/E001 \
  -H "Content-Type: application/json" \
  -d '{"telefono":"3001234567","ciudad":"Armenia","biografia":"Ingeniero de sistemas"}'
```

`PUT` es una actualización parcial: un campo ausente en el body conserva su valor actual (no lo
vacía). `id`, `empleadoId`, `nombre`, `email`, `archivado` y `fechaCreacion` no se aceptan en este
endpoint — enviarlos responde `400 VALIDATION_ERROR` (JSON decodificado con
`DisallowUnknownFields`). `telefono`/`direccion`/`ciudad` respetan los límites de columna
(30/255/100 caracteres); excederlos también responde `400 VALIDATION_ERROR`.

Documentación interactiva — montada bajo `/perfiles` (no en la raíz) para vivir detrás del Gateway
sin que este necesite ninguna regla especial, ya que `/perfiles/*` ya se proxea sin reescritura de
ruta:

```text
Swagger UI:     http://localhost:8080/perfiles/docs
OpenAPI JSON:   http://localhost:8080/perfiles/openapi.json
```

### Autenticación en Swagger (Reto 5)

Desde el Reto 5 todas las peticiones exigen un token, que valida el API Gateway. El documento
OpenAPI declara el esquema de seguridad `BearerAuth`, así que la página de Swagger muestra el
botón **Authorize**: se pega ahí el `accessToken` que devuelve `POST /auth/login` y el token viaja
en cada petición de prueba.

Es solo documentación. Este servicio no verifica la firma — de eso se encarga el Gateway, una sola
vez para todo el ecosistema. Sin esta declaración el botón no existiría y cualquier "Try it out"
respondería `401` sin forma de autenticarse.

## Arquitectura

```text
cmd/server/main.go                        Raíz de composición
internal/
├── domain/
│   ├── profile.go                        Entidad Profile
│   ├── errors.go                         ErrProfileNotFound
│   └── ports.go                          Puerto ProfileRepository
├── application/
│   ├── create_default_profile.go         empleado.creado → perfil por defecto
│   ├── sync_profile.go                   empleado.actualizado → sincroniza nombre/email
│   ├── archive_profile.go                empleado.retirado → archiva (no borra)
│   ├── get_profile.go / list_profiles.go / update_profile.go   Casos de uso REST
└── infrastructure/
    ├── config/                           Carga y validación de variables de entorno
    ├── logging/                          Logger JSON (slog), equivalente a Pino/logging_setup.py
    ├── http/
    │   ├── router.go                     Rutas + healthcheck + OpenAPI/Swagger embebidos
    │   ├── profile_handlers.go           GET/PUT de perfiles, validación de la actualización parcial
    │   ├── profile_dto.go                Serialización a camelCase
    │   ├── response.go                   Envelope success/error del ecosistema
    │   ├── openapi.json                  Documento OpenAPI 3.1 (embebido con go:embed)
    │   └── docs.html                     Swagger UI vía CDN (embebido con go:embed)
    ├── messaging/
    │   ├── envelope.go                   Espejo del envelope del catálogo
    │   └── consumer.go                   Conexión, exchange, cola, bindings, dispatch
    └── persistence/
        ├── postgres.go                   Pool pgx con reintentos y backoff
        ├── migrations.go                 Migraciones versionadas e idempotentes
        └── profile_repository.go         Implementación Postgres del puerto (dedup + CRUD)
```

Los casos de uso dependen de `domain.ProfileRepository` (interfaz), no de `pgx` directamente —
mismo principio que `EmployeeRepository` en `ms-employees` y `DedupNotificationRepository` en
`ms-notifications`. `main.go` construye la implementación Postgres y conecta todo.

### Consumidor RabbitMQ

`Consumer` declara el exchange `rhm.events` (topic, durable) de forma idempotente y su propia cola
`perfiles.queue`, enlazada a los routing keys registrados con `.On(...)`. Es una dependencia dura
(igual que en `ms-notifications`): si no logra conectarse tras los reintentos configurados, el
proceso no arranca — reaccionar a eventos es la razón de ser de este servicio.

La lógica de parseo/despacho (`route()`) está separada de la recepción AMQP (`dispatch()`)
justamente para poder probarla sin una conexión real — ver `consumer_test.go`.

### Deduplicación

`CreateIfNew`, `SyncIfNew` y `ArchiveIfNew` comparten el mismo patrón (`markProcessed` en
`profile_repository.go`), cada uno en una única transacción:

1. `INSERT INTO eventos_procesados (id) VALUES ($event_id) ON CONFLICT DO NOTHING RETURNING id`.
2. Si no hay fila (ya existía), no se ejecuta el efecto y se retorna `wasNew = false`.
3. Si hay fila, se aplica el efecto (crear el perfil, sincronizar nombre/email, o archivar).

Si el evento es nuevo pero el perfil no existe (`empleado.actualizado`/`empleado.retirado` para un
`empleadoId` sin `empleado.creado` previo — no debería pasar en operación normal), el handler
registra una advertencia (`found = false`) en vez de fallar o crear un perfil a medias.

**Verificación realizada:** para cada uno de los tres eventos, se creó/actualizó/retiró un
empleado y se republicó manualmente el mismo mensaje (mismo `id` de envelope) vía la API de
administración de RabbitMQ. En los tres casos el log de la segunda entrega mostró
`"msg": "duplicate event ignored"` y la tabla `perfiles` no cambió.

## Persistencia

Base de datos propia (`profiles_db`, PostgreSQL 17), sin relación con las demás. La migración
`create_profiles_table` crea:

- `perfiles` (`id`, `empleado_id` con `UNIQUE` como segunda barrera contra duplicados, `nombre`,
  `email`, `telefono`/`direccion`/`ciudad`/`biografia` con default `''`, `archivado` con default
  `false`, `fecha_creacion`), con índice sobre `empleado_id`.
- `eventos_procesados` (`id`, `procesado_en`), la tabla de deduplicación.

`archivado` es un campo agregado por este servicio (no aparece en el JSON de ejemplo del
enunciado): sin él no habría forma de observar "archivar, no borrar".

`nombre` es un único campo (igual que en el JSON de ejemplo del enunciado), tomado literalmente
del campo `nombre` del evento — `apellido` no se replica en el perfil porque el esquema del
enunciado no lo contempla.

## Salud

```bash
docker exec perfiles-service wget -qO- http://127.0.0.1:8080/health
```

`/health` responde el mismo envelope que el resto del ecosistema:
`{"success": true, "message": "Servicio disponible", "data": {"status": "UP"}}`.
