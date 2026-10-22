package conversation

import "fmt"

func normalizeChannel(raw string) (string, error) {
	switch raw {
	case "", "create", "execute", "deploy":
		return raw, nil
	default:
		return "", fmt.Errorf("未知通道")
	}
}
