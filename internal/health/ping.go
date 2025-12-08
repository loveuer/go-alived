package health

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type PingChecker struct {
	name    string
	host    string
	count   int
	timeout time.Duration
}

func NewPingChecker(name string, config map[string]interface{}) (*PingChecker, error) {
	host, ok := config["host"].(string)
	if !ok {
		return nil, fmt.Errorf("ping checker: missing or invalid 'host' field")
	}

	count := 1
	if c, ok := config["count"]; ok {
		switch v := c.(type) {
		case int:
			count = v
		case float64:
			count = int(v)
		}
	}

	timeout := 2 * time.Second
	if t, ok := config["timeout"].(string); ok {
		if d, err := time.ParseDuration(t); err == nil {
			timeout = d
		}
	}

	return &PingChecker{
		name:    name,
		host:    host,
		count:   count,
		timeout: timeout,
	}, nil
}

func (c *PingChecker) Name() string {
	return c.name
}

func (c *PingChecker) Type() string {
	return "ping"
}

func (c *PingChecker) Check(ctx context.Context) CheckResult {
	addr, err := net.ResolveIPAddr("ip4", c.host)
	if err != nil {
		return CheckResultFailure
	}

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return CheckResultFailure
	}
	defer conn.Close()

	successCount := 0
	for i := 0; i < c.count; i++ {
		select {
		case <-ctx.Done():
			return CheckResultFailure
		default:
		}

		if c.sendPing(conn, addr) {
			successCount++
		}
	}

	if successCount > 0 {
		return CheckResultSuccess
	}

	return CheckResultFailure
}

func (c *PingChecker) sendPing(conn *icmp.PacketConn, addr *net.IPAddr) bool {
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   1234,
			Seq:  1,
			Data: []byte("go-alived-ping"),
		},
	}

	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return false
	}

	if _, err := conn.WriteTo(msgBytes, addr); err != nil {
		return false
	}

	conn.SetReadDeadline(time.Now().Add(c.timeout))

	reply := make([]byte, 1500)
	n, _, err := conn.ReadFrom(reply)
	if err != nil {
		return false
	}

	parsedMsg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), reply[:n])
	if err != nil {
		return false
	}

	if parsedMsg.Type == ipv4.ICMPTypeEchoReply {
		return true
	}

	return false
}
