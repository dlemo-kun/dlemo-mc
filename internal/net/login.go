package net

import (
	"errors"
	"fmt"

	mcnet "github.com/Tnze/go-mc/net"
	pk "github.com/Tnze/go-mc/net/packet"
	"github.com/Tnze/go-mc/offline"
	"github.com/google/uuid"
)

// LoginOptions holds the connection parameters required to log in to a Minecraft server.
type LoginOptions struct {
	Host            string // Server hostname or IP (e.g. "localhost")
	Port            int    // Server port (e.g. 25565)
	ProtocolVersion int32  // Minecraft protocol version (e.g. 777 for 26.3)
	Username        string // Player username (e.g. "Player")
}

// LoginResult holds the active server connection and the player identity
// confirmed by the server after a successful login.
type LoginResult struct {
	Conn     *mcnet.Conn // Active connection ready for the Configuration/Play state
	UUID     uuid.UUID   // Confirmed player UUID (e.g. a01e3843-e521-3998-958a-f459800e4d11)
	Username string      // Confirmed player username (e.g. "Player")
}

// Login performs the Handshake and Login phase (offline mode only) against a
// Minecraft server and returns an active connection ready for the Configuration state.
//
// Example:
//
//	res, err := Login(LoginOptions{
//		Host:            "localhost",
//		Port:            25565,
//		ProtocolVersion: 777,
//		Username:        "Player",
//	})
//	if err != nil {
//		return err
//	}
//	defer res.Conn.Close()
//	// res.UUID: a01e3843-e521-3998-958a-f459800e4d11
//	// res.Username: "Player"
func Login(opts LoginOptions) (*LoginResult, error) {
	addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
	conn, err := mcnet.DialMC(addr)
	if err != nil {
		return nil, fmt.Errorf("dial server: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_ = conn.Close()
		}
	}()

	err = conn.WritePacket(pk.Marshal(
		0x00,
		pk.VarInt(opts.ProtocolVersion),
		pk.String(opts.Host),
		pk.UnsignedShort(opts.Port),
		pk.VarInt(2), // Login
	))
	if err != nil {
		return nil, fmt.Errorf("send handshake: %w", err)
	}

	playerUUID := offline.NameToUUID(opts.Username)

	err = conn.WritePacket(pk.Marshal(
		0x00,
		pk.String(opts.Username),
		pk.UUID(playerUUID),
	))
	if err != nil {
		return nil, fmt.Errorf("send login start: %w", err)
	}

	var p pk.Packet
	for {
		if err := conn.ReadPacket(&p); err != nil {
			return nil, fmt.Errorf("read login packet: %w", err)
		}

		switch p.ID {
		case 0x00: // Disconnect (Login)
			var reason pk.String
			if err := p.Scan(&reason); err != nil {
				return nil, fmt.Errorf("disconnected (failed to parse reason): %w", err)
			}
			return nil, fmt.Errorf("disconnected by server: %s", string(reason))

		case 0x01: // Encryption Request (Online Mode)
			return nil, errors.New("server requires online-mode authentication (encryption not implemented yet)")

		case 0x02: // Login Success
			var confirmedUUID pk.UUID
			var confirmedName pk.String
			if err := p.Scan(&confirmedUUID, &confirmedName); err != nil {
				return nil, fmt.Errorf("parse login success: %w", err)
			}

			if err := conn.WritePacket(pk.Marshal(0x03)); err != nil {
				return nil, fmt.Errorf("send login acknowledged: %w", err)
			}

			success = true
			return &LoginResult{
				Conn:     conn,
				UUID:     uuid.UUID(confirmedUUID),
				Username: string(confirmedName),
			}, nil

		case 0x03: // Set Compression
			var threshold pk.VarInt
			if err := p.Scan(&threshold); err != nil {
				return nil, fmt.Errorf("parse set compression: %w", err)
			}
			conn.SetThreshold(int(threshold))

		case 0x04: // Login Plugin Request (used by proxies like Velocity to exchange custom data during login)
			var messageID pk.VarInt
			if err := p.Scan(&messageID); err != nil {
				return nil, fmt.Errorf("parse plugin request: %w", err)
			}
			err := conn.WritePacket(pk.Marshal(
				0x02,
				messageID,
				pk.Boolean(false),
			))
			if err != nil {
				return nil, fmt.Errorf("send plugin response: %w", err)
			}

		default:
			return nil, fmt.Errorf("unexpected packet ID in login state: 0x%02X", p.ID)
		}
	}
}
