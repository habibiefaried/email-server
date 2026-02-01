# Email Server

A simple SMTP server implementation in Go using the `go-smtp` library.

## What It Does

This server:
- ✅ Accepts incoming SMTP connections on port 25
- ✅ Logs sender, recipient, and email body to console
- ✅ Automatically generates DNS configuration needed for email delivery
- ⚠️ **Does NOT** store emails or provide IMAP/POP3 access (for testing/learning only)

## Quick Start

### 1. Build
```bash
go build -o email-server
```

### 2. Run (Requires root/Administrator)
```bash
# Linux/Mac (need sudo for port 25)
sudo FQDN=mail.yourdomain.com PUBLIC_IP=1.2.3.4 ./email-server

# Windows (run as Administrator)
$env:FQDN="mail.yourdomain.com"; $env:PUBLIC_IP="1.2.3.4"; .\email-server.exe
```

### 3. Set DNS Records
The server will print the exact DNS records you need to configure. Example output:
```
TYPE   | NAME                      | VALUE
A      | mail.yourdomain.com       | 1.2.3.4
MX     | yourdomain.com            | mail.yourdomain.com (priority: 10)
PTR    | 4.3.2.1.in-addr.arpa      | mail.yourdomain.com
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `FQDN` | Yes | Full domain name (e.g., `mail.yourdomain.com`) |
| `PUBLIC_IP` | No | Public IP where server is running (e.g., `1.2.3.4`) |

## Port Requirements

- **Port 25**: Default SMTP, requires root/administrator privileges
- Runs on all interfaces (0.0.0.0:25)

## Troubleshooting

**Error: "permission denied"**
- You need root (Linux) or Administrator (Windows) privileges
- Port 25 may already be in use

**Emails not being received**
- DNS records (A, MX, PTR) must be configured
- PTR record is controlled by your ISP
- Check MX record points to your FQDN
- Verify A record points to your public IP

**Testing your PTR record:**
```bash
nslookup -type=PTR 1.2.3.4
# or
dig -x 1.2.3.4
```

Should return: `mail.yourdomain.com`

**Why it matters:** Many mail servers check PTR records to verify sender legitimacy. Without proper PTR records, your emails are more likely to be marked as spam.
