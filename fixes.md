# Machine Simulator OPC-UA Connection Fixes

## Overview

This document describes the fixes applied to resolve OPC-UA connection issues between the machine simulator and UMH-core Benthos protocol converters.

## Problem Summary

OPC-UA connections were failing with the following symptoms:
- Benthos logs showed "connection reset by peer" errors
- TCP accept queue (Recv-Q) grew unbounded (300+ pending connections)
- 0 active OPC-UA connections despite multiple client connection attempts
- Server appeared healthy but all connection attempts failed

## Root Causes Identified

### Issue 1: Endpoint URL Mismatch

**Problem**: The simulator was advertising OPC-UA endpoints using the container's internal IP address (e.g., `opc.tcp://172.19.0.2:4840`) instead of the Docker service hostname that clients use to connect (e.g., `opc.tcp://machine-simulator:4840`).

This caused endpoint URL mismatches during the OPC-UA GetEndpoints discovery phase.

**Files affected**:
- `internal/config/config.go`
- `internal/connectivity/opcua/gopcua_server.go`
- `cmd/simulator/main.go`

### Issue 2: Blocking Accept Loop (gopcua library)

**Problem**: The gopcua library's `uacp.Listener.Accept()` function performs the OPC-UA handshake (Hello/Ack exchange) synchronously within the accept loop. Non-OPC-UA connections (like nmap TCP port scans) complete the TCP handshake but never send the OPC-UA Hello message, causing the accept loop to block.

**Root cause**: UMH-core runs nmap health check scripts that continuously probe OPC-UA ports every 1 second for each protocol converter. These scans complete TCP connections but don't speak OPC-UA protocol, blocking the accept loop.

**Files affected**:
- `vendor/github.com/gopcua/opcua/uacp/conn.go`
- `vendor/github.com/gopcua/opcua/server/server.go`

## Fixes Applied

### Fix 1: OPCUA_HOSTNAME Environment Variable (v1.3.0)

Added support for configuring the hostname advertised in OPC-UA endpoint URLs.

**config.go changes**:
```go
type OPCUAConfig struct {
    // ... existing fields ...
    // Hostname is the hostname to advertise in OPC-UA endpoint URLs
    // Set this to the Docker service name (e.g., "machine-simulator") when running in Docker
    Hostname string `yaml:"hostname" env:"OPCUA_HOSTNAME" envDefault:""`
}
```

**gopcua_server.go changes**:
```go
type ServerConfig struct {
    // ... existing fields ...
    Hostname string
}

func (s *GoPCUAServer) GetEndpoint() string {
    host := s.hostname
    if host == "" {
        host = getLocalIP()  // Fallback to IP detection
    }
    return fmt.Sprintf("opc.tcp://%s:%d", host, s.port)
}
```

**docker-compose.yaml**:
```yaml
environment:
  - OPCUA_HOSTNAME=machine-simulator
```

### Fix 2: Handshake Timeout (v1.4.0)

Added a 10-second timeout to the OPC-UA handshake to prevent indefinite blocking.

**vendor/github.com/gopcua/opcua/uacp/conn.go**:
```go
type Listener struct {
    l                *net.TCPListener
    ack              *Acknowledge
    endpoint         string
    HandshakeTimeout time.Duration // Timeout for OPC-UA handshake (default: 10s)
}

func (l *Listener) Accept(ctx context.Context) (*Conn, error) {
    c, err := l.l.AcceptTCP()
    // ...

    // Set handshake timeout to prevent blocking
    timeout := l.HandshakeTimeout
    if timeout == 0 {
        timeout = 10 * time.Second
    }
    c.SetReadDeadline(time.Now().Add(timeout))

    if err := conn.srvhandshake(l.endpoint); err != nil {
        c.Close()
        return nil, err
    }

    // Clear deadline for normal operation
    c.SetReadDeadline(time.Time{})
    return conn, nil
}
```

### Fix 3: Async Accept Pattern (v1.5.0)

The timeout alone wasn't sufficient because with ~9 nmap scans per second and a 10-second timeout, the serial accept loop could only process one connection every 10 seconds, causing the backlog to grow faster than it could be cleared.

The fix decouples TCP accept from OPC-UA handshake:

**vendor/github.com/gopcua/opcua/uacp/conn.go** - Added new methods:
```go
// AcceptTCP accepts TCP without handshake (non-blocking)
func (l *Listener) AcceptTCP() (*Conn, error) {
    c, err := l.l.AcceptTCP()
    if err != nil {
        return nil, err
    }
    return &Conn{TCPConn: c, id: nextid(), ack: l.ack}, nil
}

// DoHandshake performs handshake separately (call in goroutine)
func (l *Listener) DoHandshake(conn *Conn) (*Conn, error) {
    timeout := l.HandshakeTimeout
    if timeout == 0 {
        timeout = 10 * time.Second
    }
    conn.SetReadDeadline(time.Now().Add(timeout))

    if err := conn.srvhandshake(l.endpoint); err != nil {
        conn.Close()
        return nil, err
    }

    conn.SetReadDeadline(time.Time{})
    return conn, nil
}
```

**vendor/github.com/gopcua/opcua/server/server.go** - Modified accept loop:
```go
func (s *Server) acceptAndRegister(ctx context.Context, l *uacp.Listener) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            // Accept TCP without blocking on handshake
            c, err := l.AcceptTCP()
            if err != nil {
                // ... error handling ...
            }

            // Handshake asynchronously
            go func(conn *uacp.Conn) {
                c, err := l.DoHandshake(conn)
                if err != nil {
                    return  // Non-OPC-UA connection, discard
                }
                s.cb.RegisterConn(ctx, c, s.cfg.certificate, s.cfg.privateKey)
            }(c)
        }
    }
}
```

## Results

### Before fixes:
- Recv-Q: 300+ pending connections
- Active OPC-UA connections: 0
- Client errors: "connection reset by peer"

### After fixes:
- Recv-Q: 0 (accept loop keeps up with incoming connections)
- Active OPC-UA connections: Working
- Data flowing through Benthos protocol converters

## Version History

| Version | Changes |
|---------|---------|
| v1.3.0  | Added OPCUA_HOSTNAME environment variable support |
| v1.4.0  | Added handshake timeout to gopcua vendor library |
| v1.5.0  | Added async accept pattern for parallel handshake processing |

## Configuration

To enable proper hostname resolution in Docker environments, set the following environment variable:

```yaml
environment:
  - OPCUA_HOSTNAME=machine-simulator  # Use Docker service name
```

## Upstream Contribution Notes

The gopcua library patches should be contributed upstream:
- Repository: https://github.com/gopcua/opcua
- Files: `uacp/conn.go`, `server/server.go`
- Issue: Server Accept() blocks indefinitely on non-OPC-UA connections

See `vendor/github.com/gopcua/opcua/uacp/PATCH.md` for detailed patch documentation.

## Diagnostic Commands

Check accept queue status:
```bash
docker compose exec machine-simulator netstat -tln | grep 484
```

A healthy server shows Recv-Q near 0:
```
tcp    0    0 172.19.0.2:4840    0.0.0.0:*    LISTEN
```

Check connection metrics in simulator logs:
```bash
docker compose logs machine-simulator | grep "active_connections"
```
