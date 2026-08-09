# Docker Visualizer — Operator Manual

Practical guide for running the app against **local Docker Desktop** and **remote VPS Engines** (one or many).

Architecture decisions: [`docs/adr/`](docs/adr/). Product overview / flags: [`README.md`](README.md).

Placeholders used below (replace with your values):

| Placeholder | Meaning |
|-------------|---------|
| `vps.example.com` | DNS name of the VPS (prefer this in cert SAN and `DOCKER_HOST`) |
| `198.51.100.77` | Your **public** client IP (office/home); look up via https://ifconfig.me |
| `C:\certs\my-vps` | Folder on Windows with client TLS files |
| `root` | SSH user on the VPS (often `root` or `ubuntu`) |

Do **not** commit real hostnames, IPs, or `*.pem` files into git.

---

## 1. Quick start (local)

```powershell
.\scripts\build-exe.ps1          # bumps SemVer patch + embeds UI
.\bin\docker-visualizer.exe
```

Open http://127.0.0.1:8080 — Docker Desktop must be running.

Version shows in the header/footer (`vX.Y.Z`) and in **Settings**.

### Local UI listen address / port

The HTTP server bind is **`--listen`** (not the Docker Engine port).

| How | Example |
|-----|---------|
| Default | `127.0.0.1:8080` |
| Flag | `--listen 127.0.0.1:9090` |
| Env | `DOCKER_VISUALIZER_LISTEN=127.0.0.1:9090` |
| Script | `.\scripts\run-exe.ps1 -Listen "127.0.0.1:9090"` |

```powershell
.\bin\docker-visualizer.exe --listen 127.0.0.1:9090
```

Non-loopback listen (e.g. `0.0.0.0:8080`) **requires** `--auth-token` / `DOCKER_VISUALIZER_AUTH_TOKEN` (ADR-013). Set the token in UI **Settings**.

Remote Docker Engine port (`2376`) is configured only via `--docker-host` / `--docker-hosts` / discovery — not via `--listen`.

---

## 2. How the app finds Docker (ADR-010)

For a **single** host (no `--docker-hosts`), resolution order is:

1. `--docker-host` / `DOCKER_VISUALIZER_DOCKER_HOST`
2. `DOCKER_HOST`
3. Current Docker context (`~/.docker` / `%USERPROFILE%\.docker`)
4. Platform default (`npipe:////./pipe/docker_engine` on Windows, `unix:///var/run/docker.sock` on Linux/macOS)

TLS for `tcp://` uses the **same env as Docker CLI** (via Moby `WithTLSClientConfigFromEnv`):

| Variable | Purpose |
|----------|---------|
| `DOCKER_CERT_PATH` | Directory with `ca.pem`, `cert.pem`, `key.pem` |
| `DOCKER_TLS_VERIFY` | `1` to verify server (recommended) |
| `DOCKER_TLS` | Enable TLS when needed by the client stack |

Full remote walkthrough: [§4](#4-remote-vps--docker--step-by-step).

---

## 3. One host vs many (ADR-014)

| Mode | How | UI |
|------|-----|----|
| **One** Engine | omit `--docker-hosts`; use discovery or `--docker-host` | no Host picker |
| **Many** Engines | `--docker-hosts name=url,name2=url` | Host picker in the nav |

Rules:

- Hosts are configured **only at process start** (not from the UI — SSRF protection).
- Each host has its own inventory/stats collectors.
- API/WS use `?host=<name>` (omit → default).
- Inventories are **not** merged across hosts.

### Examples

**Single remote VPS (TLS):**

```powershell
$env:DOCKER_CERT_PATH = "C:\certs\my-vps"
$env:DOCKER_TLS_VERIFY = "1"
.\bin\docker-visualizer.exe --docker-host "tcp://vps.example.com:2376"
```

**Local Desktop + remote VPS:**

```powershell
$env:DOCKER_CERT_PATH = "C:\certs\my-vps"
$env:DOCKER_TLS_VERIFY = "1"
.\bin\docker-visualizer.exe --docker-hosts "local=npipe:////./pipe/docker_engine,vps=tcp://vps.example.com:2376"
```

In the UI, use the Host picker → `vps`.

**Two remotes (same client CA):**

```powershell
$env:DOCKER_CERT_PATH = "C:\certs\shared-ca"
$env:DOCKER_TLS_VERIFY = "1"
.\bin\docker-visualizer.exe --docker-hosts "vps1=tcp://vps1.example.com:2376,vps2=tcp://vps2.example.com:2376"
```

> **TLS limitation:** one process = one `DOCKER_CERT_PATH` / TLS env set. If VPS certificates differ, use a shared CA or run **separate** visualizer processes (different `--listen` ports + different cert env).

**LAN UI bind (auth required):**

```powershell
$env:DOCKER_CERT_PATH = "C:\certs\my-vps"
$env:DOCKER_TLS_VERIFY = "1"
.\bin\docker-visualizer.exe `
  --listen 0.0.0.0:8080 `
  --auth-token "long-random-secret" `
  --docker-host "tcp://vps.example.com:2376"
```

---

## 4. Remote VPS + Docker — step by step

Goal: expose Docker Engine on the VPS as **`tcp://vps.example.com:2376` with TLS + client certs**, allow only your public IP, verify with Docker CLI on the PC, then start Docker Visualizer.

Official background: [Protect the Docker daemon socket](https://docs.docker.com/engine/security/protect-the-docker-daemon-socket/).

**Do not** expose plain **2375** to the public Internet.

### 4.0 Preconditions

- [ ] SSH access to the VPS works (`ssh root@vps.example.com`)
- [ ] Docker Engine is already installed and runs containers on the VPS
- [ ] You know the DNS name you will put in the certificate SAN (here: `vps.example.com`)
- [ ] You know your current public IP (here: `198.51.100.77`)

---

### Part A — On the VPS (SSH)

#### A1. Create cert directory

```bash
sudo mkdir -p /etc/docker/certs
cd /etc/docker/certs
```

#### A2. Generate CA

```bash
sudo openssl genrsa -out ca-key.pem 4096
sudo openssl req -new -x509 -days 3650 -key ca-key.pem -sha256 -out ca.pem -subj "/CN=docker-ca"
```

#### A3. Generate server key + certificate

SAN **must** match how you will connect (`vps.example.com` and/or the VPS IP).

```bash
sudo openssl genrsa -out server-key.pem 4096
sudo openssl req -new -key server-key.pem -out server.csr -subj "/CN=vps.example.com"

sudo tee server-ext.cnf >/dev/null <<'EOF'
subjectAltName = DNS:vps.example.com,IP:127.0.0.1
extendedKeyUsage = serverAuth
EOF

sudo openssl x509 -req -days 3650 -sha256 \
  -in server.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
  -out server-cert.pem -extfile server-ext.cnf
```

If you connect by IP instead of DNS, put that IP in `subjectAltName` (e.g. `IP:203.0.113.50`).

#### A4. Generate client key + certificate (for PC / visualizer)

```bash
sudo openssl genrsa -out key.pem 4096
sudo openssl req -new -key key.pem -out client.csr -subj "/CN=docker-client"

sudo tee client-ext.cnf >/dev/null <<'EOF'
extendedKeyUsage = clientAuth
EOF

sudo openssl x509 -req -days 3650 -sha256 \
  -in client.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
  -out cert.pem -extfile client-ext.cnf
```

Permissions:

```bash
sudo chmod 0400 ca-key.pem server-key.pem key.pem
sudo chmod 0444 ca.pem server-cert.pem cert.pem
```

**On the VPS keep:** `ca.pem`, `server-cert.pem`, `server-key.pem` (and CA key offline/safe).  
**For the PC you will copy:** `ca.pem`, `cert.pem`, `key.pem`.

#### A5. Create `/etc/docker/daemon.json`

If the file is missing (`No such file or directory`), that is **normal** — create it:

```bash
sudo tee /etc/docker/daemon.json >/dev/null <<'EOF'
{
  "hosts": ["unix:///var/run/docker.sock", "tcp://0.0.0.0:2376"],
  "tls": true,
  "tlsverify": true,
  "tlscacert": "/etc/docker/certs/ca.pem",
  "tlscert": "/etc/docker/certs/server-cert.pem",
  "tlskey": "/etc/docker/certs/server-key.pem"
}
EOF

sudo cat /etc/docker/daemon.json
```

If `daemon.json` already existed, **merge** keys carefully — do not wipe unrelated settings.

#### A6. Ubuntu / systemd: avoid `-H` conflict

On Ubuntu, `dockerd` often fails to start if both the unit and `daemon.json` set hosts. Use an override so the unit only runs `/usr/bin/dockerd`:

```bash
sudo mkdir -p /etc/systemd/system/docker.service.d
sudo tee /etc/systemd/system/docker.service.d/override.conf >/dev/null <<'EOF'
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd
EOF

sudo systemctl daemon-reload
sudo systemctl restart docker
sudo systemctl status docker --no-pager
```

**VPS check — service:**

- [ ] `Active: active (running)`
- [ ] Drop-In shows `override.conf`
- [ ] Logs mention API listen on unix socket **and** `[::]:2376` or `0.0.0.0:2376`

**VPS check — port:**

```bash
sudo ss -lntp | grep 2376
```

Expect a `LISTEN` line on `*:2376` (or `0.0.0.0:2376`) owned by `dockerd`.

If restart failed:

```bash
sudo journalctl -u docker -n 50 --no-pager
```

#### A7. Firewall (UFW) — only your public IP

Replace `198.51.100.77` with your real public IP.

```bash
sudo ufw allow OpenSSH
sudo ufw allow from 198.51.100.77 to any port 2376 proto tcp
```

Enable UFW. Interactive `Proceed with operation (y|n)?` can abort if stdin is awkward; prefer:

```bash
echo y | sudo ufw enable
sudo ufw status verbose
```

**VPS check — UFW:**

- [ ] `Status: active`
- [ ] `22/tcp` (OpenSSH) allowed (so you do not lock yourself out)
- [ ] `2376/tcp ALLOW IN 198.51.100.77` (not `Anywhere`)
- [ ] Default incoming is deny (typical)

If the cloud provider has a **separate** network firewall / security group, also allow `2376/tcp` **only** from `198.51.100.77` there.

#### A8. Stage client certs for download

`scp` as a normal user often cannot read `0400` files under `/etc/docker/certs`. Copy to `/tmp` temporarily:

```bash
sudo cp /etc/docker/certs/ca.pem /tmp/
sudo cp /etc/docker/certs/cert.pem /tmp/
sudo cp /etc/docker/certs/key.pem /tmp/
sudo chmod 644 /tmp/ca.pem /tmp/cert.pem /tmp/key.pem
ls -l /tmp/ca.pem /tmp/cert.pem /tmp/key.pem
```

**VPS check:** three files, readable (`-rw-r--r--`).

---

### Part B — On the Windows PC

#### B1. Create local cert folder

```powershell
New-Item -ItemType Directory -Force -Path C:\certs\my-vps
```

#### B2. Download client files (`scp`)

```powershell
scp root@vps.example.com:/tmp/ca.pem   C:\certs\my-vps\
scp root@vps.example.com:/tmp/cert.pem C:\certs\my-vps\
scp root@vps.example.com:/tmp/key.pem  C:\certs\my-vps\
dir C:\certs\my-vps
```

When `scp` asks for a password, that is the **SSH password for that user** (same as `ssh root@vps.example.com`). Characters are not echoed. If you normally use an SSH key:

```powershell
scp -i $env:USERPROFILE\.ssh\id_rsa root@vps.example.com:/tmp/ca.pem C:\certs\my-vps\
```

**PC check:** `C:\certs\my-vps` contains `ca.pem`, `cert.pem`, `key.pem`.

#### B3. Remove staged copies on the VPS

```bash
sudo rm /tmp/ca.pem /tmp/cert.pem /tmp/key.pem
```

#### B4. Verify Docker CLI → remote Engine

Docker Desktop’s active context (e.g. `desktop-linux`) often **ignores** `DOCKER_HOST`. Switch to `default` first (or use explicit `--host` flags).

```powershell
$env:DOCKER_HOST = "tcp://vps.example.com:2376"
$env:DOCKER_CERT_PATH = "C:\certs\my-vps"
$env:DOCKER_TLS_VERIFY = "1"

docker context use default
docker version
docker ps
```

**PC check — success:**

- [ ] `Context: default`
- [ ] **Server** is remote Engine (e.g. `Docker Engine - Community`), **not** `Docker Desktop …`
- [ ] Engine version / OSArch look like the VPS
- [ ] `docker ps` lists containers that exist on the VPS (not only local Desktop)

**Wrong (still local Desktop):** Server line says `Docker Desktop` and/or Context is `desktop-linux` even though you set `DOCKER_HOST`. Fix with `docker context use default`, or:

```powershell
docker --host tcp://vps.example.com:2376 `
  --tlsverify `
  --tlscacert C:\certs\my-vps\ca.pem `
  --tlscert C:\certs\my-vps\cert.pem `
  --tlskey C:\certs\my-vps\key.pem `
  version
```

**PC check — failure modes:**

| Symptom | Likely cause |
|---------|--------------|
| Timeout / connection refused | UFW/cloud firewall; daemon not on 2376; wrong IP |
| x509 / certificate / TLS errors | SAN ≠ hostname in URL; wrong `DOCKER_CERT_PATH`; missing files |
| Permission / bad certificate | Client cert not signed by same CA; swapped server/client files |

Do **not** start the visualizer until `docker version` shows the remote Server.

#### B5. Optional connectivity probe from PC

```powershell
Test-NetConnection vps.example.com -Port 2376
```

`TcpTestSucceeded : True` means the port is reachable (TLS handshake is a separate step).

---

### Part C — Start Docker Visualizer

Keep the same TLS env in that PowerShell session.  
One process = one `DOCKER_CERT_PATH` (all listed remotes must trust that client CA, or run separate processes — see §3).

```powershell
cd <repo-root>
$env:DOCKER_CERT_PATH = "C:\certs\my-vps"
$env:DOCKER_TLS_VERIFY = "1"
```

**One remote only:**

```powershell
.\bin\docker-visualizer.exe --docker-host "tcp://vps.example.com:2376"
```

**Several remotes only** (Host picker in UI):

```powershell
.\bin\docker-visualizer.exe --docker-hosts "vps1=tcp://vps1.example.com:2376,vps2=tcp://vps2.example.com:2376"
```

**Local Desktop + one remote:**

```powershell
.\bin\docker-visualizer.exe --docker-hosts "local=npipe:////./pipe/docker_engine,vps=tcp://vps.example.com:2376"
```

**Local Desktop + several remotes:**

```powershell
.\bin\docker-visualizer.exe --docker-hosts "local=npipe:////./pipe/docker_engine,vps1=tcp://vps1.example.com:2376,vps2=tcp://vps2.example.com:2376"
```

If the EXE is missing: `.\scripts\build-exe.ps1`.

Open http://127.0.0.1:8080 (or your `--listen` port).

**App checks:**

- [ ] Status shows connected for the selected host
- [ ] Dashboard / containers match `docker ps` for that Engine
- [ ] With `--docker-hosts`, Host picker lists every name (`vps1`, `vps2`, `local`, …) and switching reloads inventory
- [ ] Settings shows expected version / endpoint
- [ ] Optional: `GET http://127.0.0.1:8080/api/v1/ready` (and `?host=vps2` when multi-host)

---

### 4.8 Multi-VPS checklist

- [ ] Each Engine uses TLS (or is only on a private network)
- [ ] `--docker-hosts` has unique `name=` values
- [ ] One cert env works for all listed `tcp://` hosts **or** one process per cert set
- [ ] Host picker works
- [ ] Port **2375** is not public

---

## 5. Troubleshooting

| Symptom | Likely cause | What to do |
|---------|--------------|------------|
| `daemon.json` missing | Fresh install | Create file (§A5) — OK |
| `docker.service` fails after `hosts` in JSON | systemd `-H` conflict (Ubuntu) | §A6 override; `journalctl -u docker` |
| `ss` shows no 2376 | Daemon not listening | Fix daemon.json / restart; read logs |
| `ufw enable` → Aborted / inactive | Interactive prompt failed | `echo y \| sudo ufw enable` |
| `scp` asks for password | SSH auth | Same password/key as `ssh` to the VPS |
| `scp` permission denied on `/etc/docker/certs` | File mode 0400 | Stage via `/tmp` (§A8) |
| `docker version` shows Docker Desktop | Context overrides env | `docker context use default` or explicit `--host` (§B4) |
| `Docker endpoint not found` / named pipe | Desktop off / wrong endpoint | Start Desktop; check `--docker-host` |
| TLS / x509 errors | Bad certs / SAN / path | Align SAN with URL; retest CLI |
| Timeout / connection refused | Firewall / not listening | §A6–A7; cloud SG; `Test-NetConnection` |
| Unauthorized on visualizer API | Non-loopback listen, no token | `--auth-token` + Settings |
| Host picker missing | Single host only | Expected; add `--docker-hosts` |
| Second VPS TLS fails | Different client certs | Separate process or shared CA (§3) |

Actionable Engine errors are classified in-app (permission, daemon down, TLS) — see status banner.

---

## 6. Related docs

| Doc | Role |
|-----|------|
| [`README.md`](README.md) | Features, flags, build |
| [`docs/adr/010-endpoint-discovery.md`](docs/adr/010-endpoint-discovery.md) | Discovery decision |
| [`docs/adr/014-multi-host.md`](docs/adr/014-multi-host.md) | Multi-host decision |
| [`docs/hardening.md`](docs/hardening.md) | Auth, listen, limits |
| [`docs/cli-companion.md`](docs/cli-companion.md) | Commands / snapshots / diagnostics |
