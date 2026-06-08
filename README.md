# OPanel

Server control panel scritto in Go, ispirato a Plesk Obsidian.

**Target OS:** Debian 13 (Trixie)
**Stack:** Go 1.24 / SQLite / JWT / Cobra CLI / Vue 3 / TypeScript / Tailwind CSS

---

## Build e Run

### Docker (consigliato)

```bash
# Build image
docker build -t opanel .

# Run container
docker run -d -p 8443:8443 --name opanel opanel

# View logs
docker logs -f opanel

# Stop
docker stop opanel
```

### Docker Compose

```bash
# Build e avvia
docker compose up -d --build

# Logs
docker compose logs -f

# Ferma
docker compose down
```

### Build locale (senza Docker)

```bash
make build        # Compila in ./bin/opaneld (include frontend)
make run          # Build + avvia server
make test         # Test
make fmt          # Formatta
make vet          # Verifica
make frontend-dev # Avvia sviluppo frontend (hot reload su porta 3000)
make frontend-build # Build frontend per produzione
```

---

## Accesso

URL: `https://localhost:8443`

Credenziali admin default:
- Username: `admin`
- Password: `admin`

**Importante:** Cambia la password dopo il primo accesso.

---

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/health` | No | Health check |
| `POST` | `/api/auth/login` | No | Login |
| `POST` | `/api/auth/logout` | Yes | Logout |
| `GET` | `/api/auth/me` | Yes | Current user |
| `GET` | `/api/users` | Yes (admin) | List users |
| `POST` | `/api/users` | Yes (admin) | Create user |
| `PUT` | `/api/users/:id` | Yes (admin) | Update user |
| `DELETE` | `/api/users/:id` | Yes (admin) | Delete user |
| `GET` | `/api/databases` | Yes | List databases |
| `GET` | `/api/databases/:id` | Yes | Get database |
| `POST` | `/api/databases` | Yes | Create database |
| `DELETE` | `/api/databases/:id` | Yes (admin) | Delete database |
| `POST` | `/api/databases/:id/users` | Yes | Create database user |
| `DELETE` | `/api/databases/:id/users/:userId` | Yes | Delete database user |
| `PUT` | `/api/databases/:id/users/:userId` | Yes | Update database user |

### Esempi

```bash
# Health
curl http://localhost:8443/api/health

# Login
curl -X POST http://localhost:8443/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'

# Usa il token
TOKEN="<token>"
curl -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/users
```

---

## Struttura Progetto

```
OPanel/
├── cmd/opaneld/main.go           # CLI (cobra)
├── internal/
│   ├── config/config.go          # Config YAML
│   ├── database/db.go            # SQLite WAL
│   ├── database/migrations.go    # Migrazioni auto
│   ├── handler/auth.go           # Login/logout/me
│   ├── handler/database.go       # CRUD database
│   ├── handler/health.go         # Health check
│   ├── handler/user.go           # CRUD utenti
│   ├── jwt/jwt.go                # Token JWT
│   ├── middleware/auth.go        # Auth + admin guard
│   ├── middleware/logging.go     # Logging
│   ├── model/database.go         # Modelli Database
│   ├── model/user.go             # Modelli dati
│   ├── server/server.go          # Server HTTP
│   ├── server/routes.go          # Routes + SPA serving
│   └── service/
│       ├── domain.go             # Logica business domini
│       ├── mariadb.go            # Gestione MariaDB
│       ├── nginx.go              # Template engine Nginx
│       ├── phpfpm.go             # Gestione PHP-FPM pools
│       └── system.go             # Operazioni Linux
├── frontend/                     # Vue 3 SPA frontend
│   ├── src/
│   │   ├── main.ts              # Entry point
│   │   ├── App.vue              # Root component
│   │   ├── api/index.ts         # API client
│   │   ├── stores/auth.ts       # Pinia auth store
│   │   ├── router/index.ts      # Vue Router
│   │   ├── components/          # Layout components
│   │   └── views/               # Page views
│   ├── package.json
│   ├── vite.config.ts
│   └── tailwind.config.js
├── templates/
│   ├── nginx/
│   │   └── default.conf.template # Template config Nginx
│   └── phpfpm/
│       └── pool.conf.template    # Template pool PHP-FPM
├── config.example.yaml
├── Dockerfile                    # Debian 13 (Trixie) + frontend
├── docker-compose.yml
├── entrypoint.sh                 # Startup multi-servizio
├── Makefile
├── STATUS.md                     # Stato progetto
└── Plan.md                       # Architettura
```

---

## Configurazione

Copia `config.example.yaml` in `/etc/opanel/config.yaml` e modifica:

```yaml
server:
  host: "0.0.0.0"
  port: 8443

database:
  path: "/opt/opanel/db/opanel.db"

jwt:
  secret: "CAMBIA-QUESTO-SEGRETO"
  expiry_hours: 24

admin:
  username: "admin"
  password: "admin"
  email: "admin@localhost"
```

---

## Stato e Roadmap

Vedi [STATUS.md](STATUS.md) per lo stato attuale del progetto e la roadmap completa.

**Sprint completati:** 4/7
**Prossimo:** Sprint 5 - Posta e DNS
