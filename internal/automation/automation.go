// Package automation speaks the MyGo automation protocol: newline-delimited
// JSON over a socket, the same on macOS, Linux and Windows. A running app
// built with the automation plugin listens where MYGO_AUTOMATION names, and
// this package lists its windows, reads what their native UI shows, acts on
// elements, and captures them as PNG.
//
// The protocol is MyGo's; only the client lives here, so that the tests of
// this app and the e2e command share one reader of it.
package automation

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Client is one connection to a running app. Requests are answered in order,
// one at a time.
type Client struct {
	conn    net.Conn
	read    *bufio.Reader
	timeout time.Duration
	nextID  int
}

// Dial connects to the app listening at addr, as host:port.
func Dial(addr string, timeout time.Duration) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w (is the app running with MYGO_AUTOMATION=tcp:%s?)",
			addr, err, addr)
	}
	return &Client{conn: conn, read: bufio.NewReader(conn), timeout: timeout}, nil
}

// Close closes the connection.
func (c *Client) Close() error { return c.conn.Close() }

// SetTimeout sets how long one request may wait for its answer, which a
// caller waits longer for than a dial when a window animates: idle answers
// only once the window settles, or when its own timeout runs out.
func (c *Client) SetTimeout(d time.Duration) { c.timeout = d }

// Do writes one request and reads its answer, in order, as the protocol
// promises.
func (c *Client) Do(method string, params any) (json.RawMessage, error) {
	c.nextID++
	req := map[string]any{"id": c.nextID, "method": method}
	if params != nil {
		req["params"] = params
	}
	line, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, err
	}
	if _, err := c.conn.Write(append(line, '\n')); err != nil {
		return nil, err
	}
	answer, err := c.read.ReadBytes('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("the app closed the connection")
		}
		return nil, err
	}
	var resp struct {
		ID     json.RawMessage `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(answer, &resp); err != nil {
		return nil, fmt.Errorf("answer %q: %w", strings.TrimSpace(string(answer)), err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("%s: %s: %s", method, resp.Error.Code, resp.Error.Message)
	}
	return resp.Result, nil
}

// Window is one of the app's windows, as Windows lists it.
type Window struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Kind     string `json:"kind"`
	Focused  bool   `json:"focused"`
	Visible  bool   `json:"visible"`
	Bounds   Box    `json:"bounds"`
	Contents Box    `json:"contentBounds"`
}

// Box is a rectangle in DIPs.
type Box struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Windows lists the open windows, in ID order.
func (c *Client) Windows() ([]Window, error) {
	raw, err := c.Do("windows", nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Windows []Window `json:"windows"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	sort.Slice(out.Windows, func(i, j int) bool { return out.Windows[i].ID < out.Windows[j].ID })
	return out.Windows, nil
}

// Node is one element of a window's native UI, as assistive technology reads
// it.
type Node struct {
	ID          string   `json:"id"`
	Parent      string   `json:"parent"`
	Role        string   `json:"role"`
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Value       string   `json:"value"`
	Placeholder string   `json:"placeholder"`
	Description string   `json:"description"`
	Bounds      Box      `json:"bounds"`
	States      []string `json:"states"`
	Actions     []string `json:"actions"`
}

// Text is everything the element says: its label, value, placeholder and
// description, the fields a name, a choice or a hint live in.
func (n Node) Text() string {
	return strings.TrimSpace(strings.Join([]string{n.Label, n.Value, n.Placeholder, n.Description}, " "))
}

// Line writes the element as one readable line.
func (n Node) Line() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s", n.ID, n.Role)
	if n.Key != "" {
		fmt.Fprintf(&b, " key=%s", n.Key)
	}
	if text := n.Text(); text != "" {
		fmt.Fprintf(&b, " %q", text)
	}
	fmt.Fprintf(&b, " (%d,%d %dx%d)", int(n.Bounds.X), int(n.Bounds.Y), int(n.Bounds.Width), int(n.Bounds.Height))
	if len(n.States) > 0 {
		fmt.Fprintf(&b, " states=%s", strings.Join(n.States, ","))
	}
	return b.String()
}

// Has reports whether the element carries the state, as "checked".
func (n Node) Has(state string) bool {
	for _, s := range n.States {
		if s == state {
			return true
		}
	}
	return false
}

// NamedBy reports whether the element goes by name: its label, or the text it
// draws, as a stepper's "increase" sits in its description.
func (n Node) NamedBy(name string) bool {
	return n.Label == name || n.Value == name || n.Description == name || n.Placeholder == name
}

// Tree reads the window's elements, and the answer they came in, for callers
// that want the protocol's own JSON.
func (c *Client) Tree(win int) ([]Node, json.RawMessage, error) {
	raw, err := c.Do("tree", map[string]any{"window": win})
	if err != nil {
		return nil, nil, err
	}
	var out struct {
		Nodes []Node `json:"nodes"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, nil, err
	}
	return out.Nodes, raw, nil
}

// Act performs one action on an element, named by its ID or, when the app
// reads elements as assistive technology does, by its key.
func (c *Client) Act(win int, id, key, action, text string) error {
	params := map[string]any{"window": win, "action": action}
	if id != "" {
		params["id"] = id
	} else {
		params["key"] = key
	}
	if text != "" {
		params["text"] = text
	}
	_, err := c.Do("act", params)
	return err
}

// Event is one input event, as the protocol's input method takes them.
type Event map[string]any

// Input sends events to the window's content, in order.
func (c *Client) Input(win int, events []Event) error {
	_, err := c.Do("input", map[string]any{"window": win, "events": events})
	return err
}

// Capture asks for the window's image and writes the PNG at path.
func (c *Client) Capture(win int, path string) (int, error) {
	raw, err := c.Do("capture", map[string]any{"window": win})
	if err != nil {
		return 0, err
	}
	var out struct {
		PNG    string `json:"png"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return 0, err
	}
	data, err := base64.StdEncoding.DecodeString(out.PNG)
	if err != nil {
		return 0, fmt.Errorf("image: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return 0, err
	}
	return len(data), nil
}

// Idle waits until the window stops asking for frames. A window that never
// settles fails with the protocol's timeout, which callers take as the next
// read happening anyway.
func (c *Client) Idle(win int, within time.Duration) error {
	_, err := c.Do("idle", map[string]any{"window": win, "timeout": int(within.Milliseconds())})
	return err
}

// Find picks the one element a key or a label names, refusing a label that
// several elements carry: a test that acts on the wrong twin is worse than a
// test that stops. A menu item and the text it draws share a label, so among
// elements of one label the one that acts wins.
func Find(nodes []Node, key, label, action string) (Node, error) {
	if key != "" {
		var hits []Node
		for _, n := range nodes {
			if n.Key == key {
				hits = append(hits, n)
			}
		}
		switch len(hits) {
		case 1:
			return hits[0], nil
		case 0:
			return Node{}, fmt.Errorf("no element with key %q", key)
		default:
			return Node{}, fmt.Errorf("%d elements have key %q", len(hits), key)
		}
	}
	var exact, partial []Node
	for _, n := range nodes {
		switch {
		case n.NamedBy(label):
			exact = append(exact, n)
		case strings.Contains(n.Text(), label):
			partial = append(partial, n)
		}
	}
	switch len(exact) {
	case 1:
		return exact[0], nil
	case 0:
	default:
		if one, ok := oneSpeaker(exact, action); ok {
			return one, nil
		}
		return Node{}, fmt.Errorf("%d elements are labelled %q", len(exact), label)
	}
	switch len(partial) {
	case 1:
		return partial[0], nil
	case 0:
		return Node{}, fmt.Errorf("no element labelled %q", label)
	default:
		if one, ok := oneSpeaker(partial, action); ok {
			return one, nil
		}
		var names []string
		for _, n := range partial {
			names = append(names, n.Line())
		}
		return Node{}, fmt.Errorf("%q is ambiguous: %s", label, strings.Join(names, " | "))
	}
}

// oneSpeaker returns the one candidate that offers the action, preferring the
// elements that act over the text they draw.
func oneSpeaker(candidates []Node, action string) (Node, bool) {
	var actors []Node
	for _, n := range candidates {
		if n.Role == "text" || n.Role == "heading" {
			continue
		}
		actors = append(actors, n)
	}
	if len(actors) == 1 {
		return actors[0], true
	}
	var offered []Node
	for _, n := range actors {
		for _, a := range n.Actions {
			if a == action {
				offered = append(offered, n)
				break
			}
		}
	}
	if len(offered) == 1 {
		return offered[0], true
	}
	return Node{}, false
}

// FindWindow returns the one window whose title holds title, ignoring case.
func FindWindow(windows []Window, title string) (Window, bool) {
	for _, w := range windows {
		if strings.Contains(strings.ToLower(w.Title), strings.ToLower(title)) {
			return w, true
		}
	}
	return Window{}, false
}
