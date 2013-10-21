// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"text/template"
	"io/ioutil"
	"net"
	"time"
        "database/sql"
	"github.com/ghthor/gowol"
        "github.com/hoisie/web"
        "github.com/mattn/go-session-manager"
        _ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"strings"
	"encoding/xml"
	"fmt"
	"net/http/httputil"
	"net/url"
)

type Page struct {
	Title string
	Body  []byte
	Status string
}

type Settings struct {
	XMLName xml.Name `xml:"Config"`
	ServAddr string `xml:"ServAddr"`
	MacAddr string `xml:"MacAddr"`
	UserName string `xml:"UserName"`
	Password string `xml:"Password"`
	FullName string `xml:FullName"`
}

var logger = log.New(os.Stdout, "", log.Ldate|log.Ltime)
var manager = session.NewSessionManager(logger)
var currentStatus = "Unknown"
var config = Settings{}

const bcAddr = "255.255.255.255"
const dbfile = "./user.db"

type User struct {
        UserId string
        Password string
        RealName string
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
		println("Dial failed:", err.Error())
		return
	}

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
		tcpAddr, err := net.ResolveTCPAddr("tcp", config.ServAddr)
		if err != nil {
			currentStatus = "Offline"
			println("ResolveTcpAddr failed ")
			continue
		}

		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			currentStatus = "Offline"
			println("Dial failed:", err.Error())
			continue
		}

		currentStatus = "Online"
		conn.Close()
	}
}

func sendMagicPacket() {
	err := wol.SendMagicPacket(config.MacAddr, bcAddr)
	if err != nil {
		println("An error has occured sending magic page please check ")

	}
}

func rootHandler(ctx *web.Context, session *session.Session) {
	userString := "N/A"
	if session.Value != nil {
		userString = session.Value.(*User).RealName
	}

	templates.ExecuteTemplate(ctx, "index.html", map[string]interface{} {
		"Value" : session.Value, "Msg": "", "Title" : "Home", "Status" : currentStatus, "User" : userString,
	})
}

func signinHandler(ctx *web.Context, session *session.Session) {
	templates.ExecuteTemplate(ctx, "signin.html", map[string]interface{} {
		"Value" : session.Value, "Msg": "",
	})
}


func startServerHandler(ctx *web.Context) {
	err := wol.SendMagicPacket(config.MacAddr, bcAddr)
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
		insertString := fmt.Sprintf("insert into User values('%s', '%s', '%s')",config.UserName, config.Password, config.FullName )
		logger.Printf("insertString %s",insertString)
                for _, s := range []string {
                        "create table User (userid varchar(16), password varchar(20), realname varchar(20))",
                        insertString,
                } {
                        if _, e := db.Exec(s); e != nil {
                                logger.Print(e)
                                return
                        }
                }
                db.Close()
        }
}

func loadGlobalSettings() {
	cfg, err := ioutil.ReadFile("config.xml")
	if err == nil {
		xml.Unmarshal(cfg, &config)
	} else {
		logger.Printf("An error has occured")
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
	loadGlobalSettings()
        dbSetup()

        //------------------------------------------------
        // go to web
        web.Config.CookieSecret = "7C19QRmwf3mHZ9CPAaPQ0hsWeufKd"
        s := "select userid, password, realname from User where userid = ? and password = ?"

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
                        e = r.Scan(&userid, &password, &realname)
                        if e != nil {
                                logger.Print(e)
                                return
                        }
                        // store User object to sessino
                        session.Value = &User{userid, password, realname}
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

	servUrl, err := url.Parse("http://dragon/mediawiki")
	if err != nil {
		return
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(servUrl)
	web.Get("/mediawiki",  reverseProxy)

	go testSshSockUpOnServer()
	web.Run (":8111")

}

