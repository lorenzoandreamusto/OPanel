# Test OPanel su Docker

Simula una VPS Debian 13 pulita con installazione completa.

## Deploy

```bash
# Rimuovi container precedenti
docker rm -f opanel 2>/dev/null

# Avvia Debian 13 pulito
docker run -d -p 8443:8443 --name opanel debian:trixie-slim sleep infinity

# Copia i file dentro
docker cp . opanel:/tmp/opanel

# Installa tutto (Nessun systemd in Docker → niente service/firewall)
docker exec -it opanel bash -c "cd /tmp/opanel && bash install.sh"

# Avvia opaneld (senza systemd, va avviato manualmente)
docker exec -d opanel /opt/opanel/bin/opaneld server --config /etc/opanel/config.yaml
```

## Verifica

```bash
# Health check
curl http://localhost:8443/api/health

# Prendi la password generata
docker exec opanel cat /etc/opanel/config.yaml

# Login
curl -X POST http://localhost:8443/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"PASSWORD"}'
```

## Crea un dominio

```bash
curl -X POST http://localhost:8443/api/domains \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"tuosito.com"}'
```

## Cleanup

```bash
docker rm -f opanel
```
