# Go Chat

A simple chat server written in Go.

## Quick Start

To run it locally on port `3000`:
```bash
make run
nc localhost 3000
```

To run it in Docker:
```bash
docker-compose up --build chat
nc $(docker-compose port chat 3000 | tr ':' ' ')
```

When you are connected you can start sending commands, for example:
```text
join myname
say hi!
rename othername
say hi again!
```

## Features

* Response batching
* Prometheus metrics
* Graceful shutdown with notification
* Simple text protocol

## Protocol

The two main concepts are RPC and notifications.

The server accepts TCP connections, receives text commands line by line and responds with the result line.

It also sends notifications to the same TCP connection as lines of text.

Each command has a name and a list of arguments. Arguments are separated with one space character. 

Command layout:

```text
<COMMAND> <ARG1> <ARG2> ... <ARGN>\n
```

Command response layout:

```text
ok\n
error <reason>\n
```

Notification layout:

```text
<notification text>\n
```

Supported commands:

Command | Args  | Description
--------|-------|------------
join    | name  | Joins the chat room. Must be the first command to execute when connected. Name must be unique.
say     | words | Sends the message to all joined users. Accepts multiple words as arguments.
rename  | name  | Changes the current user's name. Name must be unique.

### Example session:

Lines are labeled:

`n` - notification
`r` - command response
`c` - command

```text
n Welcome!\n
c join john\n
n User john joined
r ok
c say hi!
n john: hi!
r ok
```

## Config

The server can be configured using environment variables:

Name      | Description
----------|------------
CHAT_PORT | TCP port to listen. Default: 3000

## Architecture

The server consists of two parts: transport and chat logic.

Transport is responsible for RPC, notifications delivery and connections management.

Chat logic handles command validation and execution.

The idea is to decouple chat logic from the infrastructure code for the sake if testability.

## Tests and Benchmarks

To run unit tests:

```bash
make test
```

To run benchmarks:

```bash
make bench
```
