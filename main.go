package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/fengdu/risk-websocket-server/core"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var (
	rethinkAddr = flag.String("RethinkDB__Url", "rethinkdb-websocket:28015", "The rethinkdb server address in the format of host:port")
	port        = flag.Int("port", 8080, "The server port")

	upgrader = websocket.Upgrader{
		CheckOrigin:     func(r *http.Request) bool { return true },
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	hub = core.GetHub()
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: time.Stamp,
	})

	flag.Parse()
	parseEnv()

	go hub.Start()

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(res, "risk websocket server from %s", runtime.Version())
	})

	http.HandleFunc("/clients", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(res, "Clients : %v", hub.GetClients())
	})

	http.HandleFunc("/signalr", websocketHandler)

	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "test.html")
	})

	log.Infof("Now server listen on : %v", *port)
	if err := http.ListenAndServe(fmt.Sprintf(":%v", *port), nil); err != nil {
		log.Fatal(err)
	}
}

func parseEnv() {
	rethinkAddrEnv := os.Getenv("RethinkDB__Url")
	if rethinkAddrEnv != "" {
		rethinkAddr = &rethinkAddrEnv
	}
	log.Println("RethinkDB__Url: ", *rethinkAddr)

	portEnv := os.Getenv("port")
	if portEnv != "" {
		if i, err := strconv.Atoi(portEnv); err != nil {
			port = &i
		}
	}
}

func websocketHandler(res http.ResponseWriter, req *http.Request) {
	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		http.NotFound(res, req)
		return
	}

	client := core.NewClient(conn)

	hub.AddClient(client)

	go client.Subscribe(*rethinkAddr)
	go client.Write()
	go client.Read()

}
