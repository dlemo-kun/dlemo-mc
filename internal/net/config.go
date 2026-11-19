package net

import (
	"fmt"

	mcnet "github.com/Tnze/go-mc/net"
	pk "github.com/Tnze/go-mc/net/packet"
)

// ConfigureOptions holds the client settings sent during the Configuration state.
type ConfigureOptions struct {
	Locale       string // Client locale (e.g. "en_us")
	ViewDistance int8   // Render distance in chunks (e.g. 8)
}

// Configure performs the Minecraft Configuration state (protocol 777 / 26.3)
// on an authenticated connection and transitions it into the Play state.
//
// Example:
//
//	err := Configure(res.Conn, ConfigureOptions{
//		Locale:       "en_us",
//		ViewDistance: 8,
//	})
//	if err != nil {
//		return err
//	}
func Configure(conn *mcnet.Conn, opts ConfigureOptions) error {
	if opts.Locale == "" {
		opts.Locale = "en_us"
	}
	if opts.ViewDistance <= 0 {
		opts.ViewDistance = 8
	}

	err := conn.WritePacket(pk.Marshal(
		0x00, // Client Information
		pk.String(opts.Locale),
		pk.Byte(opts.ViewDistance),
		pk.VarInt(0),          // Chat mode: enabled
		pk.Boolean(true),      // Chat colors
		pk.UnsignedByte(0x7F), // Displayed skin parts: all
		pk.VarInt(1),          // Main hand: right
		pk.Boolean(false),     // Enable text filtering
		pk.Boolean(true),      // Allow server listings
		pk.VarInt(0),          // Particle status: all
	))
	if err != nil {
		return fmt.Errorf("send client information: %w", err)
	}

	var p pk.Packet
	for {
		if err := conn.ReadPacket(&p); err != nil {
			return fmt.Errorf("read configuration packet: %w", err)
		}

		switch p.ID {
		case 0x02: // Disconnect (Configuration)
			return fmt.Errorf("disconnected by server during configuration (payload: %x)", p.Data)

		case 0x03: // Finish Configuration
			if err := conn.WritePacket(pk.Marshal(0x03)); err != nil {
				return fmt.Errorf("send finish configuration ack: %w", err)
			}
			return nil

		case 0x04: // Keep Alive (Configuration)
			var keepAliveID pk.Long
			if err := p.Scan(&keepAliveID); err != nil {
				return fmt.Errorf("parse keep alive: %w", err)
			}
			if err := conn.WritePacket(pk.Marshal(0x04, keepAliveID)); err != nil {
				return fmt.Errorf("send keep alive: %w", err)
			}

		case 0x05: // Ping (Configuration)
			var pingID pk.Int
			if err := p.Scan(&pingID); err != nil {
				return fmt.Errorf("parse ping: %w", err)
			}
			if err := conn.WritePacket(pk.Marshal(0x05, pingID)); err != nil {
				return fmt.Errorf("send pong: %w", err)
			}

		case 0x09: // Resource Pack Push
			var packUUID pk.UUID
			if err := p.Scan(&packUUID); err != nil {
				return fmt.Errorf("parse resource pack push: %w", err)
			}
			if err := conn.WritePacket(pk.Marshal(0x06, packUUID, pk.VarInt(3))); err != nil { // Accepted
				return fmt.Errorf("send resource pack accepted: %w", err)
			}
			if err := conn.WritePacket(pk.Marshal(0x06, packUUID, pk.VarInt(0))); err != nil { // Successfully loaded
				return fmt.Errorf("send resource pack loaded: %w", err)
			}

		case 0x0F: // Clientbound Known Packs
			// Echo server packs back so the server skips sending full registry payloads.
			if err := conn.WritePacket(pk.Packet{ID: 0x07, Data: p.Data}); err != nil {
				return fmt.Errorf("send known packs: %w", err)
			}

		default:
			// Ignore optional/world-data packets (e.g. Registry Data, Update Tags) until world storage is implemented.
		}
	}
}