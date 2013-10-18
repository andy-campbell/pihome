// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"time"
	"github.com/ghthor/gowol"
)

type Page struct {
	Title string
	Body  []byte
	Status string
}
var currentStatus = "Unknown"
var servAddr = "Dragon:22"
var macAddr = "00:1e:c9:2d:d6:d9"
var bcAddr = "255.255.255.255"



/*
	Tests if the server is currently up and running.
	It does this by resolving the server address and then
	if that passes then will try and connect to the ssh port
	of the server.	
*/
func sendShutDownPacket() {
	shutdownServ := "Dragon:20010"
	init := 0
	if init != 0 {
		time.Sleep(30 * time.Second)
	}
	init = 1
	tcpAddr, err := net.ResolveTCPAddr("tcp", shutdownServ)
	if err != nil {
		println("ResolveTcpAddr failed ")
		return
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		currentStatus = "Off"
		println("Dial failed:", err.Error())
		return
	}

	currentStatus = "Ready"
	conn.Close()

}


/*
	Tests if the server is currently up and running.
	It does this by resolving the server address and then
	if that passes then will try and connect to the ssh port
	of the server.	
*/
func testSshSockUpOnServer() {
	init := 0
	for {
		if init != 0 {
			time.Sleep(30 * time.Second)
		}
		init = 1
		tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
		if err != nil {
			currentStatus = "Off"
			println("ResolveTcpAddr failed ")
			continue
		}

		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			currentStatus = "Off"
			println("Dial failed:", err.Error())
			continue
		}

		currentStatus = "Ready"
		conn.Close()
	}

}

/*
	
*/
func sendMagicPacket() {
	err := wol.SendMagicPacket(macAddr, bcAddr)
	if err != nil {
		println("An error has occured sending magic page please check ")

	}
}


func (p *Page) save() error {
	filename := p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	p, err := loadPage("index.html")
	if err != nil {
		p = &Page{Title: "Home", Status: currentStatus}
	}
	renderTemplate(w, "index", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}


func startServerHandler(w http.ResponseWriter, r *http.Request) {
	println("Sending Magic packet")
	err := wol.SendMagicPacket(macAddr, bcAddr)
	if err != nil {
		println("An error has occured sending magic page please check that you have entered the right info")
	}
	http.Redirect(w, r, "/", http.StatusFound)
}


func endServerHandler(w http.ResponseWriter, r *http.Request) {
	println("Sending Magic packet")
	sendShutDownPacket()
	http.Redirect(w, r, "/", http.StatusFound)
}

var templates = template.Must(template.ParseFiles("edit.html", "view.html", "index.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var validPath = regexp.MustCompile("^/([a-zA-Z0-9]*)$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func main() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/startServer", startServerHandler)
	http.HandleFunc("/endServer", endServerHandler)
	go testSshSockUpOnServer()

	http.ListenAndServe(":8111", nil)

}

