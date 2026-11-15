package net

import (
	"fmt"

	mcnet "github.com/Tnze/go-mc/net"
	pk "github.com/Tnze/go-mc/net/packet"
)

/*
Ping:

EN: Check the connection to the server and return a JSON.

	Input:
		host string // localhost
		port int    // 25565

	Output:
		string // {"description":"nw111026","players":{"max":20,"online":0},"version":{"name":"26.3","protocol":777}}
		error  // <nil>

ES: Compruebe la conexión al servidor y devuelva un JSON.

	Input:
		host string // localhost
		port int    // 25565

	Output:
		string // {"description":"nm101126","players":{"max":20,"online":0},"version":{"name":"26.3","protocol":777}}
		error  // <nil>
*/
func Ping(host string, port int) (string, error) {
	conn, err := mcnet.DialMC(fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	err = conn.WritePacket(pk.Marshal(
		0x00,
		pk.VarInt(-1),
		pk.String(host),
		pk.UnsignedShort(port),
		pk.VarInt(1),
	))
	if err != nil {
		return "", err
	}

	if err := conn.WritePacket(pk.Marshal(0x00)); err != nil {
		return "", err
	}

	var p pk.Packet
	if err := conn.ReadPacket(&p); err != nil {
		return "", err
	}
	var status pk.String
	if err := p.Scan(&status); err != nil {
		return "", err
	}
	return string(status), nil
}
