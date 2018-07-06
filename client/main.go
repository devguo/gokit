package main

import (
	"flag"
	"log"
	"time"
	"bytes"
	"github.com/gorilla/websocket"
	"crypto/tls"
)

const (
	PingPeriod          = 54 * time.Second
	PongWait            = 60 * time.Second
	WriteWait           = 10 * time.Second
)

type WsClient struct {
	SrvAddr  string
	conn     *websocket.Conn
	SendChan chan []byte
	RecvChan chan []byte
}

func NewWsClient(srv string) *WsClient {
	c := &WsClient{
		SrvAddr:  srv,
		SendChan: make(chan []byte, 5000),
		RecvChan: make(chan []byte, 5000),
	}
	conn, err := c.dial()
	if err != nil {
		log.Printf("dial error, %v", err)
		return nil
	}

	log.Printf("server connected. %s", c.SrvAddr)
	c.conn = conn
	go c.readPump()
	go c.writePump()

	return c
}

func (client *WsClient) dial() (*websocket.Conn, error) {
	dialer := websocket.Dialer{TLSClientConfig:&tls.Config{InsecureSkipVerify:true}}
	for {
		conn, _, err := dialer.Dial(client.SrvAddr,nil)
		if err == nil {
			return conn, nil
		}

		log.Printf("establish connection failed. %v", err)
		time.Sleep(3 * time.Second)
	}
}

func (c *WsClient) setReadDeadline(d time.Time) {
	if err := c.conn.SetReadDeadline(d); err != nil {
		log.Printf("connection SetReadDeadline failed. %v", err)
	}
}

func (c *WsClient) setWriteDeadline(d time.Time) {
	if err := c.conn.SetReadDeadline(d); err != nil {
		log.Printf("connection SetWriteDeadline failed. %v", err)
	}
}

func (c *WsClient) writeMessage(messageType int, data []byte) {
	if err := c.conn.WriteMessage(messageType, data); err != nil {
		log.Printf("write msg failed. messageType=%d, %v", messageType, err)
	}
}

func (c *WsClient) readPump() {
	defer func() {
		c.conn.Close()
		close(c.RecvChan)
	}()

	//c.ws.SetReadLimit(maxMessageSize)
	c.setReadDeadline(time.Now().Add(PongWait))

	c.conn.SetPongHandler(func(string) error { c.setReadDeadline(time.Now().Add(PongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()

		if err != nil {
			log.Printf("read msg failed. %v", err)
			break
		}
		c.RecvChan <- message
	}
}

func (c *WsClient) writePump() {
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case message, ok := <-c.SendChan:
			c.setWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				c.setWriteDeadline(time.Now().Add(WriteWait))
				// The hub closed the channel.
				c.writeMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				return
			}

			if _, err := w.Write(message); err != nil {
				log.Printf("write message failed. %v", err)
			}

			// Add queued chat messages to the current websocket message.
			n := len(c.SendChan)
			for i := 0; i < n; i++ {
				if _, err := w.Write(<-c.SendChan); err != nil {
					log.Printf("write message failed. %v", err)
				}
			}

			if err := w.Close(); err != nil {
				log.Println("Write error", err)
				return
			}
		case <-ticker.C:
			c.setWriteDeadline(time.Now().Add(WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

var addr = flag.String("addr", "wss://192.168.2.220:8080", "http service addres")


func HandleMsg(data []byte){
	s := string(data)
	log.Printf("recv: %s",s)
}

func RunClient(ws *WsClient){
	for msg := range ws.RecvChan  {
		HandleMsg(msg)
	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	client := NewWsClient(*addr)

	go RunClient(client)

	ticker := time.Tick(5*time.Second)

	for  {
		select {
			case <-ticker:{
				data := "hello world"
				buf := bytes.NewBufferString(data)
				client.SendChan <- buf.Bytes()
			}
		}
	}

}
