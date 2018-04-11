package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"golang.org/x/net/websocket"
)

func startWS() {
	http.HandleFunc("/", helloWorld)
	http.Handle("/ws", websocket.Handler(pong))

	err := http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	out, _ := ioutil.ReadFile("index.html")
	w.Write(out)
	
}

func pong(ws *websocket.Conn){
	
}