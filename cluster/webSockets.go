package cluster

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"net/http"
	"time"
)

const (
	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * 5 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// How often changes will be updated
	updatePeriod = 2 * time.Second
)

var (
	homeTempl = template.Must(template.New("").Parse(homeHTML))

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	commandChan = make(chan DbCommand, 1000)
)

// will wait for pongs
func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

// will ping the client and send real-time updates of the data store
func writer(ws *websocket.Conn) {
	var message []byte
	pingTicker := time.NewTicker(pingPeriod)
	updateTicker := time.NewTicker(updatePeriod)
	defer func() {
		pingTicker.Stop()
		updateTicker.Stop()
		ws.Close()
	}()
	for {
		select {
		// here is get the latest updates of the store
		case <-updateTicker.C:

			command := <-commandChan
			byteCommand, _ := json.Marshal(command)
			message = append(message, []byte("\n")...)
			message = append(message, byteCommand...)

			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	// converts http connection to websocket connection, it is the http handshake
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	// will send pings and real time updates
	go writer(ws)
	// will read pongs
	reader(ws)
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var v = struct {
		Host string
		Data string
	}{
		r.Host,
		"Here will come the latest updated data",
	}
	homeTempl.Execute(w, &v)
}

func StartWebSocketServer() {

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", serveWs)
	log.Printf("WEBSOCKET server started..")
	if err := http.ListenAndServe(":9080", nil); err != nil {
		log.Fatal(err)
	}
}

const homeHTML = `<!DOCTYPE html>
<html lang="en">
    <head>
        <title>WebSocket Example</title>
    </head>
    <body>
        <pre id="fileData">{{.Data}}</pre>
        <script type="text/javascript">
            (function() {

                var data = document.getElementById("fileData");
                var conn = new WebSocket("ws://{{.Host}}/ws");
                conn.onclose = function(evt) {
                    data.textContent = 'Connection closed';
                }
                conn.onmessage = function(evt) {
                    console.log('file updated');
                    data.textContent = evt.data;
                }
            })();
        </script>
    </body>
</html>
`
