package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

type client struct {
	conn       *websocket.Conn
	send       chan interface{}
	monitorNos chan []string
}

func NewClient(conn *websocket.Conn) *client {
	return &client{
		conn:       conn,
		send:       make(chan interface{}),
		monitorNos: make(chan []string),
	}
}

func (c *client) Write() {
	defer c.conn.Close()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteJSON(message)
		}
	}

}

func (c *client) Read() {
	defer func() {
		hub.removeClient <- c
		c.conn.Close()
	}()

	for {
		var message []string
		err := c.conn.ReadJSON(&message)
		if err != nil {
			log.Errorf("Read message : %v", err)
			hub.removeClient <- c
			c.conn.Close()
			break
		}

		c.monitorNos <- message
		log.Infof("Message from %v : %v", *c, message)
	}
}

func (c *client) Subscribe(rethinkAddr string) {
	var ctx context.Context
	var cancel context.CancelFunc
	defer func() {
		log.Infoln("Subscribe defer")
		if cancel != nil {
			cancel()
		}
	}()

	for {
		select {
		case monitorNos, ok := <-c.monitorNos:
			if cancel != nil {
				cancel()
			}
			if !ok {
				return
			}

			ctx, cancel = context.WithCancel(context.Background())
			go c.retryDoSubscribe(ctx, rethinkAddr, monitorNos)

		}
	}

}

func (c *client) retryDoSubscribe(ctxRoot context.Context, rethinkAddr string, monitorNos []string) {
	defer log.Infoln("Session defer")
	for {
		ctx, cancel := context.WithCancel(ctxRoot)
		defer cancel()

		session, err := r.Connect(r.ConnectOpts{
			Address: rethinkAddr, // endpoint without http
		})
		if err != nil {
			log.Errorln(err)
			time.Sleep(5 * time.Second)
			continue
		}
		defer session.Close()
		session.Use(DbName)

		log.Infof("Subscribe monitorNos : %v", monitorNos)

		errChan := make(chan error)

		var filter *r.Term
		//filter a slice without allocating a new underlying array
		//create a zero-length slice with the same underlying array
		nos := monitorNos[:0]
		for _, v := range monitorNos {
			if v != "" {
				nos = append(nos, v)
			}
		}
		if len(nos) > 0 {
			pattern := fmt.Sprintf("[%s]", strings.Join(monitorNos, ","))
			f := r.Row.Field("MonitorNo").Match(pattern)
			filter = &f
		}

		go c.changeFeed(ctx, session, TableName_SubscribeCustRisk, filter, errChan)

		select {
		case err := <-errChan:
			log.Errorf("Changefeed failed: %v", err)
			cancel()
			session.Close()
			time.Sleep(5 * time.Second)
			break
		case <-ctx.Done():
			log.Info("Session done")
			return
		}
	}
}

func (c *client) changeFeed(ctx context.Context, session *r.Session, table string, filter *r.Term, errChan chan<- error) {
	defer log.Infof("ChangeFeed defer: %s", table)
	query := r.Table(table)
	if filter != nil {
		query = query.Filter(*filter)
	}

	res, err := query.Changes(r.ChangesOpts{IncludeInitial: true}).Run(session)
	if err != nil {
		log.Errorln(err)
		errChan <- err
		return
	}
	defer res.Close()

	var value interface{}
	for res.Next(&value) {
		select {
		default:
			m := value.(map[string]interface{})
			if v, ok := m["new_val"]; ok {
				c.send <- v
			} else if v, ok := m["old_val"]; ok {
				c.send <- v
			}
		case <-ctx.Done():
			return
		}
	}
	errChan <- res.Err()
}
