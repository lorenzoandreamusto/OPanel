# AGENTS.md - Istruzioni per Agenti

## Regola fondamentale: TESTARE DOPO OGNI MODIFICA

Dopo **ogni** modifica al codice (bugfix, nuovo endpoint, refactoring, modifica ai servizi):

1. **Backend:** `go build ./cmd/opaneld && go vet ./...` per verificare che compili senza errori
2. **Frontend:** `cd frontend && npm run typecheck` per verificare TypeScript
3. **Rebuild Docker:** `docker compose up --build -d`
4. **Riavviare il container:** (già incluso nel comando sopra, il container viene ricreato automaticamente)
5. **Rieseguire TUTTI i test esistenti** per verificare che nulla sia rotto (regressione)
6. **Aggiungere nuovi test** per ogni funzionalità aggiunta

### Test completi: test.ps1

**Il file ufficiale dei test e' `test.ps1`** nella root del progetto. Eseguire dopo ogni modifica:

```bash
pwsh -File test.ps1
```

### Quando aggiungere nuovi test

- **Nuovo endpoint API**: aggiungere test per happy path, errori, validazione input
- **Nuovo servizio**: testare integrazione con Docker (MariaDB, Nginx, PHP-FPM)
- **Nuova tabella migration**: verificare che il DB venga creato correttamente
- **Nuova configurazione**: testare che i valori default funzionino

### Note sulla piattaforma

- Container Docker basato su Debian Trixie (slim)
- PHP 8.4, MariaDB (Debian default), Nginx
- PHP-FPM pool dir: `/etc/php/8.4/fpm/pool.d`
- MariaDB socket: `/var/run/mysqld/mysqld.sock`
- Go 1.24 (go.mod), `CGO_ENABLED=0`
- Le DDL MariaDB (CREATE USER, ALTER USER) usano `fmt.Sprintf` con escape manuale (non parametrizzabili)
