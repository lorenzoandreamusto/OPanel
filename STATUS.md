# OPanel - Stato del Progetto

**Data:** 2026-06-09
**Sprint attuale:** 4 (completato)
**Target OS:** Debian 13 (Trixie)

---

## Stato Attuale

OPanel e' un clone di Plesk Obsidian scritto in Go. Attualmente e' stato completato lo **Sprint 4** con frontend MVP Vue 3.

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
- `GET /api/databases` - Lista database
- `GET /api/databases/:id` - Dettaglio database
- `POST /api/databases` - Crea database
- `DELETE /api/databases/:id` - Elimina database (admin only)
- `POST /api/databases/:id/users` - Crea utente database
- `DELETE /api/databases/:id/users/:userId` - Elimina utente database
- `PUT /api/databases/:id/users/:userId` - Aggiorna utente database

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
- Schema: tabelle `users`, `refresh_tokens`, `domains`, `migrations`

#### Gestione Domini (Sprint 2)
- `GET /api/domains` - Lista domini (tutti per admin,propri per user)
- `GET /api/domains/{id}` - Dettaglio dominio
- `POST /api/domains` - Crea dominio (crea utente Linux, directory, config Nginx)
- `PUT /api/domains/{id}` - Aggiorna stato dominio (active/suspended/pending)
- `DELETE /api/domains/{id}` - Elimina dominio (admin only, rimuove tutto)

#### Sistema Linux (Sprint 2)
- Generazione utenti Linux con `useradd` (home in `/var/www/vhosts/`)
- Utenti nel gruppo `opanel_users`
- Chroot SFTP configurato in `/etc/ssh/sshd_config`
- Struttura directory: `httpdocs/`, `logs/`, `tmp/`
- Ownership corretta sugli header di directory

#### Template Engine Nginx (Sprint 2)
- Template Go per generazione config Nginx virtual host
- Security headers (X-Frame-Options, X-Content-Type-Options, X-XSS-Protection)
- Gestione PHP-FPM via socket UNIX
- Blocco file sensibili (.env, .log, .sql, .conf)
- Cache statica per asset (30 giorni)

#### PHP-FPM (Sprint 3)
- Template Go per generazione pool config PHP-FPM
- Pool dedicato per dominio con isolamento utente
- Socket UNIX per ogni dominio (`/run/php/php8.2-fpm-op_{domain}.sock`)
- Configurazione PM dynamic (5 max children, 2 start servers)
- Limiti risorse per pool (128M memory, 128M upload, 300s timeout)
- Security hardening (disabled functions, expose_php off)
- Reload automatico PHP-FPM dopo creazione/modifica pool

#### MariaDB (Sprint 3)
- Connessione MariaDB via Unix socket (root, no password)
- API CRUD database:
  - `GET /api/databases` - Lista database (admin: tutti, user: propri)
  - `GET /api/databases/{id}` - Dettaglio database
  - `POST /api/databases` - Crea database (MariaDB + tracking SQLite)
  - `DELETE /api/databases/{id}` - Elimina database (admin only)
- API gestione utenti database:
  - `POST /api/databases/{id}/users` - Crea utente MariaDB + grant
  - `DELETE /api/databases/{id}/users/{userId}` - Elimina utente MariaDB
  - `PUT /api/databases/{id}/users/{userId}` - Cambia password/privilegi
- Tracking database e utenti in SQLite (tabelle `databases`, `database_users`)
- Ownership check: user vede solo i propri database

#### CLI (Cobra)
- `opaneld server` - Avvia il server
- `opaneld server --config /path/to/config.yaml` - Config personalizzata
- `opaneld version` - Mostra versione
- `opaneld user create` - Stub
- `opaneld user list` - Stub

#### Configurazione
- File YAML con viper
- Strutture: `server`, `database`, `jwt`, `admin`, `paths`, `system`
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
│   ├── handler/database.go       # CRUD database
│   ├── handler/domain.go         # CRUD domini
│   ├── handler/health.go         # Health check
│   ├── handler/user.go           # CRUD utenti
│   ├── jwt/jwt.go                # Generazione/validazione JWT
│   ├── middleware/auth.go        # Auth middleware + admin guard
│   ├── middleware/logging.go     # Request logging
│   ├── model/database.go         # Modello dati Database
│   ├── model/domain.go           # Modello dati Domain
│   ├── model/user.go             # Modello dati User
│   ├── server/server.go          # HTTP server + bootstrap admin
│   ├── server/routes.go          # Registrazione routes + SPA serving
│   └── service/
│       ├── domain.go             # Logica business domini
│       ├── mariadb.go            # Gestione MariaDB
│       ├── nginx.go              # Template engine Nginx
│       ├── phpfpm.go             # Gestione PHP-FPM pools
│       └── system.go             # Operazioni Linux (useradd, chroot)
├── frontend/                     # Vue 3 SPA frontend
│   ├── src/
│   │   ├── main.ts              # Entry point
│   │   ├── App.vue              # Root component
│   │   ├── style.css            # Tailwind CSS styles
│   │   ├── types/index.ts       # TypeScript types
│   │   ├── api/index.ts         # API client
│   │   ├── stores/auth.ts       # Pinia auth store
│   │   ├── router/index.ts      # Vue Router + guards
│   │   ├── components/
│   │   │   ├── AppLayout.vue    # Main layout
│   │   │   ├── AppSidebar.vue   # Sidebar navigation
│   │   │   └── AppHeader.vue    # Top header bar
│   │   └── views/
│   │       ├── LoginView.vue    # Login page
│   │       ├── DashboardView.vue # Dashboard with stats
│   │       ├── DomainsView.vue  # Domain management
│   │       ├── DatabasesView.vue # Database management
│   │       ├── UsersView.vue    # User management
│   │       └── SettingsView.vue # Settings page
│   ├── package.json             # NPM dependencies
│   ├── vite.config.ts           # Vite build config
│   ├── tailwind.config.js       # Tailwind CSS config
│   ├── tsconfig.json            # TypeScript config
│   └── index.html               # HTML entry point
├── static/                       # Built frontend (output of npm run build)
├── templates/
│   ├── nginx/
│   │   ├── default.conf.template # Template config Nginx
│   │   └── index.html           # Default welcome page
│   └── phpfpm/
│       └── pool.conf.template   # Template pool PHP-FPM
├── config.example.yaml           # Config di esempio
├── install.sh                     # Installer script per Debian/Ubuntu
├── Dockerfile                    # Multi-stage Debian 13 + frontend
├── docker-compose.yml            # Orchestrazione Docker
├── entrypoint.sh                 # Startup multi-servizio
├── Makefile                      # Build commands (include frontend)
├── Plan.md                       # Architettura completa
├── STATUS.md                     # Questo file
├── README.md                     # Documentazione
├── go.mod
└── go.sum
```

---

## Cosa Manca (Sprint 3-7)

### Installer Script ✅
- [x] Script `install.sh` che installa OPanel su Debian/Ubuntu vergine
- [x] Controllo OS (`/etc/os-release`) e requisiti hardware minimi (RAM, disco)
- [x] Aggiornamento repository (`apt update && apt upgrade -y`)
- [x] Installazione dipendenze di sistema: `nginx`, `php-fpm`, `mariadb-server`, `postfix`, `dovecot`, `rspamd`, `bind9`, `ufw`, `fail2ban`, `openssh-server`, `tar`, `wget`, `curl`, `sudo`, `ca-certificates`
- [x] Creazione struttura directory OPanel: `/opt/opanel/{bin,db,templates,extensions,backups,ssl}`
- [x] Copia/binaria di `opaneld` in `/opt/opanel/bin/` (attualmente build locale, futuro: download da GitHub release)
- [x] Creazione database SQLite vuoto in `/opt/opanel/db/opanel.db`
- [x] Generazione password admin casuale sicura e salvataggio in config
- [x] Generazione file config `/etc/opanel/config.yaml` con parametri generati
- [x] Creazione servizio Systemd `/etc/systemd/system/opanel.service` (se systemd disponibile)
- [x] `systemctl enable --now opanel` (se systemd disponibile)
- [x] Configurazione iniziale UFW (apertura porte 80, 443, 22, 8443) (se systemd disponibile)
- [x] Stampa a schermo delle credenziali di accesso e URL
- [x] Rilevamento ambienti Docker/container (skip systemd/UFW)

### Sprint 3 - PHP & Database ✅
- [x] Integrazione PHP-FPM (pool per sito, socket UNIX)
- [x] Installazione stack MariaDB locale
- [x] API per creare/eliminare database
- [x] Gestione utenti database con grant

### Sprint 4 - Frontend MVP ✅
- [x] Setup Vue 3 + TypeScript + Vite + Tailwind CSS
- [x] Layout: Sidebar fissa, Header, Area contenuto
- [x] Login page
- [x] Dashboard domini (con statistiche e listings)
- [x] Connessione API backend (client API completo)
- [x] Router con auth guards
- [x] Pinia store per autenticazione
- [x] Gestione domini CRUD (crea, elimina, sospendi/attiva)
- [x] Gestione database CRUD
- [x] Gestione utenti CRUD (admin only)
- [x] Dark mode nativo (tema Plesk-inspired)
- [x] Go backend serve SPA con fallback a index.html
- [x] Dockerfile multi-stage con build frontend integrato
- [x] Makefile aggiornato con target frontend

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

### Installazione su Server

```bash
# Copia la cartella OPanel sul server
scp -r OPanel/ root@your-server:/tmp/opanel

# Esegui l'installer
ssh root@your-server
cd /tmp/opanel
bash install.sh
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

# Lista domini
curl -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/domains

# Crea dominio
curl -X POST http://localhost:8443/api/domains \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"example.com"}'

# Lista database
curl -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/databases

# Crea database
curl -X POST http://localhost:8443/api/databases \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"mydb"}'

# Crea utente database
curl -X POST http://localhost:8443/api/databases/1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username":"dbuser","password":"secret","privileges":"ALL PRIVILEGES"}'
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
| `github.com/go-sql-driver/mysql` | v1.8.1 | Driver MariaDB/MySQL |

---

## Note Tecniche

- **SQLite WAL**: Abilitato per concorrenza migliore
- **CGO_ENABLED=0**: Build statico, nessuna dipendenza C
- **Multi-stage Docker**: Image finale ~15MB (debian:trixie-slim + binario)
- **Admin auto-create**: Al primo avvio, se la tabella `users` e' vuota, crea l'admin dal config
- **Migrazioni**: Sistema incrementale con tabella `migrations` per tracciare applicazioni
- **Config Estesa**: Sezioni `paths` e `system` per percorsi e configurazioni Linux
