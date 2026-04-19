package cmd

import (
	"strings"
	"testing"
)

func TestGenerateDockerCompose(t *testing.T) {
	compose := generateDockerCompose()

	for _, expected := range []string{
		"image: " + defaultDockerImage,
		"privileged: true",
		"network_mode: host",
		"./config.yaml:/etc/go-alived/config.yaml:ro",
		"./scripts:/etc/go-alived/scripts:ro",
		"- /etc/go-alived/config.yaml",
	} {
		if !strings.Contains(compose, expected) {
			t.Fatalf("generated compose file is missing %q", expected)
		}
	}
}
