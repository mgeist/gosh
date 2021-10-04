# gosh
A dead-simple hot reloader

## Motivation
I wanted a simple way to develop Golang programs in a Docker image. There are other apps that do the same thing, but I didn't want to rely on OS-specific tools. I just wanted some dead-simple polling. It's small, it's basic, and it does the job.

## Usage

Simple Dockerfile example
```dockerfile
FROM golang:1.17-alpine

RUN apk update && apk add git

WORKDIR /app

RUN ["go", "install", "github.com/mgeist/gosh@latest"]

COPY *.mod /app/
COPY *.go /app/

CMD ["gosh", "-dir", "/app", "-cmd", "go build -o out; ./out"]
```

## Config

`-cmd`: Command to execute when a change is detected. No default.
 - Note that this will kill the previous version of the process with `SIGTERM`

`-dir`: Directory to recursively watch. Default is `pwd`

`-glob`: Glob to match filenames against. Default is `*.go`. Does not currently support multiple globs.

`-ignore`: Comma-deliminated list of files and directories to ignore. Default is `-pollRate`: Time in milliseconds to between checks for file changes. Defaults to `100`

## Known Issues

Glob only supports one pattern at the moment. Will probably add support for more when I need it.

Gosh currently does not catch file deletions. Will probably fix this if it ever becomes an issue.
