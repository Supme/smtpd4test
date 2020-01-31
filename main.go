package main

import (
	"flag"
	"fmt"
	"github.com/Supme/smtpd"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

var keys struct {
	port  string
	debug bool
}

func main() {
	flag.StringVar(&keys.port, "p", "25", "Listen port")
	flag.BoolVar(&keys.debug, "d", false, "Show debug message")
	flag.Parse()

	port, err := strconv.Atoi(keys.port)
	if err != nil || port < 1 || port > 65535 {
		fmt.Println("Port mast be integer 1-65535")
		os.Exit(1)
	}

	server := &smtpd.Server{
		Handler: handler,
	}
	go func() {
		err = server.ListenAndServe(":" + keys.port)
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
	}()

	fmt.Println("Server listen on port", port)
	wait := make(chan struct{}, 0)
	<-wait
}

func handler(peer smtpd.Peer, env smtpd.Envelope) error {
	if keys.debug {
		log.Printf("Peer helo name: '%s', server name: '%s', sender: '%s', recipients: '%v'", peer.HeloName, peer.ServerName, env.Sender, env.Recipients)
	}

	wait := time.Duration(rand.Int()/10000000000) * time.Nanosecond
	time.Sleep(wait)

	switch rand.Intn(5) {
	case 4:
		return smtpd.Error{Code: 450, Message: "Come back later"}
	case 5:
		return smtpd.Error{Code: 550, Message: "Don't come again"}
	}

	return nil
}
