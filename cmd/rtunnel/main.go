package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

func main() {
	mode := flag.String("mode", "", "server or client")
	tunnel := flag.String("tunnel", "", "tunnel server address (client mode: remote server tunnel port)")
	forward := flag.String("forward", "", "local address to forward (client mode: e.g. 127.0.0.1:22)")
	tunnelPort := flag.Int("tunnel-port", 7000, "server mode: port for tunnel client connections")
	sshPort := flag.Int("ssh-port", 2222, "server mode: port for SSH client connections")
	secret := flag.String("secret", "", "shared secret for authentication")
	flag.Parse()

	if *mode == "" {
		fmt.Println("rtunnel - Reverse TCP Tunnel")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  Server (public IP):  rtunnel -mode server -tunnel-port 7000 -ssh-port 2222 [-secret XXX]")
		fmt.Println("  Client (behind NAT): rtunnel -mode client -tunnel <server>:7000 -forward 127.0.0.1:22 [-secret XXX]")
		fmt.Println("")
		fmt.Println("After client connects, SSH via: ssh -p 2222 user@<server>")
		return
	}

	switch *mode {
	case "server":
		runServer(*tunnelPort, *sshPort, *secret)
	case "client":
		if *tunnel == "" || *forward == "" {
			log.Fatal("client mode requires -tunnel and -forward")
		}
		runClient(*tunnel, *forward, *secret)
	default:
		log.Fatalf("unknown mode: %s (use 'server' or 'client')", *mode)
	}
}

func runServer(tunnelPort, sshPort int, secret string) {
	tunnelAddr := fmt.Sprintf(":%d", tunnelPort)
	sshAddr := fmt.Sprintf(":%d", sshPort)

	tunnelListener, err := net.Listen("tcp", tunnelAddr)
	if err != nil {
		log.Fatalf("tunnel listen failed: %v", err)
	}
	sshListener, err := net.Listen("tcp", sshAddr)
	if err != nil {
		log.Fatalf("ssh listen failed: %v", err)
	}

	log.Printf("rtunnel server started")
	log.Printf("  tunnel port: %d (client connects here)", tunnelPort)
	log.Printf("  ssh port:    %d (ssh into here)", sshPort)
	if secret != "" {
		log.Printf("  auth: secret-based")
	} else {
		log.Printf("  auth: none (set -secret for security)")
	}

	tunnelConnCh := make(chan net.Conn, 8)

	go func() {
		for {
			conn, err := tunnelListener.Accept()
			if err != nil {
				log.Printf("tunnel accept error: %v", err)
				continue
			}
			if secret != "" {
				buf := make([]byte, 256)
				n, err := conn.Read(buf)
				if err != nil || string(buf[:n]) != secret {
					log.Printf("tunnel auth failed from %s", conn.RemoteAddr())
					conn.Close()
					continue
				}
				conn.Write([]byte("ok"))
			}
			log.Printf("tunnel client connected: %s", conn.RemoteAddr())
			tunnelConnCh <- conn
		}
	}()

	for {
		sshConn, err := sshListener.Accept()
		if err != nil {
			log.Printf("ssh accept error: %v", err)
			continue
		}
		log.Printf("ssh connection from: %s", sshConn.RemoteAddr())

		tunnelConn := <-tunnelConnCh
		log.Printf("bridging %s <-> tunnel", sshConn.RemoteAddr())

		go func(s, t net.Conn) {
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				io.Copy(s, t)
				s.Close()
				t.Close()
			}()
			go func() {
				defer wg.Done()
				io.Copy(t, s)
				t.Close()
				s.Close()
			}()
			wg.Wait()
			log.Printf("connection closed: %s", s.RemoteAddr())
		}(sshConn, tunnelConn)
	}
}

func runClient(tunnelAddr, forwardAddr, secret string) {
	for {
		err := connectTunnel(tunnelAddr, forwardAddr, secret)
		if err != nil {
			log.Printf("tunnel disconnected: %v, reconnecting in 3s...", err)
		} else {
			log.Printf("tunnel closed, reconnecting in 3s...")
		}
		select {
		case <-time.After(3 * time.Second):
		}
	}
}

func connectTunnel(tunnelAddr, forwardAddr, secret string) error {
	tunnelConn, err := net.DialTimeout("tcp", tunnelAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connect tunnel: %w", err)
	}
	defer tunnelConn.Close()

	if secret != "" {
		_, err := tunnelConn.Write([]byte(secret))
		if err != nil {
			return fmt.Errorf("send secret: %w", err)
		}
		buf := make([]byte, 16)
		tunnelConn.SetReadDeadline(time.Now().Add(5 * time.Second))
		n, err := tunnelConn.Read(buf)
		if err != nil || string(buf[:n]) != "ok" {
			return fmt.Errorf("auth failed")
		}
		tunnelConn.SetReadDeadline(time.Time{})
	}

	log.Printf("tunnel established: %s -> %s -> %s", forwardAddr, tunnelAddr, "server")

	localConn, err := net.DialTimeout("tcp", forwardAddr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connect local %s: %w", forwardAddr, err)
	}
	defer localConn.Close()

	log.Printf("forwarding: %s <-> %s", tunnelConn.RemoteAddr(), forwardAddr)

	done := make(chan struct{}, 2)

	go func() {
		io.Copy(tunnelConn, localConn)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(localConn, tunnelConn)
		done <- struct{}{}
	}()

	<-done
	return nil
}
