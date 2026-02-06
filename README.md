# Email Server

A simple SMTP server in Go using the `go-smtp` library.

## Features
- Accepts SMTP connections on port 25
- Logs sender, recipient, and email body to console
- Prints required DNS records for mail delivery
- Stores emails to disk (file storage)
- Stores emails to PostgreSQL database with attachment tracking
- Dual-write capability (file + database simultaneously)
- Does NOT forward emails (for testing/learning only)

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
| MAIL_SERVERS  | No       | (Optional) List of FQDN,IP pairs separated by `:` (see example above). If not set the program will print `Email server is running` and expose a simple HTTP health endpoint at `/`.       |
| HTTP_PORT     | No       | (Optional) HTTP health port. Defaults to `48080` if not set.      |
| DB_URL        | No       | (Optional) PostgreSQL connection string. If provided, emails are saved to both file and database. Format: `user=username password=pass dbname=emaildb host=localhost port=5432 sslmode=disable`       |

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

## Live Testing & Screenshots

## HTTP Health Endpoint

- The server exposes a simple HTTP API on `HTTP_PORT` (default `48080`).
- `GET /` returns a plain-text status. When `MAIL_SERVERS` is not set it returns `Email server is running`.


Below are real-world screenshots and explanations of the server in action:

### 1. Setup & Receiving Email

![Server Startup and DNS Verification](screenshots/1.png)

- The server is started with the `MAIL_SERVERS` environment variable.
- It prints a table for each FQDN and IP pair, showing the status of A, MX, and PTR records.
- If any record is missing or incorrect, the server will not start and will print a clear error.

### 2.1. Sending an Email from Gmail

![Sending from Gmail](screenshots/2.2.png)

- An email is sent from Gmail to the server's configured address.

### 2.2. Email Received and Saved

![Email Saved to File](screenshots/2.1.png)
![Email content](screenshots/2.3.png)

- The server receives the email and saves it to a file in the `emails/` directory.
- The filename is a timestamp in nanoseconds, ensuring uniqueness.
- The server logs a summary: `from: <sender>, saved in <filename>`
- No email content is printed to the console for privacy and clarity.

## Docker Deployment

### Build and Run with Docker Compose

The easiest way to deploy is using Docker Compose, which includes PostgreSQL:

```bash
# Clone or download the repository
cd email-server

# Build and start the services
docker-compose up -d

# View logs
docker-compose logs -f email-server

# Stop the services
docker-compose down
```

### Environment Setup with MAIL_SERVERS

Create a `.env` file or set environment variables:

```bash
# For SMTP with DNS validation
export MAIL_SERVERS="mail.example.com,1.2.3.4"
docker-compose up -d

# For SMTP without MAIL_SERVERS (accepts all)
docker-compose up -d
```

### Manual Docker Build

```bash
docker build -t email-server .
docker run -p 25:25 -p 48080:48080 \
  -e DB_URL="user=emailuser password=emailpass dbname=emaildb host=postgres port=5432 sslmode=disable" \
  -e MAIL_SERVERS="mail.example.com,1.2.3.4" \
  email-server
```

## Database Schema

When `DB_URL` is provided, the server automatically creates two tables:

### email table
Stores email metadata and content:
- `id` (SERIAL PRIMARY KEY) — Unique email ID
- `from` (TEXT) — Sender email address
- `to` (TEXT) — Recipient email address  
- `subject` (TEXT) — Email subject
- `date` (TEXT) — Email send date
- `body` (TEXT) — Parsed email body
- `raw_content` (TEXT) — Full raw email content
- `created_at` (TIMESTAMP) — Record creation time

### attachment table
Stores email attachments:
- `id` (SERIAL PRIMARY KEY) — Unique attachment ID
- `email_id` (INTEGER FK) — Foreign key to email table
- `filename` (TEXT) — Original filename
- `content_type` (TEXT) — MIME type (e.g., image/png)
- `data` (BYTEA) — Binary attachment data
- `created_at` (TIMESTAMP) — Record creation time

## Storage Options

### File Storage (Default)
Emails are saved to `emails/<to>/<from>/timestamp.txt`. This is always enabled.

### PostgreSQL Storage (Optional)
When `DB_URL` is set, emails are saved to PostgreSQL in addition to files.

### Composite Storage
If `DB_URL` is provided, both file and database storage are used simultaneously. If PostgreSQL connection fails, the server falls back to file-only storage with a warning.