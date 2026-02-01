## Live Testing Screenshots

This section demonstrates real-world usage and results of running the email server, including DNS checks and email reception.

### 1. Server Startup and DNS Verification

![Server Startup and DNS Verification](screenshots/1.png)

- The server is started with the `MAIL_SERVERS` environment variable.
- It prints a table for each FQDN and IP pair, showing the status of A, MX, and PTR records.
- If any record is missing or incorrect, the server will not start and will print a clear error.

### 2. Sending and Receiving Email

#### 2.1. Sending an Email

![Sending an Email](screenshots/2.1.png)

- An email is sent to the server using a mail client or command-line tool.

#### 2.2. Email Saved to File

![Email Saved to File](screenshots/2.2.png)

- The server saves the received email to a file in the `emails/` directory.
- The filename is a timestamp in nanoseconds, ensuring uniqueness.

#### 2.3. Log Output

![Log Output](screenshots/2.3.png)

- The server logs a summary: `from: <sender>, saved in <filename>`
- No email content is printed to the console for privacy and clarity.

---

These screenshots show the full flow: DNS validation, email reception, file storage, and concise logging. This makes the server suitable for testing, learning, and integration scenarios where you want to verify SMTP delivery and capture messages for inspection.
## Live Testing & Screenshots

See the [screenshots/README_screenshots.md](screenshots/README_screenshots.md) for a step-by-step visual guide to:
- Server startup and DNS verification
- Sending and receiving email
- Email file storage and log output

**Highlights:**
- DNS records are checked and shown at startup (see screenshot 1)
- Emails sent to the server are saved as timestamped files (see screenshots 2.1, 2.2)
- The server logs only a summary: sender and saved filename (see screenshot 2.3)

This demonstrates the full workflow and what you can expect when running the server in a real environment.

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
