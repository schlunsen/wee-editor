package agents

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/codexapp"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/stretchr/testify/require"
)

func TestCodexAppFollowUps(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 needed for fake CLI")
	}
	for _, mode := range []string{"steer", "completion-race", "startup"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			script := `#!PYTHON
import json,sys,os
count=0
mode=os.environ['WEE_FAKE_MODE']
def emit(v):
 print(json.dumps(v),flush=True)
def done():
 emit({'method':'item/completed','params':{'threadId':'thread-1','turnId':'turn-1','item':{'type':'agentMessage','id':'answer','text':'follow-ups applied'}}})
 emit({'method':'turn/completed','params':{'threadId':'thread-1','turn':{'id':'turn-1','status':'completed'}}})
for line in sys.stdin:
 r=json.loads(line)
 with open(os.environ['WEE_FAKE_LOG'],'a') as f: f.write(line)
 if 'id' not in r: continue
 method=r['method']
 result={}
 if method in ('thread/start','thread/resume'): result={'thread':{'id':'thread-1'}}
 if method=='turn/start': result={'turn':{'id':'turn-1'}}
 if method=='turn/steer' and mode=='completion-race':
  emit({'id':r['id'],'error':{'code':-32600,'message':'No active turn'}})
  continue
 emit({'id':r['id'],'result':result})
 if method=='turn/start':
  count+=1
  if mode=='completion-race' and count==2: done()
  elif count==1:
   emit({'method':'item/started','params':{'threadId':'thread-1','turnId':'turn-1','item':{'type':'commandExecution','id':'cmd','command':'echo waiting','status':'inProgress'}}})
 if method=='turn/steer':
  count+=1
  if count==3: done()
`
			script = strings.Replace(script, "PYTHON", python, 1)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "codex"), []byte(script), 0700))
			logPath := filepath.Join(dir, "requests.jsonl")
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("WEE_FAKE_MODE", mode)
			t.Setenv("WEE_FAKE_LOG", logPath)
			database.ResetInstance()
			db, err := database.Initialize(t.TempDir())
			require.NoError(t, err)
			defer db.Close()
			sm, err := NewSessionManager(&Config{Model: "gpt-6-astra", MaxConcurrentSessions: 5}, db.GetDB())
			require.NoError(t, err)
			provider, model := CodexProviderID, "gpt-6-astra"
			sess, err := sm.CreateSession(uuid.New(), SessionOptions{Provider: &provider, Model: &model, WorkingDirectory: &dir})
			require.NoError(t, err)
			agent, err := sm.GetSession(sess.ID)
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(agent.ctx, 10*time.Second)
			defer cancel()
			agent.ctx = ctx
			require.NoError(t, sm.SendPrompt(sess.ID, "initial task", nil))
			if mode != "startup" {
				select {
				case <-agent.responseChan:
				case <-ctx.Done():
					t.Fatal("initial turn never started")
				}
			}
			require.NoError(t, sm.SendPrompt(sess.ID, "first correction", nil))
			if mode != "completion-race" {
				require.NoError(t, sm.SendPrompt(sess.ID, "second correction", nil))
			}
			agent.codexTurnMu.Lock()
			run := agent.codexRun
			agent.codexTurnMu.Unlock()
			for {
				select {
				case m := <-agent.responseChan:
					if result, ok := m.(*types.ResultMessage); ok {
						require.False(t, result.IsError)
						goto finished
					}
				case <-ctx.Done():
					t.Fatal("follow-up did not complete active run")
				}
			}
		finished:
			select {
			case <-run.done:
			case <-ctx.Done():
				t.Fatal("run did not shut down")
			}
			data, err := os.ReadFile(logPath)
			require.NoError(t, err)
			var methods []string
			var corrections []string
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				var r struct {
					Method string `json:"method"`
					Params struct {
						Expected string `json:"expectedTurnId"`
						Input    []struct {
							Text string `json:"text"`
						} `json:"input"`
					} `json:"params"`
				}
				require.NoError(t, json.Unmarshal([]byte(line), &r))
				methods = append(methods, r.Method)
				if r.Method == "turn/steer" {
					require.Equal(t, "turn-1", r.Params.Expected)
					corrections = append(corrections, r.Params.Input[0].Text)
				}
			}
			require.NotContains(t, methods, "account/login/start")
			require.Equal(t, "first correction", corrections[0])
			if mode != "completion-race" {
				require.Equal(t, []string{"first correction", "second correction"}, corrections)
			}
			require.Equal(t, SessionStatusIdle, agent.Status)
		})
	}
}

func TestCodexAppItems(t *testing.T) {
	state := newCodexTurnState()
	emits, err := state.handleAppItem("item/started", json.RawMessage(`{"type":"commandExecution","id":"cmd","command":"ls","status":"inProgress"}`))
	require.NoError(t, err)
	require.Len(t, emits, 1)
	id := emits[0].ToolID
	emits, err = state.handleAppItem("item/completed", json.RawMessage(`{"type":"commandExecution","id":"cmd","command":"ls","status":"completed","aggregatedOutput":"files","exitCode":0}`))
	require.NoError(t, err)
	require.Len(t, emits, 1)
	require.Equal(t, id, emits[0].ToolID)
	require.Equal(t, "files", emits[0].Text)
	emits, err = state.handleAppItem("item/completed", json.RawMessage(`{"type":"fileChange","id":"patch","status":"completed","changes":[{"path":"a.go","kind":{"type":"update"}}]}`))
	require.NoError(t, err)
	require.Equal(t, "update a.go", emits[1].Text)
	emits, err = state.handleAppItem("item/completed", json.RawMessage(`{"type":"reasoning","id":"r","summary":["thinking"]}`))
	require.NoError(t, err)
	require.Equal(t, "thinking", emits[0].Text)
	require.False(t, codexNoActiveTurn(context.DeadlineExceeded))
	require.False(t, codexNoActiveTurn(&codexapp.RPCError{Message: "invalid input"}))
}
