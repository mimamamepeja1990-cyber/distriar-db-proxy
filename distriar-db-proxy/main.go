package main
import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"time"
	"tailscale.com/tsnet"
)
const (
	listenAddr  = ":5432"
	targetAddr  = "100.127.197.79:5432"
)
func main() {
	authKey := os.Getenv("TS_AUTHKEY")
	if authKey == "" {
		log.Fatal("TS_AUTHKEY is not set")
	}
	ts := &tsnet.Server{
		Hostname: "distriar-db-proxy",
		AuthKey:  authKey,
		Ephemeral: true,
	}
	if err := ts.Start(); err != nil {
		log.Fatalf("tailscale start: %v", err)
	}
	defer ts.Close()
	log.Printf("Tailscale connected as distriar-db-proxy")
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("listen %s: %v", listenAddr, err)
	}
	defer ln.Close()
	log.Printf("TCP proxy listening on %s -> %s", listenAddr, targetAddr)
	for {
		client, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go proxy(ts, client)
	}
}
func proxy(ts *tsnet.Server, client net.Conn) {
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	server, err := ts.Dial(ctx, "tcp", targetAddr)
	if err != nil {
		log.Printf("dial A32: %v", err)
		return
	}
	defer server.Close()
	errCh := make(chan error, 2)
	go func() {
		_, err := io.Copy(server, client)
		errCh <- err
	}()
	go func() {
		_, err := io.Copy(client, server)
		errCh <- err
	}()
	<-errCh
}