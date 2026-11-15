package net

import (
	"errors"
	"fmt"

	mcnet "github.com/Tnze/go-mc/net"
	pk "github.com/Tnze/go-mc/net/packet"
	"github.com/Tnze/go-mc/offline"
	"github.com/google/uuid"
)

/*
LoginOptions:

EN: Contains the parameters needed to connect to a server.

	Host            string // localhost
	Port            int    // 25565
	ProtocolVersion int32  // 777 (26.3)
	Username        string // Player

ES: Contiene los parámetros necesarios para conectarse a un servidor.

	Host            string // localhost
	Port            int    // 25565
	ProtocolVersion int32  // 777 (26.3)
	Username        string // Jugador
*/
type LoginOptions struct {
	Host            string
	Port            int
	ProtocolVersion int32
	Username        string
}

/*
LoginResult:

EN: Returns the active connection and the data confirmed by the server.

	Conn     *mcnet.Conn // &{0x6e7e3682018 0x6e7e3682018 0x6e7e3682018 256}
	UUID     uuid.UUID   // a01e3843-e521-3998-958a-f459800e4d11
	Username string      // Player

ES: Devuelve la conexión activa y los datos confirmados por el servidor.

	Conn     *mcnet.Conn // &{0x3b7365d82018 0x3b7365d82018 0x3b7365d82018 256}
	UUID     uuid.UUID   // e3b74ba8-f059-3c5e-92c9-e84f10defbf6
	Username string      // Jugador
*/
type LoginResult struct {
	Conn     *mcnet.Conn
	UUID     uuid.UUID
	Username string
}

/*
Login:

EN: It performs the Handshake and Login phase (only in offline mode) against a Minecraft server.

	Input:
		opts LoginOptions // opts.Host            localhost
						  // opts.Port            25565
						  // opts.ProtocolVersion 777
						  // opts.Username        Player

	Output:
		*LoginResult // res.Conn     &{0x6e7e3682018 0x6e7e3682018 0x6e7e3682018 256}
		             // res.UUID     a01e3843-e521-3998-958a-f459800e4d11
		             // res.Username Player
		error        // <nil>

ES: Realiza el Handshake y la fase de Login (only en modo offline) contra un servidor Minecraft.

	Input:
		opts LoginOptions // opts.Host            localhost
						  // opts.Port            25565
						  // opts.ProtocolVersion 777
						  // opts.Username        Jugador

	Output:
		*LoginResult // res.Conn     &{0x3b7365d82018 0x3b7365d82018 0x3b7365d82018 256}
		             // res.UUID     e3b74ba8-f059-3c5e-92c9-e84f10defbf6
		             // res.Username Jugador
		error        // <nil>
*/
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

		case 0x04: // Login Plugin Request (Used by the Velocity proxy (I don't know what it's about))
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
