package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	conn *websocket.Conn
	room *Room
	id   string
	name string
	send chan []byte
}

type Room struct {
	mu        sync.Mutex
	name      string
	clients    map[*Client]bool
	votes      map[string]string
	revealed   bool
	timerStart time.Time
	decisionTime string
}

type Hub struct {
	mu    sync.Mutex
	rooms map[string]*Room
}

func newHub() *Hub {
	return &Hub{rooms: make(map[string]*Room)}
}

func (h *Hub) getOrCreateRoom(name string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r, ok := h.rooms[name]; ok {
		return r
	}
	r := &Room{
		name:       name,
		clients:    make(map[*Client]bool),
		votes:      make(map[string]string),
		timerStart: time.Now(),
	}
	h.rooms[name] = r
	return r
}

func (h *Hub) removeRoom(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms, name)
}

type Msg struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

var fibonacci = []string{"1", "2", "3", "5", "8", "13", "21", "34", "55", "89", "?"}

var templates *template.Template

func main() {
	templates = template.Must(template.ParseGlob("templates/*.html"))
	hub := newHub()
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		templates.ExecuteTemplate(c.Writer, "index.html", nil)
	})

	r.GET("/ws", func(c *gin.Context) {
		teamName := c.Query("team")
		userName := c.Query("name")
		if teamName == "" || userName == "" {
			c.Status(http.StatusBadRequest)
			return
		}
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println("upgrade error:", err)
			return
		}
		room := hub.getOrCreateRoom(teamName)
		client := &Client{conn: conn, room: room, id: uuid.NewString(), name: userName, send: make(chan []byte, 256)}
		room.mu.Lock()
		room.clients[client] = true
		room.mu.Unlock()

		go client.writePump()
		go client.readPump(hub)

		room.broadcast(hub)
	})

	log.Println("Listening on :7878")
	r.Run(":7878")
}

func (c *Client) readPump(hub *Hub) {
	defer func() {
		c.room.mu.Lock()
		delete(c.room.clients, c)
		delete(c.room.votes, c.id)
		empty := len(c.room.clients) == 0
		roomName := c.room.name
		c.room.mu.Unlock()
		if empty {
			hub.removeRoom(roomName)
		} else {
			c.room.broadcast(hub)
		}
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		var msg Msg
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "vote":
			c.room.mu.Lock()
			c.room.votes[c.id] = msg.Payload
			c.room.mu.Unlock()
			c.room.broadcast(hub)
		case "reveal":
			c.room.mu.Lock()
			c.room.revealed = true
			elapsed := int(time.Since(c.room.timerStart).Seconds())
			c.room.decisionTime = fmt.Sprintf("%d:%02d", elapsed/60, elapsed%60)
			c.room.mu.Unlock()
			c.room.broadcast(hub)
		case "clear":
			c.room.mu.Lock()
			c.room.votes = make(map[string]string)
			c.room.revealed = false
			c.room.decisionTime = ""
			c.room.timerStart = time.Now()
			c.room.mu.Unlock()
			c.room.broadcast(hub)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type Summary struct {
	Average      string
	Consensus    bool
	Low          string
	High         string
	DecisionTime string
}

type RoomState struct {
	Members      []MemberState
	Revealed     bool
	AllVoted     bool
	TimerStart   int64
	TimerDisplay string
	Summary      Summary
}

type MemberState struct {
	ID    string
	Name  string
	Voted bool
	Vote  string
}

func calcSummary(votes map[string]string) Summary {
	var nums []float64
	for _, v := range votes {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			nums = append(nums, f)
		}
	}
	if len(nums) == 0 {
		return Summary{}
	}
	sum, lo, hi := 0.0, nums[0], nums[0]
	for _, n := range nums {
		sum += n
		if n < lo {
			lo = n
		}
		if n > hi {
			hi = n
		}
	}
	avg := sum / float64(len(nums))
	consensus := lo == hi

	fmtNum := func(f float64) string {
		if f == float64(int(f)) {
			return fmt.Sprintf("%d", int(f))
		}
		return fmt.Sprintf("%.1f", f)
	}

	return Summary{
		Average:   fmtNum(avg),
		Consensus: consensus,
		Low:       fmtNum(lo),
		High:      fmtNum(hi),
	}
}

func (r *Room) broadcast(hub *Hub) {
	r.mu.Lock()
	defer r.mu.Unlock()

	members := make([]MemberState, 0, len(r.clients))
	allVoted := len(r.clients) > 0
	for c := range r.clients {
		voted := r.votes[c.id] != ""
		if !voted {
			allVoted = false
		}
		vote := ""
		if r.revealed {
			vote = r.votes[c.id]
		}
		members = append(members, MemberState{ID: c.id, Name: c.name, Voted: voted, Vote: vote})
	}
	sort.Slice(members, func(i, j int) bool {
		if members[i].Name != members[j].Name {
			return members[i].Name < members[j].Name
		}
		return members[i].ID < members[j].ID
	})

	var summary Summary
	if r.revealed {
		summary = calcSummary(r.votes)
		summary.DecisionTime = r.decisionTime
	}

	elapsed := int(time.Since(r.timerStart).Seconds())
	timerDisplay := fmt.Sprintf("%d:%02d", elapsed/60, elapsed%60)

	state := RoomState{
		Members:      members,
		Revealed:     r.revealed,
		AllVoted:     allVoted,
		TimerStart:   r.timerStart.UnixMilli(),
		TimerDisplay: timerDisplay,
		Summary:      summary,
	}

	for c := range r.clients {
		myVote := r.votes[c.id]
		data := buildRoomHTML(state, c.id, myVote, fibonacci)
		select {
		case c.send <- data:
		default:
			close(c.send)
			delete(r.clients, c)
		}
	}
}

func buildRoomHTML(state RoomState, myID string, myVote string, fibs []string) []byte {
	type tmplData struct {
		State  RoomState
		MyID   string
		MyVote string
		Fibs   []string
	}
	var buf []byte
	t := templates.Lookup("room.html")
	if t == nil {
		return []byte("<div>template error</div>")
	}
	w := &bytesWriter{buf: &buf}
	t.Execute(w, tmplData{State: state, MyID: myID, MyVote: myVote, Fibs: fibs})
	return buf
}

type bytesWriter struct {
	buf *[]byte
}

func (bw *bytesWriter) Write(p []byte) (int, error) {
	*bw.buf = append(*bw.buf, p...)
	return len(p), nil
}
