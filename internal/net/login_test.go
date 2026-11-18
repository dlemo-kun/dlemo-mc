package net

import "testing"

func TestLogin(t *testing.T) {
	host := "localhost"
	port := 25565

	res, err := Login(LoginOptions{
		Host:            host,
		Port:            port,
		ProtocolVersion: 777,
		Username:        "Player",
	})
	if err != nil {
		t.Fatalf("Error in Login: %v", err)
	}
	defer res.Conn.Close()

	t.Log(res.Conn)
	t.Log(res.UUID)
	t.Log(res.Username)
}