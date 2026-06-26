package scenario

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"
)

func TestMAVLinkUDPSinkWritesNativeFrameToFeedBoundary(t *testing.T) {
	conn := listenUDP(t)
	defer conn.Close()
	frame := []byte{0xfd, 0x00, 0x00, 0x00}
	sink, err := NewMAVLinkUDPSink(UDPFeedSinkConfig{
		Addr:         conn.LocalAddr().String(),
		WriteTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}

	result, err := sink.IngestFrame(context.Background(), frame)
	if err != nil {
		t.Fatalf("ingest frame: %v", err)
	}
	if result.Mutations != 0 || result.RawRef == "" {
		t.Fatalf("result = %+v, want UDP raw ref and zero graph mutations", result)
	}
	got := readUDP(t, conn, len(frame))
	if !bytes.Equal(got, frame) {
		t.Fatalf("udp frame = %v, want %v", got, frame)
	}
	if health := sink.Health(); !health.Ready || health.FramesReceived != 1 || health.GraphMutations != 0 {
		t.Fatalf("health = %+v", health)
	}
}

func TestCoTUDPSinkWritesNativeEventToFeedBoundary(t *testing.T) {
	conn := listenUDP(t)
	defer conn.Close()
	rawXML := []byte(`<event version="2.0" uid="alpha" type="a-f-G-U-C" how="m-g"/>`)
	sink, err := NewCoTUDPSink(UDPFeedSinkConfig{
		Addr:         conn.LocalAddr().String(),
		WriteTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}

	result, err := sink.IngestEvent(context.Background(), rawXML)
	if err != nil {
		t.Fatalf("ingest event: %v", err)
	}
	if result.Mutations != 0 || result.RawRef == "" {
		t.Fatalf("result = %+v, want UDP raw ref and zero graph mutations", result)
	}
	got := readUDP(t, conn, len(rawXML))
	if !bytes.Equal(got, rawXML) {
		t.Fatalf("udp event = %q, want %q", got, rawXML)
	}
	if health := sink.Health(); !health.Ready || health.EventsReceived != 1 || health.GraphMutations != 0 {
		t.Fatalf("health = %+v", health)
	}
}

func TestUDPFeedSinksRejectInvalidConfig(t *testing.T) {
	if _, err := NewMAVLinkUDPSink(UDPFeedSinkConfig{}); err == nil {
		t.Fatal("expected MAVLink sink address error")
	}
	if _, err := NewCoTUDPSink(UDPFeedSinkConfig{Addr: "127.0.0.1:1", WriteTimeout: -1}); err == nil {
		t.Fatal("expected CoT sink timeout error")
	}
}

func listenUDP(t *testing.T) *net.UDPConn {
	t.Helper()
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("resolve udp: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	return conn
}

func readUDP(t *testing.T, conn *net.UDPConn, size int) []byte {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	buffer := make([]byte, size+32)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		t.Fatalf("read udp: %v", err)
	}
	return append([]byte(nil), buffer[:n]...)
}
