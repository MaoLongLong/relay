# Relay

Minimal TCP relay (proxy).

```text
+--------+         +--------+         +--------+
| Client |-------->| Relay  |-------->| Server |
+--------+         +--------+         +--------+
```

Install and usage:

```bash
go install go.chensl.me/relay@latest

# relay [-debug] <from> <to>
relay 127.0.0.1:8081 127.0.0.1:8080
```
