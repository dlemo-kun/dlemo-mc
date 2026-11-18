package client

import (
	"fmt"
	"log"

	mcnet "github.com/dlemo-kun/dlemo-mc/internal/net"
)

func Connect(host string, port int, username string) error {
	res, err := mcnet.Login(mcnet.LoginOptions{
		Host:            host,
		Port:            port,
		ProtocolVersion: 777, // 26.3
		Username:        username,
	})
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	defer res.Conn.Close()

	log.Printf("Login exitoso! Usuario: %s, UUID: %s", res.Username, res.UUID)

	return nil
}