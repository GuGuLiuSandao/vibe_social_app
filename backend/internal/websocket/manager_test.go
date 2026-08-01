package websocket

import "testing"

func TestUnregisterStaleClientKeepsReplacement(t *testing.T) {
	server := &Server{Clients: make(map[uint]*Client)}
	oldClient := &Client{ID: 10000001}
	newClient := &Client{ID: 10000001}

	replaced, total := server.registerClient(oldClient)
	if replaced != nil || total != 1 {
		t.Fatalf("unexpected initial registration: replaced=%v total=%d", replaced, total)
	}

	replaced, total = server.registerClient(newClient)
	if replaced != oldClient || total != 1 {
		t.Fatalf("unexpected replacement: replaced=%p total=%d", replaced, total)
	}

	removed, total := server.unregisterClient(oldClient)
	if removed {
		t.Fatalf("stale client removed the active replacement")
	}
	if total != 1 || server.Clients[oldClient.ID] != newClient {
		t.Fatalf("replacement connection was not preserved")
	}
}

func TestUnregisterCurrentClientRemovesConnection(t *testing.T) {
	server := &Server{Clients: make(map[uint]*Client)}
	client := &Client{ID: 10000001}
	server.registerClient(client)

	removed, total := server.unregisterClient(client)
	if !removed {
		t.Fatalf("current client was not removed")
	}
	if total != 0 || len(server.Clients) != 0 {
		t.Fatalf("unexpected clients after removal: total=%d", total)
	}
}

func TestClientSnapshotIsIndependentFromConnectionMap(t *testing.T) {
	server := &Server{Clients: make(map[uint]*Client)}
	client := &Client{ID: 10000001}
	server.registerClient(client)

	snapshot := server.clientSnapshot()
	server.unregisterClient(client)

	if len(snapshot) != 1 || snapshot[0] != client {
		t.Fatalf("snapshot changed after connection map mutation")
	}
}
