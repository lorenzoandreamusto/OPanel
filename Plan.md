Questo è il **Documento di Architettura e Implementazione Definitivo per OPanel**. 
OPanel è un clone di Plesk Obsidian scritto in Go.
---

# 🚀 OPanel: Piano di Implementazione Architetturale

## 1. Architettura di Base e Stack Tecnologico

OPanel funzionerà come un demone di sistema monolitico e indipendente, senza dipendere da server web esterni per la propria interfaccia, garantendo stabilità anche se Apache o Nginx crashano.

*   **Backend / Core:** Go (Golang, ultima versione). Fornirà un server web integrato (`net/http` o framework fiber/gin) per le API REST/WebSocket.
*   **Frontend UI:** **Vue.js 3** (Composition API) + **TypeScript** + **Tailwind CSS**. Scelto perché component-based, estremamente stabile se strutturato bene, leggero e perfetto per costruire le SPA (Single Page Applications) reattive.
*   **Database OPanel:** SQLite (via driver puro Go `modernc.org/sqlite` per evitare dipendenze CGO), file posizionato in `/opt/opanel/db/opanel.db`.
*   **Isolamento:** Utenti di sistema Linux (`useradd`), permessi POSIX rigidi, `chroot` per SFTP.
*   **Sistema Operativo Target Iniziale:** Debian 12 (Bookworm) / Ubuntu 24.04 LTS.

---

## 2. Struttura del File System e Isolamento Utenti (Chroot)

OPanel adotterà una struttura directory chiara e immutabile, simile a Plesk, per garantire sicurezza e ordine.

*   **Cartelle di OPanel:**
    *   `/opt/opanel/bin/opaneld` (Il demone Go)
    *   `/opt/opanel/templates/` (Template per vhost, DNS, mail)
    *   `/opt/opanel/extensions/` (Script delle estensioni)
    *   `/opt/opanel/backups/` (Punto di mount di default per i backup locali)
*   **Cartelle dei Siti Web (`/var/www/vhosts/`):**
    Quando viene creato il sito `demo.com`:
    1.  Go esegue `useradd -d /var/www/vhosts/demo.com -s /bin/false op_demo`.
    2.  Viene creata la struttura:
        *   `/var/www/vhosts/demo.com/httpdocs/` (Root pubblica del sito)
        *   `/var/www/vhosts/demo.com/logs/` (File di log specifici di Nginx/Apache per questo sito)
        *   `/var/www/vhosts/demo.com/tmp/` (Sessioni PHP, upload temp)
    3.  **Sicurezza Chroot SFTP:** Il file `/etc/ssh/sshd_config` verrà configurato da OPanel in modo nativo per ingabbiare gli utenti.
        ```sshdconfig
        Match Group opanel_users
            ChrootDirectory /var/www/vhosts/%u
            ForceCommand internal-sftp
            AllowTcpForwarding no
        ```

---

## 3. Gestione Web Server e Motori (PHP, Node, Go, .NET)

Il sistema supporterà configurazioni ibride. Nginx fungerà da reverse proxy frontend veloce, instradando le richieste al backend appropriato.

*   **Gestione PHP (FPM Mutliplo):**
    *   OPanel utilizzerà il repository `ondrej/php` (Debian) per installare PHP 7.4, 8.0, 8.1, 8.2, 8.3.
    *   Per ogni sito, OPanel crea un file "pool" dedicato in `/etc/php/8.x/fpm/pool.d/demo.com.conf`.
    *   **Isolamento totale:** Il pool PHP-FPM girerà come l'utente `op_demo` e ascolterà su un socket UNIX isolato (es. `/run/php/php8.2-fpm-op_demo.sock`). Nginx comunicherà direttamente con questo socket.
*   **Siti in Node.js, Go e .NET:**
    *   OPanel assegnerà una porta locale casuale libera (es. 3005).
    *   L'utente carica il codice. OPanel genera un servizio `systemd` in user-space (`systemctl --user`) o usa un process manager (come `pm2` wrapato in Go) per tenere in vita l'app sulla porta 3005.
    *   Nginx viene configurato da OPanel come Reverse Proxy (`proxy_pass http://127.0.0.1:3005`).
*   **Dominio, Sslip.io e SSL:**
    *   *Logica sslip.io:* Se l'utente seleziona "Locale", Go fa una query sull'interfaccia di rete, ottiene l'IP (es. `192.168.1.100`), e salva il dominio come `nomesito.192.168.1.100.sslip.io`. Se pubblico, usa l'IP WAN (ottenuto via API o routing).
    *   *Let's Encrypt:* Modulo Go interno (usando librerie come `lego` o `certmagic`) che gestisce le challenge HTTP-01 e DNS-01, salvando i certificati in `/opt/opanel/ssl/` e ricaricando Nginx/Apache.

---

## 4. Stack Mail (Seamless e Identico a Plesk)

Questo è il componente più critico per la stabilità. OPanel scriverà i file di configurazione nativi, interfacciandosi con servizi collaudati.

*   **Componenti:**
    *   **Postfix** (MTA - Invio/Ricezione SMTP).
    *   **Dovecot** (MDA/IMAP/POP3 - Lettura posta).
    *   **Rspamd** (Filtro Antispam moderno, veloce e gestione automatica firme DKIM).
*   **Integrazione Go & Mail:**
    *   OPanel userà file in formato map testuale o lo stesso SQLite integrato in `/opt/opanel/db/mail.db` per dire a Postfix e Dovecot quali sono gli utenti validi e le loro password crittografate (argon2 o bcrypt).
    *   *Creazione di una casella:* Quando crei `info@demo.com` da UI, Go inserisce il record nel database mail, genera la cartella `/var/www/vhosts/demo.com/mail/info/`, ricalcola i permessi e dice a Dovecot di ricaricarsi (`systemctl reload dovecot`).
    *   *Autoconfigurazione:* OPanel configurerà Rspamd per generare dinamicamente le chiavi DKIM alla creazione di un nuovo dominio mail, fornendo all'utente i record DNS TXT esatti da inserire (o inserendoli automaticamente nel DNS locale).

---

## 5. DNS, Database e Funzionalità Core

*   **Gestione DNS (Bind9):**
    *   Go utilizzerà template testuali per generare file di zona Bind9 (`/etc/bind/zones/db.demo.com`).
    *   Implementazione automatica di record di default alla creazione del dominio (A, CNAME per www, MX, TXT per SPF, SRV).
*   **Database (MariaDB):**
    *   Integrazione con MariaDB. Go si connette come root via socket locale.
    *   Creazione DB e Utenti: Quando l'utente richiede un DB, Go esegue le query SQL native: `CREATE DATABASE demo_db; CREATE USER 'demo_user'@'localhost' IDENTIFIED BY 'password'; GRANT ALL...`.
*   **File Manager Web:**
    *   Frontend: Un componente Vue 3 che simula un esplora risorse desktop.
    *   Backend: Endpoints REST (`/api/files/list`, `/api/files/upload`). 
    *   *Sicurezza:* Il processo Go (che gira da root) prima di leggere/scrivere un file in `/var/www/vhosts/demo.com/`, esegue una syscall per **cambiare il proprio UID/GID** temporaneamente a quello di `op_demo`. Questo garantisce che non vengano mai scritti file di root nelle cartelle degli utenti.
*   **One Click Installers (WordPress, ecc.):**
    *   Go esegue una routine asincrona: scarica il `.zip` o `.tar.gz` dal vendor ufficiale nella RAM, lo estrae nella `httpdocs`, crea il database, genera password casuali, inietta questi dati in `wp-config-sample.php`, rinomina il file e imposta i permessi corretti. Tutto in meno di 2 secondi.

---

## 6. Sicurezza, Performance Monitoring e Backup

*   **Firewall (UFW + Fail2Ban):**
    *   OPanel UI gestirà le regole UFW (Uncomplicated Firewall) sottostanti.
    *   Moduli pre-configurati per chiudere tutto e aprire solo 80, 443, 22, 21 (se legacy ftp serve), 25, 143, 465, 587, 993.
    *   Integrazione visiva di Fail2Ban: OPanel leggerà lo stato di Fail2Ban e permetterà il Ban/Unban manuale di IP tramite UI.
*   **Monitoring:**
    *   Goroutine dedicata che fa polling ogni X secondi di `/proc/stat`, `/proc/meminfo`, e I/O disco.
    *   Dati inviati al frontend via **WebSocket** per grafici live a 60fps (carico CPU, RAM, Network I/O).
*   **Terminal Web (SSH):**
    *   Libreria frontend: `Xterm.js`.
    *   Libreria backend Go: `creack/pty`. Go aprirà un terminale pseudo-tty. Se loggato come Admin, aprirà `bash` come root. Se loggato come cliente limitato, aprirà un processo ssh forzato nel chroot verso l'utente di quel dominio.
*   **Sistema di Backup:**
    *   Go orchestera archivi `tar.gz` zippati in streaming (per non saturare il disco).
    *   Esegue `mysqldump` dei DB del sito, inserisce l'export nel tarball insieme a `httpdocs` e alla posta.
    *   Supporto upload asincrono a S3 (Amazon, Cloudflare R2) nativo in Go.

---

## 7. Frontend UI / UX (Il Clone Moderno)

Il design in Vue 3 dovrà essere pixel-perfect, dark mode nativo, ed estremamente reattivo.

*   **Tema Generale:** Sfondo generale molto scuro (es. `#121212`), pannelli centrali grigio scuro (`#1E1E1E`). Testi bianchi/grigio chiaro. Pulsanti primari in **Light Blue** (es. `#00AEEF` o variazioni del tema Plesk).
*   **Layout:**
    *   *Sidebar Sinistra (Fissa):* Logo OPanel in alto. Menu navigazione (Home, Domini, Clienti, Strumenti e Impostazioni...). Icone lineari e pulite. Elemento attivo evidenziato col background grigio leggermente più chiaro e barra laterale azzurra.
    *   *Header (Top bar):* Ricerca globale (che chiama un'API Go in tempo reale), notifiche (badge per backup falliti, aggiornamenti disponibili), profilo utente.
    *   *Area Contenuto (Scrollable):* Pannelli "Card" con bordi arrotondati e leggere ombre.
*   **Esperienza Utente "Domain Dashboard":**
    *   Cliccando su un dominio, si apre la vista di gestione.
    *   A sinistra: Mini-card con statistiche (Disco occupato in MB/GB con progress bar, Traffico mensile).
    *   Al centro/Destra: Griglia di pulsanti grossi con icona (Impostazioni Hosting, PHP Settings, SSL/TLS, File Manager, Mail Accounts, Database).
    *   Nessun caricamento di pagina intero. Quando si clicca "File Manager", la view centrale cambia usando Vue Router con un'animazione di fade leggerissima, mantenendo la sidebar laterale intatta.

---

## 8. Sistema di Estensioni (Script-Based Hooks)

Per mantenere la promessa di espandibilità semplice e potente (punto 4), implementeremo un'architettura **Hook & Scripting**.

*   **Come funziona:** In `/opt/opanel/extensions/`, gli sviluppatori possono creare cartelle (es. `/opt/opanel/extensions/my_custom_cache/`).
*   **File di Manifesto (`extension.json`):** Definisce nome, versione, autore e, soprattutto, a quali "eventi" del pannello e a quali "pulsanti UI" si aggancia l'estensione.
*   **Gli Script:** L'estensione conterrà eseguibili (script Bash, file Python, binari Go, file JS eseguibili da Node).
*   **Esecuzione via Hooks:**
    *   Quando da UI crei un dominio, Go intercetta l'evento interno `domain.created`.
    *   Go cerca tutte le estensioni che hanno registrato un hook per `domain.created`.
    *   Go esegue lo script dell'estensione passando un JSON in *stdin* con i dati del dominio (Nome, IP, Path).
    *   Lo script fa il suo lavoro (es. contatta un'API esterna di fatturazione, pulisce una cache cloudflare personalizzata) ed esce con `exit 0`.
*   **UI Inject:** Tramite l'`extension.json`, un'estensione può dire a Vue.js (tramite l'API di OPanel) di aggiungere un nuovo pulsante nella Dashboard del Dominio, che aprirà una view iFrame HTML servita dall'estensione o un form standardizzato generato dinamicamente.

---

## 9. Installazione (Il punto di ingresso)

L'installer deve essere a prova di errore e automatizzato al 100%.

*   **Lo script `install.sh`:**
    1.  Controllo OS (`/etc/os-release`).
    2.  Check dei requisiti hardware minimi (RAM, Spazio disco).
    3.  Aggiornamento repository (`apt update && apt upgrade -y`).
    4.  Installazione pacchetti base (`wget, curl, tar, sudo, ca-certificates`).
    5.  Download dell'ultimo binario Go compilato dal repository GitHub di OPanel.
    6.  Setup cartelle (`mkdir -p /opt/opanel/...`).
    7.  Generazione password amministratore casuale e sicura e creazione Database SQLite vuoto.
    8.  Creazione del servizio Systemd:
        ```ini
        [Unit]
        Description=OPanel Control Panel Daemon
        After=network.target

        [Service]
        Type=simple
        ExecStart=/opt/opanel/bin/opaneld server
        Restart=on-failure
        User=root

        [Install]
        WantedBy=multi-user.target
        ```
    9.  `systemctl enable --now opanel`.
    10. Stampa a schermo:
        `Installazione completata! Accedi a https://[IP_SERVER]:8443`
        `Username: admin`
        `Password: [PASSWORD_GENERATA]`

---

## Riepilogo Sviluppo (Da dove partire)

Per realizzare questo enorme progetto, lo sviluppo va frammentato (Sprint):

1.  **Sprint 1 (Fondamenta Go):** Setup del demone Go, routing HTTP, middleware di autenticazione (JWT), connessione SQLite. Interfaccia CLI base.
2.  **Sprint 2 (Web e File System):** Logica di generazione utenti Linux, chroot. Motore di templating Go per Nginx. Creazione/Cancellazione domini fisica su disco.
3.  **Sprint 3 (PHP & Database):** Integrazione moduli PHP-FPM, installazione stack MariaDB locale. API per creare DB.
4.  **Sprint 4 (Frontend MVP):** Setup del progetto Vue 3 + Tailwind. Creazione layout (Sidebar, Header). Connessione API per login e listato domini.
5.  **Sprint 5 (Posta e DNS):** Scrittura controller per Bind9. Scrittura controller integrato per Postfix+Dovecot (la parte più complessa a livello sistemistico).
6.  **Sprint 6 (Strumenti avanzati):** File Manager web, Terminale SSH nel browser (Xterm), Installatore WordPress 1-click, Sistema di Backup locale/S3.
7.  **Sprint 7 (Estensioni e Polish):** Motore di esecuzione script per le estensioni, check UI/UX, testing intensivo su VPS vergini.

Questo piano copre ogni singola richiesta al massimo dettaglio architetturale, fornendo una road-map chiara per te e per chiunque svilupperà il codice (Go backend / Vue frontend). Non ci sono zone d'ombra: è un'architettura enterprise.
