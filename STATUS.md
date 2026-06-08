# OPanel - Stato del Progetto

**Data:** 2026-06-08
**Sprint attuale:** 1 (completato)
**Target OS:** Debian 13 (Trixie)

---

## Stato Attuale

OPanel e' un clone di Plesk Obsidian scritto in Go. Attualmente e' stato completato lo **Sprint 1** con tutte le funzionalita' base del backend.

### Cosa Funziona

#### Server HTTP
- Server Go integrato su porta **8443** (HTTP, TLS da aggiungere)
- Ascolto su `0.0.0.0:8443`
- Graceful shutdown con gestione segnali SIGINT/SIGTERM
- Timeout letture/scritture (15s) e idle timeout (60s)

#### Autenticazione JWT
- Login con username/password, restituisce token JWT HS256
- Middleware Bearer token su tutte le route protette
- Scadenza token configurabile (default 24h)
- Refresh token struttura presente (table SQLite, logica non ancora implementata)

#### Gestione Utenti (CRUD)
- `POST /api/auth/login` - Login
- `POST /api/auth/logout` - Logout (stub)
- `GET /api/auth/me` - Info utente corrente
- `GET /api/users` - Lista utenti (admin only)
- `POST /api/users` - Crea utente (admin only)
- `PUT /api/users/:id` - Aggiorna utente (admin only)
- `DELETE /api/users/:id` - Elimina utente (admin only)
- `GET /api/health` - Health check

#### Sicurezza
- Password hashate con **bcrypt**
- Due ruoli: `admin` e `user`
- Guard admin: solo gli admin possono gestire utenti
- Validazione input base su tutti gli handler
- Logging middleware con method, path, status, duration

#### Database
- **SQLite** via `modernc.org/sqlite` (pure Go, no CGO)
- WAL mode per performance
- Busy timeout 5s
- Foreign keys abilitate
- Sistema di migrazioni automatico con tabella `migrations`
- Schema: tabelle `users`, `refresh_tokens`, `migrations`

#### CLI (Cobra)
- `opaneld server` - Avvia il server
- `opaneld server --config /path/to/config.yaml` - Config personalizzata
- `opaneld version` - Mostra versione
- `opaneld user create` - Stub
- `opaneld user list` - Stub

#### Configurazione
- File YAML con viper
- Strutture: `server`, `database`, `jwt`, `admin`
- Default sensati per tutti i campi
- Auto-creazione admin al primo avvio se non esistono utenti

#### Container
- **Dockerfile** multi-stage: builder Go 1.23 + runtime Debian 13 (Trixie)
- **docker-compose.yml** con volumi per persistence
- Image funzionante e testato

---

## Struttura File

```
OPanel/
├── cmd/opaneld/main.go           # CLI entrypoint (cobra)
├── internal/
│   ├── config/config.go          # Config YAML (viper)
│   ├── database/db.go            # Connessione SQLite
│   ├── database/migrations.go    # Migrazioni automatiche
│   ├── handler/auth.go           # Login/logout/me
│   ├── handler/health.go         # Health check
│   ├── handler/user.go           # CRUD utenti
│   ├── jwt/jwt.go                # Generazione/validazione JWT
│   ├── middleware/auth.go        # Auth middleware + admin guard
│   ├── middleware/logging.go     # Request logging
│   ├── model/user.go             # Modello dati User
│   ├── server/server.go          # HTTP server + bootstrap admin
│   └── server/routes.go          # Registrazione routes
├── config.example.yaml           # Config di esempio
├── Dockerfile                    # Multi-stage Debian 13
├── docker-compose.yml            # Orchestrazione Docker
├── Makefile                      # Build commands
├── Plan.md                       # Architettura completa
├── STATUS.md                     # Questo file
├── README.md                     # Documentazione
├── go.mod
└── go.sum
```

---

## Cosa Manca (Sprint 2-7)

### Sprint 2 - Web e File System
- [ ] Generazione utenti Linux (`useradd`) con home in `/var/www/vhosts/`
- [ ] Configurazione chroot SFTP per utenti `opanel_users`
- [ ] Template engine Go per generare config Nginx/Apache
- [ ] Creazione/cancellazione domini fisica su disco
- [ ] Struttura directory: `httpdocs/`, `logs/`, `tmp/`

### Sprint 3 - PHP & Database
- [ ] Integrazione PHP-FPM (pool per sito, socket UNIX)
- [ ] Installazione stack MariaDB locale
- [ ] API per creare/eliminare database
- [ ] Gestione utenti database con grant

### Sprint 4 - Frontend MVP
- [ ] Setup Vue 3 + TypeScript + Tailwind CSS
- [ ] Layout: Sidebar fissa, Header, Area contenuto
- [ ] Login page
- [ ] Dashboard domini
- [ ] Connessione API backend

### Sprint 5 - Posta e DNS
- [ ] Controller Bind9 (generazione zone file)
- [ ] Controller Postfix + Dovecot
- [ ] Gestione account email
- [ ] DKIM automatico con Rspamd
- [ ] Autoconfigurazione email client

### Sprint 6 - Strumenti Avanzati
- [ ] File Manager web (upload/download/gestione)
- [ ] Terminale SSH nel browser (Xterm.js + creack/pty)
- [ ] Installatore WordPress 1-click
- [ ] Sistema backup locale + S3
- [ ] Monitoring live (CPU, RAM, I/O) via WebSocket

### Sprint 7 - Estensioni e Polish
- [ ] Motore hook/script per estensioni
- [ ] UI inject da extension.json
- [ ] Test intensivo su VPS Debian 13
- [ ] Ottimizzazione performance
- [ ] Documentazione API completa

---

## Comandi Utili

### Build e Run (Docker)

```bash
# Build image
docker build -t opanel .

# Run container
docker run -d -p 8443:8443 --name opanel opanel

# View logs
docker logs -f opanel

# Stop
docker stop opanel

# Restart
docker restart opanel
```

### Build e Run (Docker Compose)

```bash
# Build e avvia in background
docker compose up -d --build

# View logs
docker compose logs -f

# Stop
docker compose down
```

### Build locale (senza Docker)

```bash
make build        # Compila in ./bin/opaneld
make run          # Build + esegui
make test         # Esegui test
make fmt          # Formatta codice
make vet          # Verifica codice
```

### Test API

```bash
# Health check
curl http://localhost:8443/api/health

# Login
curl -X POST http://localhost:8443/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'

# Usare il token
TOKEN="<token-from-login>"
curl -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/users
```

---

## Dipendenze Go

| Pacchetto | Versione | Scopo |
|-----------|----------|-------|
| `modernc.org/sqlite` | v1.34.5 | SQLite pure Go (no CGO) |
| `github.com/golang-jwt/jwt/v5` | v5.2.1 | Token JWT |
| `github.com/spf13/cobra` | v1.8.1 | CLI framework |
| `github.com/spf13/viper` | v1.19.0 | Config YAML |
| `golang.org/x/crypto` | v0.21.0 | Bcrypt password hash |

---

## Note Tecniche

- **SQLite WAL**: Abilitato per concorrenza migliore
- **CGO_ENABLED=0**: Build statico, nessuna dipendenza C
- **Multi-stage Docker**: Image finale ~15MB (debian:trixie-slim + binario)
- **Admin auto-create**: Al primo avvio, se la tabella `users` e' vuota, crea l'admin dal config
- **Migrazioni**: Sistema incrementale con tabella `migrations` per tracciare applicazioni
