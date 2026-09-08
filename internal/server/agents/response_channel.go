package agents

import (
	"time"

	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// responseChanBuffer is the buffer size of a session's response channel.
const responseChanBuffer = 10

// responseChanSendTimeout bounds how long a producer waits for the streamer to
// drain the response channel. It also bounds how long swapResponseChan can be
// blocked by an in-flight send, so a wedged reader can never stall the next
// prompt indefinitely.
const responseChanSendTimeout = 5 * time.Second

// responseSendResult reports how a send to the response channel ended.
type responseSendResult int

const (
	responseSent      responseSendResult = iota // Delivered to the streamer.
	responseAborted                             // Session context cancelled mid-send.
	responseTimedOut                            // Nobody drained the channel in time.
	responseNoChannel                           // Session has no response channel.
)

// sendResponse delivers msg on the session's current response channel.
//
// The channel is swapped and closed on every new prompt and on interrupt (see
// swapResponseChan). Producers therefore must not read session.responseChan
// without holding respMu: a producer that captures the channel just before the
// swap would otherwise send on the closed old channel and panic with
// "send on closed channel". That panic used to unwind the Codex turn goroutine
// before its stream was closed, leaving the codex subprocess alive and holding
// the thread-store writer lock — which made every following prompt fail with
// "thread ... already has an active writer".
//
// Holding respMu for read across the send is what makes the swap safe: the
// writer lock in swapResponseChan waits for in-flight sends to finish, and any
// send that starts afterwards sees the fresh channel.
func (sm *SessionManager) sendResponse(session *AgentSession, msg types.Message) responseSendResult {
	session.respMu.RLock()
	defer session.respMu.RUnlock()

	ch := session.responseChan
	if ch == nil {
		return responseNoChannel
	}

	select {
	case ch <- msg:
		return responseSent
	case <-session.ctx.Done():
		return responseAborted
	case <-time.After(responseChanSendTimeout):
		return responseTimedOut
	}
}

// Readers must NOT take respMu. streamFiberResponses receives the channel by
// parameter and ranges over it until a swap closes it, which is what keeps this
// scheme live: Go's RWMutex parks new readers as soon as Lock is waiting, so a
// reader that re-acquired respMu per message could be blocked by a pending
// swapper while a producer is parked on a full buffer — the two would wedge
// each other until the send timed out.
//
// swapResponseChan installs a fresh response channel and closes the old one,
// so any leaked streamFiberResponses goroutine still ranging over it exits.
// Closing happens under the write lock, which no producer can hold, so a
// concurrent sender can never observe the closed channel.
func swapResponseChan(session *AgentSession) {
	session.respMu.Lock()
	defer session.respMu.Unlock()

	old := session.responseChan
	session.responseChan = make(chan types.Message, responseChanBuffer)
	if old != nil {
		close(old)
	}
}

// logDroppedResponse records a message that never reached the streamer.
func logDroppedResponse(session *AgentSession, what string, result responseSendResult) {
	switch result {
	case responseTimedOut:
		logging.Warning("Session %s: %s dropped — response channel not drained within %s", session.ID, what, responseChanSendTimeout)
	case responseNoChannel:
		logging.Warning("Session %s: %s dropped — no response channel", session.ID, what)
	}
}
