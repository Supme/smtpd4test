package main

import (
	"blitiri.com.ar/go/spf"
	"flag"
	"fmt"
	"github.com/Supme/smtpd"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	//"github.com/emersion/go-msgauth/dkim"
)

const version = "0.0.1-beta3"

var keys struct {
	domains   []string
	port      string
	checkHelo bool
	checkSPF  bool
	//checkDKIM bool
	debug bool
}

func main() {
	flag.StringVar(&keys.port, "p", "25", "Listen port")
	flag.BoolVar(&keys.debug, "d", false, "Show debug message")
	flag.BoolVar(&keys.checkHelo, "helo", false, "Check HELO reverse IP")
	flag.BoolVar(&keys.checkSPF, "spf", false, "Check SPF")
	//cdkim := flag.Bool("spf", false, "Check DKIM")
	v := flag.Bool("v", false, "Show version")
	flag.Parse()

	keys.domains = flag.Args()

	if *v {
		fmt.Println("smtpd4test version", version)
		return
	}

	port, err := strconv.Atoi(keys.port)
	if err != nil || port < 1 || port > 65535 {
		fmt.Println("Port mast be integer 1-65535")
		os.Exit(1)
	}

	server := &smtpd.Server{
		Hostname:         "smtpd4test",
		HeloChecker:      heloChecker,
		SenderChecker:    senderChecker,
		RecipientChecker: recipientChecker,
		Handler:          handler,
		DataWriter:       dataDiscardWriter,
	}

	rand.NewSource(time.Now().UnixNano())

	go func() {
		err = server.ListenAndServe(":" + keys.port)
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
	}()

	fmt.Println("Server listen on port", port)
	if len(keys.domains) > 0 {
		fmt.Printf("Receive for domains %v checkHelo=%t checkSPF=%t\r\n", keys.domains, keys.checkHelo, keys.checkSPF)
	}
	wait := make(chan struct{}, 0)
	<-wait
}

func heloChecker(peer smtpd.Peer, name string) error {
	wait := time.Duration(rand.Int()/10000000000) * time.Nanosecond
	time.Sleep(wait)
	if keys.checkHelo {
		host, _, _ := net.SplitHostPort(peer.Addr.String())
		names, err := net.LookupAddr(host)
		debug("Peer addr: '%s' lookup %v", peer.Addr.String(), names)
		if err != nil {
			debug("Peer addr: '%s' wait HELO '%s' has not resolved IP", peer.Addr.String(), wait)
			return smtpd.Error{Code: 550, Message: "Your HELO/EHLO greeting must resolve"}
		}
		if !hasDomainInArray(names, name) {
			debug("Peer addr: '%s' wait HELO '%s' greeting '%s' but has resolving '%v", peer.Addr.String(), wait, name, names)
			return smtpd.Error{Code: 550, Message: "Your HELO/EHLO greeting does not match you IP"}
		}
	}
	debug("Peer addr: '%s' wait HELO '%s'", peer.Addr.String(), wait)
	return nil
}

func senderChecker(peer smtpd.Peer, addr string) error {
	if !keys.checkSPF {
		return nil
	}
	sAddr := strings.Split(addr, "@")
	if len(sAddr) == 2 {
		host, _, _ := net.SplitHostPort(peer.Addr.String())
		ip := net.ParseIP(host)
		r, err := spf.CheckHostWithSender(ip, peer.HeloName, addr)
		if err != nil {

		}
		debug("Peer addr: '%s' check SPF result '%s' for domain '%s'", peer.Addr.String(), r, sAddr[1])
		if r == spf.Fail || r == spf.PermError || r == spf.TempError {
			return smtpd.Error{Code: 550, Message: "Check spf '" + string(r) + "'"}
		}
		if r != spf.Pass && r != spf.None {
			return smtpd.Error{Code: 421, Message: "Check spf '" + string(r) + "'"}
		}
		return nil
	}
	return smtpd.Error{Code: 550, Message: "Invalid from email"}
}

func recipientChecker(peer smtpd.Peer, addr string) error {
	sAddr := strings.Split(addr, "@")
	if len(sAddr) == 2 {
		if hasDomain(sAddr[1]) {
			return nil
		}
		return smtpd.Error{Code: 550, Message: "I'm not a relay"}
	}
	return smtpd.Error{Code: 550, Message: "Invalid email"}
}

func handler(peer smtpd.Peer, env smtpd.Envelope) error {
	var res smtpd.Error
	switch rand.Intn(5) {
	case 4:
		res = smtpd.Error{Code: 450, Message: "Come back later"}
	case 5:
		res = smtpd.Error{Code: 550, Message: "Don't come again"}
	}
	if keys.debug {
		var r int
		if res.Code == 0 {
			r = 250
		} else {
			r = res.Code
		}
		debug("Peer name: '%s', sender: '%s', recipients: '%v' result code: '%v'", peer.HeloName, env.Sender, env.Recipients, r)
	}
	if res.Code != 0 {
		return res
	}
	return nil
}

type DiscardWriteCloser struct{}

func (w DiscardWriteCloser) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w DiscardWriteCloser) Close() error {
	return nil
}

func dataDiscardWriter(peer smtpd.Peer) ([]byte, io.WriteCloser, error) {
	wc := DiscardWriteCloser{}
	return []byte("fakeID"), wc, nil
}

func hasDomain(d string) bool {
	if len(keys.domains) == 0 {
		return true
	}
	if hasDomainInArray(keys.domains, d) {
		return true
	}
	return false
}

func hasDomainInArray(array []string, s string) bool {
	for i := range array {
		if strings.ToLower(strings.TrimSpace(strings.TrimRight(array[i], "."))) == strings.ToLower(strings.TrimSpace(strings.TrimRight(s, "."))) {
			return true
		}
	}
	return false
}

func debug(format string, v ...interface{}) {
	if keys.debug {
		log.Printf(format, v...)
	}
}
