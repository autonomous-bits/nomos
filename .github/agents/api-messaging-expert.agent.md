---
name: api-messaging-expert
description: Provides expertise in API design, gRPC patterns, async messaging, and inter-service communication for provider integration and distributed systems.
---

## Standards Source
- https://github.com/autonomous-bits/development-standards/tree/main/api_design
- https://github.com/autonomous-bits/development-standards/tree/main/messaging_patterns
- https://github.com/autonomous-bits/development-standards/tree/main/async_api
Last synced: 2025-12-25

## Coverage Areas
- REST API Concepts
- API Versioning
- Async Operations
- gRPC Patterns
- Protocol Buffers
- Message Serialization
- Provider Communication Protocols

## Content

### REST API Concepts

#### Resource-Based Design
- **URIs identify resources**: Each resource has a unique URI (e.g., `https://api.contoso.com/orders/1`)
- **Use nouns, not verbs**: Resources should be noun-based (`/orders` not `/getOrders`)
- **Plural for collections**: `/customers`, `/orders` (not `/customer`, `/order`)
- **HTTP methods define actions**: GET (retrieve), POST (create), PUT (update), PATCH (partial update), DELETE (remove)

#### Uniform Interface
REST APIs use standard HTTP verbs consistently:
- **GET**: Retrieve resource (safe, idempotent)
- **POST**: Create new resource (not idempotent)
- **PUT**: Update/replace resource (idempotent)
- **PATCH**: Partial update (not guaranteed idempotent)
- **DELETE**: Remove resource (idempotent)

**Method patterns by resource type:**
| Resource | POST | GET | PUT | DELETE |
|----------|------|-----|-----|--------|
| `/customers` | Create new customer | Retrieve all customers | Bulk update | Remove all customers |
| `/customers/1` | Error | Retrieve customer 1 | Update customer 1 | Remove customer 1 |
| `/customers/1/orders` | Create order for customer 1 | Retrieve orders | Bulk update orders | Remove all orders |

#### Richardson Maturity Model
- **Level 0 (Swamp of POX)**: Single URI, single HTTP method, all actions in payload
- **Level 1 (Resources)**: Multiple URIs for different resources, still using POST for everything
- **Level 2 (HTTP Verbs)**: Proper use of HTTP methods and status codes
- **Level 3 (HATEOAS)**: Hypermedia controls for API discoverability

#### Content Negotiation
- **Accept header**: Client specifies desired response format
- **Content-Type header**: Server indicates actual response format
- **Error handling**: Return 415 (Unsupported Media Type) or 406 (Not Acceptable)

#### HTTP Status Codes
**2xx Success:**
- 200 OK: Request succeeded
- 201 Created: Resource created
- 202 Accepted: Async processing started
- 204 No Content: Success with no body

**4xx Client Errors:**
- 400 Bad Request: Invalid input
- 401 Unauthorized: Authentication required
- 403 Forbidden: Insufficient permissions
- 404 Not Found: Resource doesn't exist
- 409 Conflict: State conflict
- 422 Unprocessable Entity: Validation errors

**5xx Server Errors:**
- 500 Internal Server Error
- 502 Bad Gateway
- 503 Service Unavailable
- 504 Gateway Timeout

---

### API Versioning

#### Versioning Strategies

**1. No Versioning (Additive Only)**
- Only add new fields, never remove or rename
- Clients ignore unknown fields
- Works for internal APIs with controlled clients
- **Not suitable for breaking changes**

**2. URI Versioning**
```http
GET https://api.contoso.com/v2/customers/3
```
- Simple and explicit
- Version in the path
- Can complicate HATEOAS implementation

**3. Query String Versioning**
```http
GET https://api.contoso.com/customers/3?version=2
```
- Same base URI across versions
- Semantically correct (URI identifies resource, not version)
- May affect caching in older proxies

**4. Header Versioning**
```http
GET https://api.contoso.com/customers/3
Custom-Header: api-version=2
```
- URI remains unchanged
- Version context in headers
- Cleaner URIs

**5. Media Type Versioning**
```http
GET https://api.contoso.com/customers/3
Accept: application/vnd.contoso.v2+json
```
- Version embedded in media type
- Follows REST principles
- More complex to implement

#### Version Strategy Selection
- **URI versioning**: Simplest, most explicit
- **Query string**: Semantically cleaner than URI versioning
- **Header versioning**: Cleanest URIs, requires header parsing
- **Media type**: Most RESTful, highest complexity

---

### Async Operations

#### Long-Running Operations Pattern
For operations that take significant time (POST, PUT, PATCH, DELETE):

**1. Initial Response (202 Accepted)**
```http
HTTP/1.1 202 Accepted
Location: /api/status/12345
```

**2. Status Endpoint**
```http
GET /api/status/12345

HTTP/1.1 200 OK
{
  "status": "In progress",
  "link": {
    "rel": "cancel",
    "method": "delete",
    "href": "/api/status/12345"
  }
}
```

**3. Completion (303 See Other)**
```http
HTTP/1.1 303 See Other
Location: /api/orders/12345
```

#### Async Request-Reply Pattern
- **Accept immediately**: Return 202 with status URI
- **Process asynchronously**: Background worker processes request
- **Status endpoint**: Client polls for completion
- **Final response**: Redirect to result or return result directly

**Client-side polling best practices:**
- Implement exponential backoff
- Use Retry-After header when provided
- Set maximum attempt limits
- Handle timeouts gracefully

#### Alternative Notification Mechanisms
- **WebSockets**: Real-time bidirectional notification
- **Webhooks**: Callback to client-provided URL
- **Server-Sent Events (SSE)**: One-way server push

---

### gRPC Patterns

#### Core Characteristics
- **Binary protocol**: Uses Protocol Buffers (protobuf)
- **High performance**: Smaller payload, faster serialization
- **Strongly typed**: Compile-time type safety
- **HTTP/2 based**: Multiplexing, streaming, flow control
- **Multi-language support**: Code generation for many languages

#### Service Definition (.proto files)
```protobuf
syntax = "proto3";

service InventoryService {
  // Unary RPC
  rpc CheckStock (StockRequest) returns (StockResponse);
  
  // Server streaming
  rpc StreamInventory (Empty) returns (stream InventoryUpdate);
  
  // Client streaming
  rpc UploadData (stream DataRequest) returns (DataResponse);
  
  // Bidirectional streaming
  rpc Chat (stream ChatMessage) returns (stream ChatMessage);
}

message StockRequest {
  string product_id = 1;
  int32 quantity = 2;
}

message StockResponse {
  bool available = 1;
  int32 current_stock = 2;
}
```

#### Streaming Patterns

**1. Server Streaming**
- Server sends multiple responses to single client request
- Use cases: Large dataset pagination, real-time updates, data feeds

**2. Client Streaming**
- Client sends multiple requests, server responds once
- Use cases: File uploads, batch operations, metric aggregation

**3. Bidirectional Streaming**
- Both client and server send streams of messages
- Use cases: Chat, real-time collaboration, interactive sessions

#### Error Handling
- **Status codes**: OK, CANCELLED, INVALID_ARGUMENT, NOT_FOUND, ALREADY_EXISTS, etc.
- **Error details**: Include structured error information in metadata
- **Deadlines**: Set timeouts to prevent indefinite blocking

#### gRPC Best Practices
- Enable HTTP/2 (required for gRPC)
- Configure message size limits
- Use interceptors for logging and error handling
- Enable TLS in production
- Configure keep-alive settings for connection pooling
- Handle cancellation tokens properly
- Manage backpressure in streaming scenarios

---

### Protocol Buffers

#### Message Definition
```protobuf
syntax = "proto3";

message Product {
  int32 id = 1;              // Field number (not value)
  string name = 2;
  double price = 3;
  ProductStatus status = 4;
  repeated string tags = 5;   // Repeated = array/list
}

enum ProductStatus {
  AVAILABLE = 0;
  OUT_OF_STOCK = 1;
  DISCONTINUED = 2;
}
```

#### Schema Evolution
**Rules for backward compatibility:**
- **Never change field numbers**: Field numbers identify fields in binary format
- **Add new fields**: Always safe, old readers ignore them
- **Mark fields as reserved**: Prevent reuse of deleted field numbers
- **Use optional fields**: For fields that may not always be present

**Field number management:**
```protobuf
message User {
  reserved 2, 15, 9 to 11;     // Reserve deleted field numbers
  reserved "old_field_name";    // Reserve deleted field names
  
  int32 id = 1;
  string name = 3;              // Skipped 2 (reserved)
  string email = 4;
}
```

#### Best Practices
- **Use meaningful field names**: Self-documenting
- **Document messages and fields**: Add comments for clarity
- **Group related messages**: Use packages and nested messages
- **Version your schemas**: Track changes over time
- **Centralize .proto files**: Share definitions across services
- **Use well-known types**: Timestamp, Duration, Any, etc.

---

### Message Serialization

#### Format Comparison

| Format | Size | Speed | Human-Readable | Schema | Use Case |
|--------|------|-------|----------------|--------|----------|
| **JSON** | Large | Moderate | Yes | Optional | Public APIs, debugging |
| **Protocol Buffers** | Very Small | Fastest | No | Required | gRPC, performance-critical |
| **Avro** | Smallest | Fast | No | Required | Big data, Kafka |
| **MessagePack** | Compact | Fast | No | No | Cache, real-time apps |
| **XML** | Very Large | Slow | Yes | Optional | Legacy enterprise |

#### Protocol Buffers vs Avro

**Protocol Buffers:**
- Fastest serialization performance
- Very small payload size
- Rigid schema evolution (careful field number management)
- Ideal for: gRPC, high-performance microservices, internal communication

**Avro:**
- Smallest payload size
- Best-in-class schema evolution (decouples reader and writer schemas)
- Strong within Kafka ecosystem
- Ideal for: Big data pipelines, flexible schema evolution, Kafka-based systems

#### Serialization Best Practices
- **Choose based on use case**: JSON for public APIs, binary for performance
- **Version message schemas**: Plan for evolution from day one
- **Document schema changes**: Track breaking vs non-breaking changes
- **Validate at boundaries**: Ensure message integrity
- **Compress large payloads**: gzip for text formats
- **Consider bandwidth costs**: Binary formats reduce network usage

---

### Provider Communication Protocols

#### RPC Patterns for Provider Integration

**1. Request-Response (Unary)**
- Single request, single response
- Synchronous operation
- Best for: Simple queries, status checks, configuration retrieval

**2. Server Streaming**
- Single request, multiple responses
- Server pushes data to client
- Best for: Log streaming, event feeds, long-polling alternatives

**3. Client Streaming**
- Multiple requests, single response
- Client uploads data in chunks
- Best for: File uploads, batch operations, telemetry submission

**4. Bidirectional Streaming**
- Multiple requests and responses
- Full-duplex communication
- Best for: Interactive protocols, real-time collaboration, continuous data exchange

#### Provider-Proto Integration

For Nomos provider-proto contracts:

**Define provider service:**
```protobuf
syntax = "proto3";

service TerraformProvider {
  // Configure provider
  rpc Configure (ConfigureRequest) returns (ConfigureResponse);
  
  // Validate resource configuration
  rpc ValidateResourceConfig (ValidateResourceConfigRequest) 
    returns (ValidateResourceConfigResponse);
  
  // Apply resource changes
  rpc ApplyResourceChange (ApplyResourceChangeRequest) 
    returns (ApplyResourceChangeResponse);
  
  // Read current resource state
  rpc ReadResource (ReadResourceRequest) 
    returns (ReadResourceResponse);
}
```

**Message design principles:**
- **Self-contained messages**: Include all context needed for processing
- **Correlation IDs**: Track requests across boundaries
- **Versioning fields**: Include schema version in messages
- **Error details**: Structured error information
- **Metadata**: Timestamps, tracing headers, retry counts

#### Asynchronous Messaging Patterns

**1. Publish-Subscribe (Event Broadcasting)**
- One publisher, multiple subscribers
- Subscribers receive all events
- Best for: Domain events, state change notifications, fan-out scenarios

**2. Point-to-Point (Message Queue)**
- One publisher, one consumer per message
- Load balancing across consumers
- Best for: Commands, task distribution, background jobs

**3. Request-Reply over Message Queue**
- Synchronous behavior using async infrastructure
- Combines messaging benefits with request-response semantics
- Best for: Reliable command execution with response

#### AsyncAPI Specification

For documenting async APIs:

**Required components:**
- **asyncapi**: Specification version (3.0.0+)
- **info**: Metadata (title, version, description)
- **channels**: Communication endpoints (topics, queues)
- **operations**: Send/receive operations the application implements
- **messages**: Message structure, payload, headers

**Example AsyncAPI document:**
```yaml
asyncapi: 3.0.0
info:
  title: Provider Events API
  version: 1.0.0
  description: Provider lifecycle events

channels:
  providerEvents:
    address: provider.events
    messages:
      providerConfigured:
        $ref: '#/components/messages/ProviderConfigured'

operations:
  publishProviderConfigured:
    action: send
    channel:
      $ref: '#/channels/providerEvents'
    messages:
      - $ref: '#/components/messages/ProviderConfigured'

components:
  messages:
    ProviderConfigured:
      contentType: application/json
      payload:
        type: object
        properties:
          providerId:
            type: string
          timestamp:
            type: string
            format: date-time
```

#### CloudEvents Integration

All events should follow CloudEvents v1.0 specification:

**Required HTTP headers:**
- `ce-id`: Unique event identifier (UUID)
- `ce-source`: Context where event occurred (URI)
- `ce-specversion`: CloudEvents version (1.0)
- `ce-type`: Event type (hierarchical, versioned)

**Optional headers:**
- `ce-time`: Event timestamp (ISO 8601)
- `ce-datacontenttype`: Payload content type
- `ce-subject`: Subject of the event
- `ce-dataschema`: Schema URI

#### Communication Protocol Selection

| Scenario | Recommended Protocol |
|----------|---------------------|
| User-facing APIs | HTTP/REST with JSON |
| Internal service calls | gRPC with Protocol Buffers |
| Provider integration | gRPC (for provider-proto) |
| Event broadcasting | Async messaging (Pub/Sub) |
| Background tasks | Message queues (Point-to-Point) |
| Real-time updates | gRPC bidirectional streaming |
| Large file transfer | gRPC client streaming |

#### Resilience Patterns

For reliable provider communication:

**1. Timeouts**
- Set aggressive, defined timeouts on all network requests
- Prevent indefinite blocking on failed providers

**2. Retries with Exponential Backoff**
- Retry failed requests automatically
- Use jitter to prevent thundering herd
- Limit retry attempts (typically 3-5)

**3. Circuit Breakers**
- Stop calling failing providers temporarily
- Allow providers time to recover
- Prevent cascading failures

**4. Idempotency**
- Design operations to be safely retried
- Use idempotency tokens/keys
- Handle duplicate requests gracefully

**5. Dead Letter Queues**
- Route failed messages to DLQ
- Enable manual inspection and reprocessing
- Prevent blocking of main processing queue

---

## Usage

Consult this agent for:

1. **API Design Questions**
   - REST API structure and conventions
   - Resource modeling and URI design
   - HTTP method selection and status codes
   - API versioning strategies

2. **gRPC Implementation**
   - Service definition and .proto design
   - Streaming pattern selection
   - Error handling and deadlines
   - Performance optimization

3. **Provider Communication**
   - provider-proto contract design
   - RPC patterns for provider operations
   - Message format selection
   - Protocol buffer schema evolution

4. **Async Operations**
   - Long-running operation handling
   - Status endpoint design
   - Polling vs push notifications
   - AsyncAPI documentation

5. **Message Serialization**
   - Format selection (JSON, Protobuf, Avro)
   - Schema design and versioning
   - Performance considerations
   - Backward compatibility

6. **Integration Patterns**
   - Synchronous vs asynchronous communication
   - Event-driven architecture
   - Request-response patterns
   - Service decoupling strategies

## Examples

### Example 1: Designing a Provider Configuration API

**Question**: How should I design the Configure RPC for a new provider?

**Answer**: Use unary RPC with comprehensive request/response messages:

```protobuf
service MyProvider {
  rpc Configure (ConfigureRequest) returns (ConfigureResponse);
}

message ConfigureRequest {
  string version = 1;              // Provider version
  map<string, string> config = 2;  // Configuration parameters
  string correlation_id = 3;       // Request tracking
}

message ConfigureResponse {
  bool success = 1;
  string message = 2;
  repeated string warnings = 3;
}
```

Include validation, error details, and support for schema evolution.

### Example 2: Async Operation for Long-Running Apply

**Question**: How do I handle a long-running resource apply operation?

**Answer**: Use the async request-reply pattern:

1. Return 202 Accepted immediately:
```http
POST /api/resources/apply
→ 202 Accepted
   Location: /api/operations/abc-123
```

2. Provide status endpoint:
```http
GET /api/operations/abc-123
→ 200 OK
{
  "status": "in_progress",
  "progress": 45,
  "estimatedCompletion": "2025-12-25T10:35:00Z"
}
```

3. Redirect to result on completion:
```http
GET /api/operations/abc-123
→ 303 See Other
   Location: /api/resources/resource-456
```

### Example 3: Versioning a Provider Protocol

**Question**: How do I version the provider protocol messages?

**Answer**: Use field numbers carefully and include version information:

```protobuf
message ResourceRequest {
  string resource_id = 1;
  string schema_version = 2;  // Track message version
  
  // V1 fields
  string name = 3;
  string type = 4;
  
  // V2 additions (new field numbers)
  map<string, string> tags = 5;     // Added in V2
  ResourceMetadata metadata = 6;    // Added in V2
  
  // Reserved for deleted fields
  reserved 10, 11;
  reserved "old_field";
}
```

Never reuse field numbers, always add new fields with new numbers.

---

## Related Agents

- **Go Coding Standards Expert**: For implementing providers in Go
- **Testing & Quality Expert**: For API and provider testing strategies
- **Documentation Expert**: For API documentation and AsyncAPI specs

---

*Last updated: 2025-12-25*
*Standards synchronized with autonomous-bits/development-standards*
