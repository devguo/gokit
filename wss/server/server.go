package main

import (
	"net/http"
	"github.com/gorilla/websocket"
	"log"
	"flag"
)

var addr = flag.String("addr", "0.0.0.0:8080", "http service address")

var upgrader = websocket.Upgrader{
	CheckOrigin:func(r *http.Request) bool {
		return true
	},
}

func echo(w http.ResponseWriter, r *http.Request){
	c, err := upgrader.Upgrade(w,r,nil)

	if err != nil{
		log.Printf("upgrade: %v", err)
		return
	}

	defer c.Close()

	for {
		mt, msg, err := c.ReadMessage()
		if err != nil {
			log.Printf("read error:%v",err)
			break
		}

		log.Printf("recv: %s",msg)

		err = c.WriteMessage(mt,msg)
		if err != nil {
			log.Printf("write error: %v",err)
			break
		}
	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/",echo)

	log.Fatal(http.ListenAndServeTLS(*addr,"./keyfile/wss.crt","./keyfile/wss.key",nil))

}
