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

## Segurança

- **Access Token** do Entra ID (não ID Token) com validação de assinatura JWKS
- **RBAC** por scopes (`orders.read`, `orders.write`) e app roles (`Orders.Reader`, `Orders.Writer`, `Orders.Admin`)
- **TLS** obrigatório em produção (`TLS_CERT_FILE`, `TLS_KEY_FILE`)
- **Rate limiting** por IP (padrão: 100 req/min)
- **Security headers** (HSTS, CSP, X-Frame-Options, etc.)
- **PostgreSQL** com `sslmode=require` obrigatório em produção
- **Limite de body** HTTP configurável (`MAX_BODY_BYTES`)

### Configurar scopes no Entra ID

1. App Registration → **Expose an API** → adicionar scopes:
   - `orders.read` — leitura de pedidos e status de sync
   - `orders.write` — criar, atualizar e cancelar envios
2. **App roles** (opcional): `Orders.Reader`, `Orders.Writer`, `Orders.Admin`
3. Cliente solicita token com scope: `api://<client-id>/orders.read`

### Exemplo de token

```http
Authorization: Bearer <access_token>
```

O access token deve conter `aud=api://<client-id>` e `scp=orders.read orders.write`.

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
| `GET` | `/swagger` | Swagger UI interativo (público) |
| `GET` | `/api/v1/orders` | Lista pedidos paginado (`?page=1&limit=20&status=OPEN`) |
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

Isso aplica o schema e o seed de desenvolvimento (`000003_seed_dev`).

### Seed de desenvolvimento

| Pedido | Status | Itens |
|--------|--------|-------|
| `PO-2026-001` | OPEN | 2 demandas |
| `PO-2026-002` | IN_PROGRESS | 2 demandas |
| `PO-2026-003` | CLOSED | 1 demanda |

Todos criados por `dev@brunodias.dev`. Autentique no Entra ID com esse e-mail (ou configure o usuário de teste no Azure) para consumir os dados via API.

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
make test                 # unitários
make test-integration     # integração com PostgreSQL (testcontainers + Docker)
```

## Deploy no Railway

1. Crie um projeto no [Railway](https://railway.app)
2. Adicione um serviço **PostgreSQL**
3. Adicione um serviço conectado ao repositório GitHub
4. Configure as variáveis:

| Variável | Valor |
|----------|-------|
| `ENV` | `production` |
| `TLS_TERMINATED_AT_EDGE` | `true` |
| `AZURE_TENANT_ID` | seu tenant |
| `AZURE_CLIENT_ID` | seu client id |
| `AZURE_AUDIENCE` | `api://<client-id>` |
| `DATABASE_URL` | gerado pelo plugin PostgreSQL |

5. O `railway.toml` executa migrations no **pre-deploy** e usa `/health/ready` como health check
6. Acesse `https://<app>.up.railway.app/swagger` para a documentação interativa

## Paginação

```bash
GET /api/v1/orders?page=1&limit=20&status=OPEN
```

```json
{
  "data": [...],
  "page": 1,
  "limit": 20,
  "total": 42,
  "totalPages": 3
}
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
