package websocket

import "testing"

func newManagerTestServer() *Server {
	return &Server{Clients: make(map[uint]*Client)}
}

func TestDLQ_TC_005_WS_001_RegisterReplacesCurrentClient(t *testing.T) {
	server := newManagerTestServer()
	first := &Client{ID: 42}
	second := &Client{ID: 42}
	other := &Client{ID: 43}

	replaced, total := server.registerClient(first)
	if replaced != nil || total != 1 {
		t.Fatalf("first registration = (%p, %d), want (nil, 1)", replaced, total)
	}
	if replaced, total = server.registerClient(other); replaced != nil || total != 2 {
		t.Fatalf("other registration = (%p, %d), want (nil, 2)", replaced, total)
	}
	replaced, total = server.registerClient(second)
	if replaced != first || total != 2 {
		t.Fatalf("replacement = (%p, %d), want (%p, 2)", replaced, total, first)
	}
	if server.Clients[42] != second || server.Clients[43] != other {
		t.Fatal("replacement changed the wrong manager entry")
	}
}

func TestDLQ_TC_005_WS_001_ReregisterSameClientIsIdempotent(t *testing.T) {
	server := newManagerTestServer()
	client := &Client{ID: 42}
	server.registerClient(client)
	replaced, total := server.registerClient(client)
	if replaced != nil || total != 1 {
		t.Fatalf("same-client registration = (%p, %d), want (nil, 1)", replaced, total)
	}
}

func TestDLQ_TC_006_WS_001_StaleUnregisterCannotRemoveReplacement(t *testing.T) {
	server := newManagerTestServer()
	stale := &Client{ID: 42}
	current := &Client{ID: 42}
	other := &Client{ID: 43}
	server.registerClient(stale)
	server.registerClient(current)
	server.registerClient(other)

	removed, total := server.unregisterClient(stale)
	if removed || total != 2 || server.Clients[42] != current {
		t.Fatalf("stale unregister = (%v, %d, %p), want (false, 2, %p)", removed, total, server.Clients[42], current)
	}
	removed, total = server.unregisterClient(current)
	if !removed || total != 1 || server.Clients[43] != other {
		t.Fatalf("current unregister = (%v, %d), want (true, 1) with other client intact", removed, total)
	}
}

func TestDLQ_TC_006_WS_001_SnapshotIsDetached(t *testing.T) {
	server := newManagerTestServer()
	client := &Client{ID: 42}
	server.registerClient(client)
	snapshot := server.clientSnapshot()
	if len(snapshot) != 1 || snapshot[0] != client {
		t.Fatalf("snapshot = %#v, want current client", snapshot)
	}
	snapshot[0] = nil
	if server.Clients[42] != client {
		t.Fatal("mutating snapshot changed manager state")
	}
}
