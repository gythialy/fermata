package automation

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// onePixel is a 1×1 PNG, the image a fake app answers capture with.
const onePixel = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

// serve fakes the app's socket: it answers each request with the reply of the
// same index, then closes.
func serve(t *testing.T, replies ...map[string]any) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		lines := bufio.NewScanner(conn)
		for i := 0; lines.Scan(); i++ {
			if i >= len(replies) {
				return
			}
			answer, err := json.Marshal(replies[i])
			if err != nil {
				return
			}
			if _, err := conn.Write(append(answer, '\n')); err != nil {
				return
			}
		}
	}()
	return ln.Addr().String()
}

func result(v any) map[string]any { return map[string]any{"id": 1, "result": v} }

func TestFindPrefersTheElementThatActs(t *testing.T) {
	nodes := []Node{
		// A control and the text beside it carry one label: the control wins.
		{ID: "1", Role: "text", Label: "Start at login"},
		{ID: "2", Role: "checkBox", Label: "Start at login"},
		{ID: "3", Role: "text", Label: "English"},
		{ID: "4", Role: "listItem", Label: "English", States: []string{"selectable"}},
		{ID: "5", Role: "tab", Label: "Schedule"},
		{ID: "6", Role: "button", Key: "appearance", Label: "Appearance"},
	}

	got, err := Find(nodes, "", "Start at login", "press")
	if err != nil || got.ID != "2" {
		t.Errorf("control beside its text: got %v, %v", got.ID, err)
	}
	got, err = Find(nodes, "", "English", "press")
	if err != nil || got.ID != "4" {
		t.Errorf("menu item beside its text: got %v, %v", got.ID, err)
	}
	got, err = Find(nodes, "", "Schedule", "press")
	if err != nil || got.ID != "5" {
		t.Errorf("unique label: got %v, %v", got.ID, err)
	}
	got, err = Find(nodes, "appearance", "", "press")
	if err != nil || got.ID != "6" {
		t.Errorf("key: got %v, %v", got.ID, err)
	}
	if _, err := Find(nodes, "", "Close", "press"); err == nil {
		t.Error("a missing label should fail")
	}
}

// TestFindRefusesTwoControls: two elements that both act are a real
// ambiguity, and guessing would click the wrong one.
func TestFindRefusesTwoControls(t *testing.T) {
	nodes := []Node{
		{ID: "1", Role: "switch", Label: "Start at login"},
		{ID: "2", Role: "checkBox", Label: "Start at login"},
	}
	if _, err := Find(nodes, "", "Start at login", "press"); err == nil {
		t.Error("two controls with one label should be refused, not guessed")
	}
}

// TestFindReadsTheDescription: a stepper's button draws "+" and names itself
// "increase" in the description, so the label alone finds nothing.
func TestFindReadsTheDescription(t *testing.T) {
	nodes := []Node{
		{ID: "1", Role: "button", Description: "Increase", Actions: []string{"press"}},
		{ID: "2", Role: "text", Label: "5 min"},
	}
	got, err := Find(nodes, "", "Increase", "press")
	if err != nil || got.ID != "1" {
		t.Errorf("description: got %v, %v", got.ID, err)
	}
}

func TestCaptureWritesThePNG(t *testing.T) {
	addr := serve(t, result(map[string]any{"png": onePixel, "width": 1, "height": 1}))
	c, err := Dial(addr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	out := filepath.Join(t.TempDir(), "nested", "shot.png")
	n, err := c.Capture(1, out)
	if err != nil {
		t.Fatal(err)
	}
	want, err := base64.StdEncoding.DecodeString(onePixel)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(want) || string(got) != string(want) {
		t.Errorf("wrote %d bytes, want %d", n, len(want))
	}
}

func TestTreeReadsTheNodes(t *testing.T) {
	addr := serve(t, result(map[string]any{"window": 1, "nodes": []map[string]any{
		{"id": "7", "role": "tab", "label": "General", "bounds": map[string]any{"x": 1, "y": 2, "width": 3, "height": 4}},
	}}))
	c, err := Dial(addr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	nodes, _, err := c.Tree(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || nodes[0].Label != "General" || nodes[0].Bounds.Width != 3 {
		t.Errorf("nodes = %+v", nodes)
	}
	if !strings.Contains(nodes[0].Line(), `"General"`) {
		t.Errorf("line = %q", nodes[0].Line())
	}
}

// TestAnErrorIsReported: an answer with an error stops the call, named by its
// code, instead of reading as an empty result.
func TestAnErrorIsReported(t *testing.T) {
	addr := serve(t, map[string]any{"id": 1, "error": map[string]any{"code": "no_window", "message": "no window 9"}})
	c, err := Dial(addr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	_, _, err = c.Tree(9)
	if err == nil || !strings.Contains(err.Error(), "no_window") {
		t.Errorf("err = %v, want the protocol's code", err)
	}
}

func TestFindWindowIgnoresCase(t *testing.T) {
	windows := []Window{{ID: 1, Title: "Fermata"}, {ID: 2, Title: "设置"}}
	if w, ok := FindWindow(windows, "fermata"); !ok || w.ID != 1 {
		t.Errorf("ascii title: %v %v", w.ID, ok)
	}
	if w, ok := FindWindow(windows, "设置"); !ok || w.ID != 2 {
		t.Errorf("cjk title: %v %v", w.ID, ok)
	}
	if _, ok := FindWindow(windows, "About"); ok {
		t.Error("a missing window should not be found")
	}
}
