// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"text/template"
	"io/ioutil"
	"net"
	"net/http"
	"time"
        "database/sql"
	"github.com/ghthor/gowol"
        "github.com/hoisie/web"
        "github.com/mattn/go-session-manager"
        _ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"strings"
)

type Page struct {
	Title string
	Body  []byte
	Status string
}

var logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
var manager = session.NewSessionManager(logger)

var currentStatus = "Unknown"
var servAddr = "Dragon:22"
var macAddr = "00:1e:c9:2d:d6:d9"
var bcAddr = "255.255.255.255"

const dbfile = "./user.db"

type User struct {
        UserId string
        Password string
        RealName string
        Age int64
}

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

func rootHandler(ctx *web.Context, session *session.Session) {
	templates.ExecuteTemplate(ctx, "index.html", map[string]interface{} {
		"Value" : session.Value, "Msg": "", "Title" : "Home", "Status" : currentStatus,
	})
}

func signinHandler(ctx *web.Context, session *session.Session) {
	templates.ExecuteTemplate(ctx, "signin.html", map[string]interface{} {
		"Value" : session.Value, "Msg": "",
	})
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


func startServerHandler(ctx *web.Context) {
	err := wol.SendMagicPacket(macAddr, bcAddr)
	if err != nil {
		logger.Printf("An error has occured sending magic page please check that you have entered the right info")
	}

        ctx.Redirect(302, "/")
}


func endServerHandler(ctx *web.Context) {
	logger.Printf("Sending Magic packet")
	sendShutDownPacket()

	ctx.Redirect(302, "/")
}

var templates = template.Must(template.ParseFiles("index.html", "signin.html"))

func getSession(ctx *web.Context, manager *session.SessionManager) *session.Session {
        id, _ := ctx.GetSecureCookie("SessionId")
        session := manager.GetSessionById(id)
        ctx.SetSecureCookie("SessionId", session.Id, int64(manager.GetTimeout()))
        ctx.SetHeader("Pragma", "no-cache", true)
        return session
}

func getParam(ctx *web.Context, name string) string {
        value, found := ctx.Params[name]
        if found {
                return strings.Trim(value, " ")
        }
        return ""
}

func dbSetup() {
        if _, e := os.Stat(dbfile); e != nil {
                db, e := sql.Open("sqlite3", dbfile)
                if e != nil {
                        logger.Print(e)
                        return
                }
                for _, s := range []string {
                        "create table User (userid varchar(16), password varchar(20), realname varchar(20), age integer)",
                        "insert into User values('go', 'lang', 'golang', 3)",
                } {
                        if _, e := db.Exec(s); e != nil {
                                logger.Print(e)
                                return
                        }
                }
                db.Close()
        }
}

func main() {

        //------------------------------------------------
        // initialize session manager
        manager.OnStart(func(session *session.Session) {
                logger.Printf("Start session(\"%s\")", session.Id)
        })
        manager.OnEnd(func(session *session.Session) {
                logger.Printf("End session(\"%s\")", session.Id)
        })
        manager.SetTimeout(28800)


        //------------------------------------------------
        // initialize database
        dbSetup()

        //------------------------------------------------
        // go to web
        web.Config.CookieSecret = "7C19QRmwf3mHZ9CPAaPQ0hsWeufKd"
        s := "select userid, password, realname, age from User where userid = ? and password = ?"

        web.Get("/", func(ctx *web.Context) {
                session := getSession(ctx, manager)
		rootHandler(ctx, session)
        })

	web.Get("/signin", func(ctx *web.Context) {
		session := getSession(ctx, manager)
		signinHandler(ctx, session)
	})

        web.Post("/login", func(ctx *web.Context) {
                session := getSession(ctx, manager)
                userid := getParam(ctx, "userid")
                password := getParam(ctx, "password")
                if userid != "" && password != "" {
                        // find user
                        db, e := sql.Open("sqlite3", dbfile)
                        defer db.Close()
                        st, _ := db.Prepare(s)
                        r, e := st.Query(userid, password)
                        if e != nil {
                                logger.Print(e)
                                return
                        }
                        if !r.Next() {
                                // not found
                                templates.Execute(ctx, map[string]interface{} {
                                        "Value": nil, "Msg": "User not found",
                                })
                                return
                        }
                        var userid, password, realname string
                        var age int64
                        e = r.Scan(&userid, &password, &realname, &age)
                        if e != nil {
                                logger.Print(e)
                                return
                        }
                        // store User object to sessino
                        session.Value = &User{userid, password, realname, age}
                        logger.Printf("User \"%s\" login", session.Value.(*User).UserId)
                }
                ctx.Redirect(302, "/")
        })
        web.Post("/logout", func(ctx *web.Context) {
                session := getSession(ctx, manager)
                if session.Value != nil {
                        // abandon
                        logger.Printf("User \"%s\" logout", session.Value.(*User).UserId)
                        session.Abandon()
                }
                ctx.Redirect(302, "/")
        })

	web.Get("/startServer", func(ctx *web.Context) {
		session := getSession(ctx, manager)
		if session.Value == nil {
			logger.Printf("A theif tried to play with my toys")
			return
		}
		startServerHandler(ctx)
	})

	web.Get("/endServer", func(ctx *web.Context) {
		session := getSession(ctx, manager)
		if session.Value == nil {
			logger.Printf("A theif tried to shutdown my toy")
			return
		}
		endServerHandler(ctx)
	})
//	http.HandleFunc("/", rootHandler)
//	http.HandleFunc("/startServer", startServerHandler)
//	http.HandleFunc("/endServer", endServerHandler)
	go testSshSockUpOnServer()
	web.Run (":8111")

}

