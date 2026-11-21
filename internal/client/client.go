package client

import (
	"fmt"
	"log"
	"sync"
	"time"

	gmcnet "github.com/Tnze/go-mc/net"
	pk "github.com/Tnze/go-mc/net/packet"
	mcnet "github.com/dlemo-kun/dlemo-mc/internal/net"
)

type safeConn struct {
	*gmcnet.Conn
	mu sync.Mutex
}

func (c *safeConn) writePacket(p pk.Packet) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.WritePacket(p)
}

// Connect logs in to a Minecraft server, completes the Configuration state,
// and runs the Play packet loop until disconnected.
func Connect(host string, port int, username string) error {
	pj, err := mcnet.Ping(host, port)

	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	log.Printf("Ping succeeded: JSON=%s", pj)

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

	log.Printf("Login succeeded: username=%s uuid=%s", res.Username, res.UUID)

	err = mcnet.Configure(res.Conn, mcnet.ConfigureOptions{
		Locale:       "en_us",
		ViewDistance: 8,
	})
	if err != nil {
		return fmt.Errorf("configuration failed: %w", err)
	}

	log.Println("Configuration completed, entering Play state")

	return runPlayLoop(&safeConn{Conn: res.Conn})
}

func runPlayLoop(conn *safeConn) error {
	var (
		posMu      sync.Mutex
		x, y, z    float64
		yaw, pitch float32
		spawned    bool
	)

	done := make(chan struct{})
	defer close(done)

	// Send one movement update + Client Tick End (0x0D) per server tick (50ms).
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				posMu.Lock()
				if !spawned {
					posMu.Unlock()
					continue
				}
				yaw += 2.0
				cx, cy, cz, cyaw, cpitch := x, y, z, yaw, pitch
				posMu.Unlock()

				_ = conn.writePacket(pk.Marshal(
					0x1F, // Set Player Position and Rotation
					pk.Double(cx),
					pk.Double(cy),
					pk.Double(cz),
					pk.Float(cyaw),
					pk.Float(cpitch),
					pk.Byte(0x01), // On Ground
				))
				_ = conn.writePacket(pk.Marshal(0x0D)) // Client Tick End (required in MC 26.3)
			}
		}
	}()

	var p pk.Packet
	for {
		if err := conn.ReadPacket(&p); err != nil {
			return fmt.Errorf("read play packet: %w", err)
		}

		switch p.ID {
		case 0x0B: // Chunk Batch Finished
			if err := conn.writePacket(pk.Marshal(0x0B, pk.Float(20.0))); err != nil { // Chunk Batch Received
				return fmt.Errorf("send chunk batch received: %w", err)
			}

		case 0x20: // Disconnect (Play)
			return fmt.Errorf("disconnected by server during play (payload: %x)", p.Data)

		case 0x2D: // Keep Alive (Play)
			var keepAliveID pk.Long
			if err := p.Scan(&keepAliveID); err != nil {
				return fmt.Errorf("parse keep alive: %w", err)
			}
			if err := conn.writePacket(pk.Marshal(0x1C, keepAliveID)); err != nil {
				return fmt.Errorf("send keep alive: %w", err)
			}

		case 0x32: // Login (Play)
			log.Println("Spawned into world")

		case 0x49: // Synchronize Player Position
			var (
				teleportID             pk.VarInt
				px, py, pz, vx, vy, vz pk.Double
				pyaw, ppitch           pk.Float
			)
			if err := p.Scan(&teleportID, &px, &py, &pz, &vx, &vy, &vz, &pyaw, &ppitch); err != nil {
				return fmt.Errorf("parse synchronize player position: %w", err)
			}

			if err := conn.writePacket(pk.Marshal(0x00, teleportID, px, py, pz, pyaw, ppitch)); err != nil { // Accept Teleportation
				return fmt.Errorf("send accept teleportation: %w", err)
			}
			if err := conn.writePacket(pk.Marshal(0x1F, px, py, pz, pyaw, ppitch, pk.Byte(0x00))); err != nil {
				return fmt.Errorf("send initial position: %w", err)
			}
			if err := conn.writePacket(pk.Marshal(0x0D)); err != nil { // Client Tick End
				return fmt.Errorf("send client tick end: %w", err)
			}

			posMu.Lock()
			x, y, z = float64(px), float64(py), float64(pz)
			yaw, pitch = float32(pyaw), float32(ppitch)
			spawned = true
			posMu.Unlock()

			log.Printf("Initial position: x=%.2f y=%.2f z=%.2f", x, y, z)
		}
	}
}