package health

import (
	"context"
	"fmt"
	"net"
)

type TCPChecker struct {
	name string
	host string
	port int
}

func NewTCPChecker(name string, config map[string]interface{}) (*TCPChecker, error) {
	host, ok := config["host"].(string)
	if !ok {
		return nil, fmt.Errorf("tcp checker: missing or invalid 'host' field")
	}

	var port int
	switch v := config["port"].(type) {
	case int:
		port = v
	case float64:
		port = int(v)
	default:
		return nil, fmt.Errorf("tcp checker: missing or invalid 'port' field")
	}

	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("tcp checker: invalid port number: %d", port)
	}

	return &TCPChecker{
		name: name,
		host: host,
		port: port,
	}, nil
}

func (c *TCPChecker) Name() string {
	return c.name
}

func (c *TCPChecker) Type() string {
	return "tcp"
}

func (c *TCPChecker) Check(ctx context.Context) CheckResult {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return CheckResultFailure
	}

	conn.Close()
	return CheckResultSuccess
}