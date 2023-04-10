package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	// debug stuff
	_ "net/http/pprof"
	"runtime"
	"runtime/debug"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server - main server object
type Server struct {
	Listen    string
	Collector *Collector
	Debug     bool
	echo      *echo.Echo
}

// Status - response status struct
type Status struct {
	Status    string                       `json:"status"`
	SendQueue int                          `json:"send_queue,omitempty"`
	Servers   map[string]*ClickhouseServer `json:"servers,omitempty"`
	Tables    map[string]*Table            `json:"tables,omitempty"`
}

// NewServer - create server
func NewServer(listen string, collector *Collector, debug bool) *Server {
	return &Server{listen, collector, debug, echo.New()}
}

func (server *Server) CheckUserClickHouse(c echo.Context, user string, pass string) bool {
	qs := c.QueryString()
	s := "SELECT timezone()"
	qs = "user=" + user + "&password=" + pass
	resp, status, _ := server.Collector.Sender.SendQuery(&ClickhouseRequest{Params: qs, Content: s, isInsert: false})

	if status != 200 {
		log.Printf("INFO:[%+v]", resp)
		if server.Debug {
			log.Printf("DEBUG: CheckUserClickHouse can`t add user [%+v]\n", user)
		}
		return false
	}
	if server.Debug {
		log.Printf("DEBUG: CheckUserClickHouse user [%+v] successfully added \n", user)
	}
	return true
}

func (server *Server) CheckCredentialsUser(c echo.Context, user string, pass string) bool {
	credential, exist := server.Collector.Credentials[user]
	if exist && credential.CreditTime.Before(time.Now()) ||
		server.CheckUserClickHouse(c, user, pass) {
		if server.Debug {
			log.Printf("DEBUG: CheckCredentialsUser user [%+v] already exist in Credential\n", user)
		}
		return true
	} else {
		if server.Debug {
			log.Printf("DEBUG: CheckCredentialsUser user [%+v] NOT in Credential\n", user)
		}
		return false
	}
}

// Эта функция не учитывает, что админ тоже может делать insert, которые необходимо сложить в общую таблицу (надо добавить такую функцию)
func (server *Server) AdminWriteHandler(c echo.Context, s string, qs string, user string, pass string) error {
	if server.Debug {
		log.Printf("DEBUG: AdminWriteHandler\n")
	}
	if qs == "" {
		qs = "user=" + user + "&password=" + pass
	} else {
		qs = "user=" + user + "&password=" + pass + "&" + qs
	}
	resp, status, _ := server.Collector.Sender.SendQuery(&ClickhouseRequest{Params: qs, Content: s, isInsert: false})
	return c.String(status, resp)
}

func (server *Server) UserWriteHandler(c echo.Context, s string, qs string, user string, pass string) error {
	if qs == "" {
		qs = "user=" + "departmentdmicp" + "&password=" + "passdmicp"
	} else {
		qs = "user=" + "departmentdmicp" + "&password=" + "passdmicp" + "&" + qs
	}
	params, content, insert := server.Collector.ParseQuery(qs, s)
	if insert && !strings.Contains(s, "SELECT") {
		if server.Debug {
			log.Printf("DEBUG: UserWriteHandler find INSERT in query\n")
		}
		if len(content) == 0 {
			log.Printf("INFO: empty insert params: [%+v] content: [%+v]\n", params, content)
			return c.String(http.StatusInternalServerError, "Empty insert\n")
		}
		go server.Collector.Push(params, content)
		if server.Debug {
			log.Printf("DEBUG: UserWriteHandler pushed content:[%+v] with params: [%+v]\n", content, params)
		}
		return c.String(http.StatusOK, "")
	} else if strings.HasPrefix(s, "SELECT count() FROM system.databases") ||
		strings.HasPrefix(s, "SELECT version()") ||
		strings.HasPrefix(s, "SELECT timezone()") {
		resp, status, _ := server.Collector.Sender.SendQuery(&ClickhouseRequest{Params: qs, Content: s, isInsert: false})
		if server.Debug {
			log.Printf("DEBUG: UserWriteHandler find SELECT version/ timezone/ ... from system.database in query\n")
		}
		return c.String(status, resp)
	} else {
		if server.Debug {
			log.Printf("DEBUG: UserWriteHandler set blackListCredential for user: [%+v]\n", user)
		}
		server.Collector.blackListCredential(user)
		return c.String(http.StatusOK, "")
	}
}

func (server *Server) writeHandler(c echo.Context) error {
	if server.Debug {
		log.Printf("DEBUG: writeHandler: Tables count: [%+v]\n", len(server.Collector.Tables))
	}
	q, _ := ioutil.ReadAll(c.Request().Body)
	s := string(q)
	user, pass, ok := c.Request().BasicAuth()
	if server.Debug {
		log.Printf("DEBUG: query %+v %+v\n", c.QueryString(), s)
	}
	if ok {
		role := server.Collector.Role(user)
		qs := c.QueryString()
		if role == "admin" {
			return server.AdminWriteHandler(c, s, qs, user, pass)
		} else if role == "normal" {
			if server.CheckCredentialsUser(c, user, pass) {
				return server.UserWriteHandler(c, s, qs, user, pass)
			} else {
				return c.String(401, "There is no user with such name or password is incorrect")
			}
		} else {
			return c.String(403, "User in blacklist")
		}
	} else {
		return c.String(400, "Authentication failed because of bad request")
	}
}

func (server *Server) statusHandler(c echo.Context) error {
	return c.JSON(200, Status{Status: "ok"})
}

func (server *Server) gcHandler(c echo.Context) error {
	runtime.GC()
	return c.JSON(200, Status{Status: "GC"})
}

func (server *Server) freeMemHandler(c echo.Context) error {
	debug.FreeOSMemory()
	return c.JSON(200, Status{Status: "freeMem"})
}

// manual trigger for cleaning tables
func (server *Server) tablesCleanHandler(c echo.Context) error {
	log.Printf("DEBUG: clean tables:\n%+v", server.Collector.Tables)
	for k, t := range server.Collector.Tables {
		log.Printf("DEBUG: check if table is empty: %+v with key:%+v\n", t, k)
		if ok := t.Empty(); ok {
			log.Printf("DEBUG: delete empty table: %+v with key:%+v\n", t, k)
			server.Collector.Tables[k].CleanTable()
			defer delete(server.Collector.Tables, k)
		}
	}
	return c.JSON(200, Status{Status: "cleaned empty tables"})
}

// Start - start http server
func (server *Server) Start() error {
	return server.echo.Start(server.Listen)
}

// Shutdown - stop http server
func (server *Server) Shutdown(ctx context.Context) error {
	return server.echo.Shutdown(ctx)
}

// InitServer - run server
func InitServer(listen string, collector *Collector, debug bool) *Server {
	server := NewServer(listen, collector, debug)

	// server.echo.Group("/play/*", middleware.Proxy(middleware.NewRoundRobinBalancer(targets)))
	server.echo.POST("/", server.writeHandler)
	server.echo.GET("/status", server.statusHandler)
	server.echo.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	// debug stuff
	server.echo.GET("/debug/gc", server.gcHandler)
	server.echo.GET("/debug/freemem", server.freeMemHandler)
	server.echo.GET("/debug/pprof/*", echo.WrapHandler(http.DefaultServeMux))
	server.echo.GET("/debug/tables-clean", server.tablesCleanHandler)

	return server
}

// SafeQuit - safe prepare to quit
func SafeQuit(collect *Collector, sender Sender) {
	collect.FlushAll()
	if count := sender.Len(); count > 0 {
		log.Printf("Sending %+v tables\n", count)
	}
	for !sender.Empty() && !collect.Empty() {
		collect.WaitFlush()
	}
	collect.WaitFlush()
}

// RunServer - run all
func RunServer(cnf Config) {
	InitMetrics()
	dumper := NewDumper(cnf.DumpDir)
	sender := NewClickhouse(cnf.Clickhouse.DownTimeout, cnf.Clickhouse.ConnectTimeout, cnf.Clickhouse.tlsServerName, cnf.Clickhouse.tlsSkipVerify)
	sender.Dumper = dumper
	targets_ := make([]*middleware.ProxyTarget, 0)
	for _, url_ := range cnf.Clickhouse.Servers {
		sender.AddServer(url_)
		url__, _ := url.Parse(url_)
		pt_ := middleware.ProxyTarget{URL: url__}
		targets_ = append(targets_, &pt_)
	}

	collect := NewCollector(sender, cnf.FlushCount, cnf.FlushInterval, cnf.CleanInterval, cnf.RemoveQueryID)

	// send collected data on SIGTERM and exit
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	srv := InitServer(cnf.Listen, collect, cnf.Debug)
	srv.echo.Group("/play*", middleware.Proxy(middleware.NewRoundRobinBalancer(targets_)))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() {
		for {
			_ = <-signals
			log.Printf("STOP signal\n")
			if err := srv.Shutdown(ctx); err != nil {
				log.Printf("Shutdown error %+v\n", err)
				SafeQuit(collect, sender)
				os.Exit(1)
			}
		}
	}()

	if cnf.DumpCheckInterval >= 0 {
		dumper.Listen(sender, cnf.DumpCheckInterval)
	}

	err := srv.Start()
	if err != nil {
		log.Printf("ListenAndServe: %+v\n", err)
		SafeQuit(collect, sender)
		os.Exit(1)
	}
}
