# Call2Action

An event-driven command execution framework written in Go. It reads classification messages from a Redis queue, looks up the configured command templates for each `classifier_name`, and dispatches those commands to [Poppit](https://github.com/its-the-vibe/Poppit) via a Redis list.

---

## How It Works

```
Redis queue  ──BLPOP──▶  Call2Action  ──RPUSH──▶  Poppit list  ──▶  Commands executed
```

1. Call2Action pops JSON messages from a configurable Redis list (the incoming queue).
2. It reads the `classifier_name` field and looks up the matching command templates in `config.yaml`.
3. Placeholder values (e.g. `{original_path}`, `{new_path}`) are substituted with fields from the message.
4. A [Poppit](https://github.com/its-the-vibe/Poppit) payload is pushed to the configured Redis list for execution.

### Incoming message format

```json
{
  "file_info": {
    "id": "F0123456789",
    "name": "report.pdf",
    "title": "Q4 Report",
    "mimetype": "application/pdf",
    "size": 204800
  },
  "original_path": "/downloads/general/report.pdf",
  "new_path": "/classified/documents/report.pdf",
  "classifier_name": "documents",
  "classified_at": "2025-01-15T12:34:56Z"
}
```

### Poppit payload published to Redis

```json
{
  "repo": "its-the-vibe/Call2Action",
  "branch": "refs/heads/main",
  "type": "call2action",
  "dir": "/tmp",
  "commands": [
    "process-doc --input /downloads/general/report.pdf --output /classified/documents/report.pdf"
  ]
}
```

---

## Prerequisites

- Go 1.24 or later
- A running Redis server (external – not bundled)
- [Poppit](https://github.com/its-the-vibe/Poppit) monitoring the same Redis instance

---

## Setup

### 1. Clone the repository

```bash
git clone https://github.com/its-the-vibe/Call2Action.git
cd Call2Action
```

### 2. Create the configuration file

```bash
cp config.example.yaml config.yaml
```

Edit `config.yaml` to match your Redis address, queue names, and classifier commands. See [Configuration](#configuration) for details.

### 3. Create the environment file

```bash
cp .env.example .env
```

Set `REDIS_PASSWORD` to your Redis password (leave blank if Redis requires no authentication).

### 4. Build and run

```bash
go build -o call2action ./cmd/call2action
./call2action
```

The binary reads `config.yaml` from the current directory by default. Override the path with the `CONFIG_PATH` environment variable:

```bash
CONFIG_PATH=/etc/call2action/config.yaml ./call2action
```

---

## Configuration

All settings live in `config.yaml` (gitignored). Use `config.example.yaml` as a starting template.

| Key | Description |
|-----|-------------|
| `redis.addr` | Redis server address (`host:port`) |
| `redis.db` | Redis database index (default `0`) |
| `queue.name` | Redis list to consume incoming messages from |
| `poppit.list` | Redis list Poppit is monitoring |
| `poppit.repo` | GitHub repository included in Poppit payloads |
| `poppit.branch` | Git branch included in Poppit payloads |
| `poppit.type` | Event type included in Poppit payloads |
| `poppit.dir` | Working directory for executed commands |
| `classifiers.<name>.commands` | List of shell command templates for the classifier |

### Sensitive values (`.env`)

| Variable | Description |
|----------|-------------|
| `REDIS_PASSWORD` | Redis authentication password |
| `CONFIG_PATH` | Path to `config.yaml` (default: `config.yaml`) |

### Command template placeholders

The following placeholders are replaced with values from the incoming message:

| Placeholder | Source field |
|-------------|-------------|
| `{file_id}` | `file_info.id` |
| `{file_name}` | `file_info.name` |
| `{file_title}` | `file_info.title` |
| `{file_mimetype}` | `file_info.mimetype` |
| `{file_size}` | `file_info.size` |
| `{original_path}` | `original_path` |
| `{new_path}` | `new_path` |
| `{classifier_name}` | `classifier_name` |
| `{classified_at}` | `classified_at` |

---

## Running with Docker Compose

```bash
# Build the image
docker compose build

# Start the service
docker compose up -d

# View logs
docker compose logs -f
```

The service container runs as **read-only**. The Redis server is external – configure its address in `config.yaml` and its password in `.env`.

---

## Testing a message

Push a test message directly to Redis using `redis-cli`:

```bash
redis-cli RPUSH call2action:queue '{
  "file_info": {
    "id": "F0123456789",
    "name": "report.pdf",
    "title": "Q4 Report",
    "mimetype": "application/pdf",
    "size": 204800
  },
  "original_path": "/downloads/general/report.pdf",
  "new_path": "/classified/documents/report.pdf",
  "classifier_name": "documents",
  "classified_at": "2025-01-15T12:34:56Z"
}'
```

Call2Action will pop the message, look up the `documents` classifier in `config.yaml`, substitute the placeholders, and push the resulting Poppit payload to the configured Poppit Redis list.

---

## Running tests

```bash
go test ./...
```
