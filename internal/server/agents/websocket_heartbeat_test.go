package agents

import "testing"

// TestWebSocketHeartbeatTimings pins the relationship between the keepalive
// constants.
//
// These values are only safe in relation to each other, and getting them wrong
// fails in opposite, equally bad ways: too short a wsPongWait closes healthy
// connections mid-task, while too long a one lets a dead peer keep its slot and
// keeps the server buffering messages nobody will ever receive. Neither shows
// up in a build or a normal test run, so pin it explicitly.
func TestWebSocketHeartbeatTimings(t *testing.T) {
	if wsPongWait <= wsPingPeriod {
		t.Fatalf("wsPongWait (%v) must exceed wsPingPeriod (%v), otherwise the read "+
			"deadline expires before the peer can answer a ping and healthy "+
			"connections are closed", wsPongWait, wsPingPeriod)
	}

	// Require room for at least two missed pings. With no headroom a single
	// dropped frame or a slow round-trip would drop a working connection.
	if minWait := 2 * wsPingPeriod; wsPongWait < minWait {
		t.Errorf("wsPongWait (%v) leaves no headroom for a missed ping; want >= %v (2x wsPingPeriod)",
			wsPongWait, minWait)
	}

	if wsWriteWait <= 0 {
		t.Errorf("wsWriteWait (%v) must be positive, otherwise the ping write deadline "+
			"has already expired when it is set", wsWriteWait)
	}

	// The ping write must give up well before the next ping is due, so a stalled
	// write cannot pile goroutines up behind the ticker.
	if wsWriteWait >= wsPingPeriod {
		t.Errorf("wsWriteWait (%v) must be shorter than wsPingPeriod (%v) so a stalled "+
			"ping write cannot overlap the next tick", wsWriteWait, wsPingPeriod)
	}
}
