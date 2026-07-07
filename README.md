# sap-orders-api

Go REST API for orders & demands — Microsoft Entra ID auth + SAP RFC/XML sync.

Backend em Go para gestão de pedidos e demandas com data de entrega. Autenticação via **Microsoft Entra ID** (Azure AD) e sincronização com SAP via **RFC**, com payloads **XML** para atualização de dados.

## Stack

- Go 1.23
- Chi (HTTP router)
- PostgreSQL + pgx
- Microsoft Entra ID (OIDC/JWT)
- Outbox pattern para integração SAP assíncrona
- Structured logging (`slog`)

## Arquitetura

```
Client → Entra ID (JWT) → API Go → PostgreSQL
                              ↓
                         Outbox Worker → SAP RFC (XML)
```

## Boas práticas aplicadas

- Camadas com interfaces (`handler → service → repository`)
- Erros tipados no domínio (`domain.ErrNotFound`, `domain.ErrValidation`, etc.)
- Transações para operações compostas (enqueue + update status)
- Outbox pattern com retry e `FOR UPDATE SKIP LOCKED`
- Health checks separados (`/health/live` e `/health/ready`)
- OpenAPI em `/openapi.yaml`
- Testes unitários com mocks

## Modelo de dados

- **orders** — pedidos
- **order_items** — itens/demandas com `delivery_date`
- **sap_sync_logs** — auditoria das chamadas RFC/XML
- **sap_outbox** — fila de sincronização assíncrona com SAP

## Endpoints

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/health/live` | Liveness (público) |
| `GET` | `/health/ready` | Readiness com ping no DB (público) |
| `GET` | `/openapi.yaml` | Especificação OpenAPI (público) |
| `GET` | `/api/v1/orders` | Lista pedidos |
| `POST` | `/api/v1/orders` | Cria pedido |
| `GET` | `/api/v1/orders/{id}` | Detalhe com itens |
| `PATCH` | `/api/v1/orders/{id}` | Atualiza status |
| `POST` | `/api/v1/orders/{id}/items` | Adiciona demanda |
| `PATCH` | `/api/v1/orders/{id}/items/{itemId}` | Atualiza demanda |
| `POST` | `/api/v1/orders/{id}/items/{itemId}/sync` | Enfileira sync SAP (202) |
| `GET` | `/api/v1/orders/{id}/items/{itemId}/sync` | Consulta status do último envio |
| `GET` | `/api/v1/orders/{id}/items/{itemId}/sync/{outboxId}` | Consulta status de um envio específico |
| `DELETE` | `/api/v1/orders/{id}/items/{itemId}/sync/{outboxId}` | Cancela envio pendente |
| `GET` | `/api/v1/orders/{id}/items/{itemId}/sync-logs` | Histórico de sync |

## Como rodar

### Pré-requisitos

- Go 1.23+
- Docker
- App registrado no Microsoft Entra ID

### 1. Subir o banco

```bash
docker compose up -d
```

### 2. Rodar migrations

```bash
export DATABASE_URL=postgres://saporders:saporders@localhost:5434/saporders?sslmode=disable
make migrate-up
```

### 3. Configurar variáveis

```bash
cp .env.example .env
```

### 4. Rodar a API

```bash
go mod tidy
make run
```

A API ficará disponível em `http://localhost:8081`.

### Testes

```bash
make test
```

## Fluxo de sincronização SAP

1. `POST /sync` enfileira mensagem na `sap_outbox` (transação atômica, retorna **202**)
2. `GET /sync` ou `GET /sync/{outboxId}` consulta o status do envio
3. `DELETE /sync/{outboxId}` cancela o envio enquanto estiver **PENDING** (retorna **409** se já estiver em processamento)
4. Worker em background processa a fila a cada 5s
5. Payload XML é enviado ao SAP via RFC
6. Resultado é gravado em `sap_sync_logs` e status do item atualizado
7. `GET /sync-logs` retorna o histórico completo de tentativas

### Status do outbox

| Status | Descrição |
|--------|-----------|
| `PENDING` | Aguardando processamento (pode cancelar) |
| `PROCESSING` | Em processamento pelo worker |
| `COMPLETED` | Enviado com sucesso ao SAP |
| `FAILED` | Falha após esgotar tentativas |
| `CANCELLED` | Cancelado pelo usuário |

### Exemplo de consulta de status

```bash
GET /api/v1/orders/{id}/items/{itemId}/sync/{outboxId}
Authorization: Bearer <token>
```

```json
{
  "outbox": {
    "id": "uuid",
    "status": "COMPLETED",
    "attempts": 1,
    "createdAt": "2026-07-07T10:00:00Z"
  },
  "latestLog": {
    "status": "SUCCESS",
    "rfcFunction": "Z_UPDATE_DEMAND"
  }
}
```

## Repositório

https://github.com/brufelix/sap-orders-api
