# Docker Visualizer — Operator Manual

Practical guide for running the app against **local Docker Desktop** and **remote VPS Engines** (one or many).

Architecture decisions: [`docs/adr/`](docs/adr/). Product overview / flags: [`README.md`](README.md).

---

## 1. Quick start (local)

```powershell
.\scripts\build-exe.ps1          # bumps SemVer patch + embeds UI
.\bin\docker-visualizer.exe
```

Open http://127.0.0.1:8080 — Docker Desktop must be running.

Version shows in the header/footer (`vX.Y.Z`) and in **Settings**.

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

See detailed VPS checklist: [§4](#4-remote-vps--docker--checklist).

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
$env:DOCKER_CERT_PATH = "C:\certs\vps1"
$env:DOCKER_TLS_VERIFY = "1"
.\bin\docker-visualizer.exe --docker-host "tcp://203.0.113.10:2376"
```

**Local + two VPS:**

```powershell
$env:DOCKER_CERT_PATH = "C:\certs\shared-or-vps1"
$env:DOCKER_TLS_VERIFY = "1"
.\bin\docker-visualizer.exe --docker-hosts "local=npipe:////./pipe/docker_engine,vps1=tcp://203.0.113.10:2376,vps2=tcp://198.51.100.20:2376"
```

> **TLS limitation:** one process = one `DOCKER_CERT_PATH` / TLS env set. If VPS certificates differ, either use a shared CA or run **separate** visualizer processes (different ports + cert env).

**LAN UI bind (auth required):**

```powershell
.\bin\docker-visualizer.exe `
  --listen 0.0.0.0:8080 `
  --auth-token "long-random-secret" `
  --docker-host "tcp://203.0.113.10:2376"
```

Then set the token in **Settings** (stored in `localStorage`).

---

## 4. Remote VPS + Docker — checklist

Do this **on the VPS** before pointing the visualizer at it.

### 4.1 Security baseline (required)

- [ ] Prefer **TLS on port 2376**. Do **not** expose plain `2375` to the public Internet.
- [ ] Firewall: allow `2376/tcp` only from your office/VPN IP (or private network), not `0.0.0.0/0`.
- [ ] Keep Docker Engine updated; restrict who can SSH to the VPS.
- [ ] Treat client certs (`cert.pem` / `key.pem`) as secrets — never commit them.

### 4.2 Create TLS material (on VPS or a secure CA host)

Typical Docker remote setup (OpenSSL). Adjust CN/SAN to the VPS hostname or IP.

```bash
# Example layout on the operator PC (or secure workstation):
#   ca.pem  ca-key.pem
#   server-cert.pem  server-key.pem
#   cert.pem  key.pem   (client — for visualizer / docker CLI)
```

Official Docker docs: [Protect the Docker daemon socket](https://docs.docker.com/engine/security/protect-the-docker-daemon-socket/).

Copy to VPS, e.g. `/etc/docker/certs/`:

- `ca.pem`, `server-cert.pem`, `server-key.pem`

Copy to the machine that runs visualizer, e.g. `C:\certs\vps1\`:

- `ca.pem`, `cert.pem`, `key.pem`

### 4.3 Configure Docker daemon (on VPS)

`/etc/docker/daemon.json` (merge with existing keys carefully):

```json
{
  "hosts": ["unix:///var/run/docker.sock", "tcp://0.0.0.0:2376"],
  "tls": true,
  "tlsverify": true,
  "tlscacert": "/etc/docker/certs/ca.pem",
  "tlscert": "/etc/docker/certs/server-cert.pem",
  "tlskey": "/etc/docker/certs/server-key.pem"
}
```

On systemd, ensure `-H` flags in the unit don’t conflict with `hosts` in `daemon.json` (often you must remove duplicate `-H fd://` overrides — see Docker docs for your distro).

```bash
sudo systemctl daemon-reload
sudo systemctl restart docker
sudo ss -lntp | grep 2376
```

### 4.4 Firewall (on VPS)

```bash
# Example: UFW — only your IP
sudo ufw allow OpenSSH
sudo ufw allow from YOUR.OFFICE.IP.ADDR to any port 2376 proto tcp
sudo ufw enable
sudo ufw status
```

### 4.5 Verify with Docker CLI (from your PC)

```powershell
$env:DOCKER_HOST = "tcp://YOUR_VPS_IP:2376"
$env:DOCKER_CERT_PATH = "C:\certs\vps1"
$env:DOCKER_TLS_VERIFY = "1"
docker version
docker ps
```

If CLI fails, visualizer will fail the same way — fix certs/firewall first.

### 4.6 Point Docker Visualizer at the VPS

```powershell
$env:DOCKER_CERT_PATH = "C:\certs\vps1"
$env:DOCKER_TLS_VERIFY = "1"
.\bin\docker-visualizer.exe --docker-host "tcp://YOUR_VPS_IP:2376"
```

Check:

- [ ] Status banner shows **connected**
- [ ] Dashboard lists containers from the VPS
- [ ] Settings → version / docker endpoint looks right
- [ ] `GET /api/v1/ready` returns ready (or clear error message)

### 4.7 Multi-VPS checklist

- [ ] Each Engine has TLS (or is only reachable on a private network)
- [ ] `--docker-hosts` lists every Engine with a unique `name=`
- [ ] Cert env works for **all** listed `tcp://` hosts **or** you run one process per cert set
- [ ] Host picker switches inventory as expected
- [ ] You did **not** open `2375` publicly

---

## 5. Troubleshooting

| Symptom | Likely cause | What to do |
|---------|--------------|------------|
| `Docker endpoint not found` / named pipe | Desktop not running / wrong context | Start Docker Desktop; `docker context ls` |
| TLS / x509 errors | Bad `DOCKER_CERT_PATH`, wrong CA, hostname mismatch | Align SAN with URL; `DOCKER_TLS_VERIFY=1`; test with `docker version` |
| Timeout / connection refused | Firewall, Docker not listening on 2376 | §4.3–4.4 |
| Unauthorized on API | Non-loopback listen without token | `--auth-token` + Settings |
| Host picker missing | Only one host configured | Expected; add `--docker-hosts` |
| Second VPS TLS fails | Different client certs | Separate process or shared CA (§3) |

Actionable Engine errors are classified in-app (permission, daemon down, TLS) — see status banner / `collect` messages.

---

## 6. Related docs

| Doc | Role |
|-----|------|
| [`README.md`](README.md) | Features, flags, build |
| [`docs/adr/010-endpoint-discovery.md`](docs/adr/010-endpoint-discovery.md) | Discovery decision |
| [`docs/adr/014-multi-host.md`](docs/adr/014-multi-host.md) | Multi-host decision |
| [`docs/hardening.md`](docs/hardening.md) | Auth, listen, limits |
| [`docs/cli-companion.md`](docs/cli-companion.md) | Commands / snapshots / diagnostics |
