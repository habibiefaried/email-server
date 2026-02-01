
# Email Server

A simple SMTP server in Go using the `go-smtp` library.

## Features
- Accepts SMTP connections on port 25
- Logs sender, recipient, and email body to console
- Prints required DNS records for mail delivery
- Does NOT store or forward emails (for testing/learning only)

## Quick Start

### 1. Build
```sh
make build
```


### 2. Run (Requires root/Administrator)
```sh
# Linux/Mac
sudo MAIL_SERVERS="mail1.example.com,1.2.3.4:mail2.example.com,5.6.7.8" ./email-server.exe

# Windows (run as Administrator)
$env:MAIL_SERVERS="mail1.example.com,1.2.3.4:mail2.example.com,5.6.7.8"; .\email-server.exe
```

### 3. DNS Records
The server prints a table of required DNS records (A, MX, PTR) and their status. PTR is optional but recommended.


## Environment Variables
| Variable      | Required | Description                                                      |
|---------------|----------|------------------------------------------------------------------|
| MAIL_SERVERS  | Yes      | List of FQDN,IP pairs separated by `:` (see example above)        |

## Project Structure
- `cmd/email-server/main.go` — Entry point
- `internal/server/` — SMTP backend/session/server logic
- `internal/dnsutil/` — DNS validation and checking

## Clean Up
To remove the binary:
```sh
make clean
```

**Testing your PTR record:**
```bash
nslookup -type=PTR 1.2.3.4
# or
dig -x 1.2.3.4
```


Should return: your FQDN (e.g., `mail1.example.com`)

**Why it matters:** Many mail servers check PTR records to verify sender legitimacy. Without proper PTR records, your emails are more likely to be marked as spam.
