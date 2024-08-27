## ncdmv

Monitor NCDMV for new appointment slots and get notified through Discord.

<img src="https://i.imgur.com/pW9Vxio.png" alt="Discord message example" width="75%"/>

### Usage

```
$ go run ./cmd/ncdmv -h
Usage of ncdmv:
  -appt_type string
        appointment type (options: permit,driver-license,non-cdl-road-test,driver-license-duplicate,driver-license-renewal,id-card,knowledge-test,motorcycle-skills-test) (default "permit")
  -database_path string
        path to database file (default "./database/ncdmv.db")
  -debug
        enable debug mode
  -disable_gpu
        disable GPU
  -discord_webhook string
        Discord webhook URL for notifications (optional)
  -headless
        enable headless browser (default true)
  -interval int
        interval between checks, in minutes (default 30)
  -locations string
        comma-seperated list of locations to check (options: durham-south,hillsborough,raleigh-east,raleigh-north,ahoskie,durham-east,fuquay-varina,garner,raleigh-west,cary,clayton,wendell) (default "cary,durham-east,durham-south")
  -migrations_path string
        path to migrations directory
  -notify_unavailable
        if true, notify when a previously available appointment becomes unavailable (default true)
  -stop_on_failure
        if true, stop completely on a failure instead of just logging
  -timeout int
        timeout, in seconds (default 120)
```

### Examples

Run in headless mode:

```
go run ./cmd/ncdmv -locations=cary,durham-east,durham-south -discord_webhook=[WEBHOOK]
```

Show the browser with a timeout of 2 minutes each check (across all locations) and an interval of 10 minutes:

```
go run ./cmd/ncdmv -headless=false -locations=cary,durham-east,durham-south -discord_webhook=[WEBHOOK] -timeout=120 -interval=10
```

Run on Docker:

```
docker run --rm ghcr.io/aksiksi/ncdmv:latest -h
```

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
