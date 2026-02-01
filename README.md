# Email Server

A simple SMTP server implementation in Go using the `go-smtp` library.

## Configuration

### Port

This server is configured to run on **port 25** (default SMTP port) by default.

⚠️ **Important Notes:**
- Port 25 requires **root/administrator privileges** on Linux/Unix systems
- On Windows, you may need to run as Administrator if port 25 is already in use

### Running the Server

**On Linux/Unix (requires root):**
```bash
sudo go run main.go
```

**On Windows (run as Administrator):**
```
(Run Command Prompt/PowerShell as Administrator)
go run main.go
```

If you get a permission error:
```
listen tcp :25: permission denied (or access is denied)
```

This means:
- You don't have root/administrator privileges, OR
- Port 25 is already in use by another service

## Features

- Accepts incoming SMTP connections
- Logs sender and recipient information
- Captures email body content
- Implements the `go-smtp` backend interface

## DNS Configuration

For external mail delivery, configure:

### A Record
```
mail.yourdomain.com  A  1.2.3.4
```

### MX Record
```
yourdomain.com  MX  10  mail.yourdomain.com
```

### PTR Record (Reverse DNS)

PTR records map IP addresses back to hostnames. They're controlled by your **ISP or hosting provider** (not your domain registrar).

**Format for IPv4:**
```
[REVERSED_IP].in-addr.arpa  PTR  [hostname]
```

**Example for IP `1.2.3.4`:**
```
4.3.2.1.in-addr.arpa  PTR  mail.yourdomain.com
```

⚠️ **Note:** The IP octets are **reversed** (1.2.3.4 becomes 4.3.2.1)

**Step-by-step:**
1. Identify your IP: `1.2.3.4`
2. Reverse the octets: `4.3.2.1`
3. Add suffix: `4.3.2.1.in-addr.arpa`
4. Set PTR value to: `mail.yourdomain.com`
5. Contact your ISP to configure this record in their reverse DNS zone

**Testing your PTR record:**
```bash
nslookup -type=PTR 1.2.3.4
# or
dig -x 1.2.3.4
```

Should return: `mail.yourdomain.com`

**Why it matters:** Many mail servers check PTR records to verify sender legitimacy. Without proper PTR records, your emails are more likely to be marked as spam.
