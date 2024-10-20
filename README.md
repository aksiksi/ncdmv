# ncdmv

Monitor NCDMV for new appointment slots and get notified through Discord.

<img src="https://i.imgur.com/pW9Vxio.png" alt="Discord message example" width="75%"/>

## Usage

```
ncdmv monitors NC DMV appointments

Usage:
  ncdmv [flags]

Flags:
  -t, --appt-type string         appointment type (one of: [knowledge-test motorcycle-skills-test non-cdl-road-test permit driver-license driver-license-duplicate driver-license-renewal id-card]) (default "permit")
  -d, --database-path string     database path
      --debug                    enable debug mode
      --debug-chrome             enable debug mode for Chrome
      --disable-gpu              disable GPU acceleration
  -w, --discord-webhook string   Discord webhook URL
      --headless                 run Chrome in headless mode (no GUI) (default true)
  -h, --help                     help for ncdmv
      --interval duration        interval between searches (default 5m0s)
  -l, --locations strings        locations to search (default [cary,durham-east,durham-south])
      --notify-unavailable       if set, send a notification if an appointment becomes unavailable (default true)
      --stop-on-failure          if set, completely stop on failure instead of just logging
      --timeout duration         timeout for each search, in seconds (default 5m0s)
```

## Examples

Run in headless mode:

```
go run ./cmd/ncdmv -l cary,durham-east,durham-south -w [WEBHOOK] --database-path ./ncdmv.db
```

Show the browser with a timeout of 5 minutes each check (across all locations) and an interval of 10 minutes:

```
go run ./cmd/ncdmv -l cary,durham-east,durham-south -w [WEBHOOK] --database-path ./ncdmv.db --timeout 5m --interval 10m --headless=false 
```

## Docker

Note: you can only run headless Chrome with Docker.

```
docker run --rm -v $(pwd):/config -e NCDMV_APPT_TYPE=permit -e NCDMV_LOCATIONS=cary,durham-east ghcr.io/aksiksi/ncdmv:latest
```

### Docker Compose

```yaml
services:
  ncdmv:
    image: ghcr.io/aksiksi/ncdmv:latest
    volumes:
      - /var/volumes/ncdmv:/config
    environment:
      NCDMV_APPT_TYPE: permit
      NCDMV_LOCATIONS: cary,durham-east
      NCDMV_DISCORD_WEBHOOK: "https://..." # optional
      NCDMV_TIMEOUT: 5m # optional
      NCDMV_INTERVAL: 5m # optional
      NCDMV_NOTIFY_UNAVAILABLE: true # optional
      NCDMV_DISABLE_GPU: false # optional
```

## Appendix

### If you are new to Go

1. Install: https://go.dev/doc/install
2. ```go mod tidy```
3. ```go run ./cmd/ncdmv-migrate```
4. Run!

### Setup (Debian)

1. Install Google Chrome: https://www.if-not-true-then-false.com/2021/install-google-chrome-on-debian/. Quick summary:

```
sudo su -

cat  /etc/apt/sources.list.d/google-chrome.list
deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main
EOF

wget -O- https://dl.google.com/linux/linux_signing_key.pub |gpg --dearmor > /etc/apt/trusted.gpg.d/google.gpg

apt update

apt install google-chrome-stable
```

2. Run!

### Setup (Ubuntu)
1. Install Google Chrome: 

```
wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb

sudo dpkg -i google-chrome-stable_current_amd64.deb

sudo apt --fix-broken install
```

2. Install sqlite
```
sudo apt install sqlite3
```
