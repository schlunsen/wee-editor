package codexapp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func fakeServer(t *testing.T, body string) string {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 needed for fake server")
	}
	path := filepath.Join(t.TempDir(), "codex")
	if err = os.WriteFile(path, []byte("#!"+python+"\n"+body), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNotificationsDoNotBlockRPC(t *testing.T) {
	path := fakeServer(t, `import json,sys
for line in sys.stdin:
 r=json.loads(line)
 for i in range(256): print(json.dumps({'method':'item/completed','params':{'index':i}}),flush=True)
 print(json.dumps({'id':r['id'],'result':{'ok':True}}),flush=True)
`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, err := Start(ctx, path, os.Environ(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var result struct {
		OK bool `json:"ok"`
	}
	if err = c.Call(ctx, "test", nil, &result); err != nil || !result.OK {
		t.Fatalf("RPC blocked behind notifications: %v", err)
	}
	for i := 0; i < 256; i++ {
		select {
		case m := <-c.Events():
			if string(m.Params) != fmt.Sprintf(`{"index": %d}`, i) {
				t.Fatalf("out of order: %s", m.Params)
			}
		case <-ctx.Done():
			t.Fatal("lost notifications")
		}
	}
}

func TestCancellationUnblocksCall(t *testing.T) {
	path := fakeServer(t, "import time\ntime.sleep(60)\n")
	ctx, cancel := context.WithCancel(context.Background())
	c, err := Start(ctx, path, os.Environ(), "")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- c.Call(ctx, "never", nil, nil) }()
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected cancellation")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("call did not unblock")
	}
	c.Close()
}
