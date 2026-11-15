package net

import (
	"fmt"

	mcnet "github.com/Tnze/go-mc/net"
	pk "github.com/Tnze/go-mc/net/packet"
)

/* Ping:
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