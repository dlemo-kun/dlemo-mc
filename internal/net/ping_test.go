package net

import "testing"

func TestPing(t *testing.T) {
	host := "localhost"
	port := 25565

	status, err := Ping(host, port)
	if err != nil {
		t.Fatalf("Error in Ping: %v", err)
	}
	t.Logf("Return Ping: %s", status)
}