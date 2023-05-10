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

// CheckUserClickHouse - Check user pass with "SELECT 1" query;
// return true if credentials accepted by CH
func (server *Server) CHCheckCredentialsUser(user string, pass string) bool {
	s := "SELECT 1"
	qs := "user=" + user + "&password=" + pass
	_, status, _ := server.Collector.Sender.SendQuery(&ClickhouseRequest{Params: qs, Content: s, isInsert: false})
	if status == http.StatusOK {
		return true
	} else {
		return false
	}
}

func (s *Server) ChanelCHCredentials(period time.Duration) {
	t := time.NewTicker(period * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			s.CHCheckCredentialsAll()
		}
	}
}

// CHCheckCredentialsAll - check all users in Credential map with CH, update CreditTime, set metric with active departments
func (s *Server) CHCheckCredentialsAll() {
	c := s.Collector
	if s.Debug {
		log.Printf("Checking credentials for all (%+v) users in map Credentials", len(c.Credentials))
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for user := range c.Credentials {
		credential := c.Credentials[user]
		if credential.CreditTime.After(time.Now()) {
			if s.CHCheckCredentialsUser(user, credential.Account.Pass) {
				credential.CreditTime = AddTime(c.CredentialInt)
			} else {
				c.addBlacklist(credential.Account.Login, credential.Account.Pass, c.CredentialInt)
			}
		}
	}
}

// AdminWriteHandler - implemtn querys from admin users;
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

// UserActions user with CH, returns ImplementUserQuery if OK or 401 error if not
func (server *Server) UserActions(user string, pass string, c echo.Context, s string, qs string) error {
	if server.CHCheckCredentialsUser(user, pass) {
		server.Collector.addCredential(user, pass)
		return server.ImplementUserQuery(c, s, qs, user, pass)
	} else {
		server.Collector.addBlacklist(user, pass, server.Collector.CredentialInt)
		return c.String(http.StatusUnauthorized, "")
	}
}

// UserWriteHandler - implement querys from users;
func (server *Server) UserWriteHandler(c echo.Context, s string, qs string, user string, pass string) error {
	collector := server.Collector
	if collector.CredentialExist(user) {
		if collector.PasswordMatchCredential(user, pass) {
			return server.ImplementUserQuery(c, s, qs, user, pass)
		} else {
			return c.String(http.StatusUnauthorized, "")
		}
	} else {
		credit := Account{user, pass}
		if collector.BlackListExist(credit) {
			if collector.BlackListTimeEnded(credit) {
				return server.UserActions(user, pass, c, s, qs)
			} else {
				return c.String(http.StatusForbidden, "")
			}
		} else {
			return server.UserActions(user, pass, c, s, qs)
		}
	}
}

// ImplementUserQuery - implemtn querys from ordinary users;
// login and password changed with dmicp
func (server *Server) ImplementUserQuery(c echo.Context, s string, qs string, user string, pass string) error {
	dmicp := server.Collector.Dmicp
	if qs == "" {
		qs = "user=" + dmicp.Login + "&password=" + dmicp.Pass
	} else {
		qs = "user=" + dmicp.Login + "&password=" + dmicp.Pass + "&" + qs
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
	} else if !strings.HasPrefix(s, "SELECT count() FROM system.databases") &&
		strings.Contains(s, "SELECT") && strings.Contains(s, "FROM") {
		log.Printf("DEBUG: User [%+v] without admin credentials try to SELECT\n", user)
		return c.String(http.StatusForbidden, "")
	} else {
		resp, status, _ := server.Collector.Sender.SendQuery(&ClickhouseRequest{Params: qs, Content: s, isInsert: false})
		return c.String(status, resp)
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
	collector := server.Collector
	if ok {
		role := collector.identifyRole(user)
		qs := c.QueryString()
		switch role {
		case Dmicp:
			log.Printf("Direct connection with dmicp_login forbidden")
			return c.String(http.StatusForbidden, "")
		case Admin:
			return server.AdminWriteHandler(c, s, qs, user, pass)
		case User:
			return server.UserWriteHandler(c, s, qs, user, pass)
		}
	}
	return c.String(http.StatusBadRequest, "Authentication failed because of bad request")
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
	InitMetrics(cnf)
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

	collect := NewCollector(sender, cnf)

	// send collected data on SIGTERM and exit
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	srv := InitServer(cnf.Listen, collect, cnf.Debug)
	srv.echo.Group("/play*", middleware.Proxy(middleware.NewRoundRobinBalancer(targets_)))

	//credential updating
	go func() {
		for {
			ctxCHCredentialsChecker := context.Background()
			ctxCHCredentialsChecker, CHCredentialsCancel := context.WithCancel(ctxCHCredentialsChecker)
			defer CHCredentialsCancel()
			go srv.ChanelCHCredentials(time.Duration(cnf.CredInterval))
			select {
			case <-ctxCHCredentialsChecker.Done():
				log.Printf("INFO: stop using Blacklist")
			}
		}
	}()

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
