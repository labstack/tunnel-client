package daemon

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/labstack/gommon/log"
	"github.com/spf13/viper"
	"gopkg.in/resty.v1"
	"io/ioutil"
	"net"
	"net/rpc"
	"os"
)

type (
	Server struct {
		resty       *resty.Client
		Connections map[string]*Connection
	}

	Protocol string

	ConnectRequest struct {
		Configuration string
		Address       string
		Protocol      Protocol
	}

	ConnectReply struct {
	}

	StartRequest struct {
		ID string
	}

	StartReply struct {
	}

	StopRequest struct {
		ID string
	}

	StopReply struct {
	}

	PSRequest struct {
	}

	PSReply struct {
		Connections []*Connection
	}

	RMRequest struct {
		ID    string
		Force bool
	}

	RMReply struct {
	}

	StopDaemonRequest struct {
	}

	StopDaemonReply struct {
	}
)

const (
	ProtocolHTTPS = "https"
	ProtocolTCP   = "tcp"
	ProtocolTLS   = "tls"
)

func (s *Server) Connect(req *ConnectRequest, rep *ConnectReply) (err error) {
	c, err := s.newConnection(req)
	if err != nil {
		return
	}
	go c.start()
	select {
	case <-c.startChan:
	case err = <-c.errorChan:
	}
	return
}

func (s *Server) Start(req *StartRequest, rep *StartReply) (err error) {
	if c, ok := s.Connections[req.ID]; ok {
		go c.start()
		select {
		case <-c.startChan:
		case err = <-c.errorChan:
		}
	}
	return
}

func (s *Server) Stop(req *StopRequest, rep *StopReply) error {
	if c, ok := s.Connections[req.ID]; ok {
		c.stop()
	}
	return nil
}

func (s *Server) PS(req *PSRequest, rep *PSReply) error {
	for _, c := range s.Connections {
		rep.Connections = append(rep.Connections, c)
	}
	return nil
}

func (s *Server) RM(req *RMRequest, rep *RMReply) error {
	if c, ok := s.Connections[req.ID]; ok {
		if c.Status == ConnectionStatusStatusOffline ||
			c.Status == ConnectionStatusStatusOnline && req.Force {
			c.stop()
			return c.delete()
		}
		return fmt.Errorf("cannot remove an online connection %s, to force remove use `-f`", c.ID)
	}
	return nil
}

func Start() {
	log.Info("starting daemon")
	r := resty.New()
	r.SetHostURL(viper.GetString("api_url"))
	r.SetAuthToken(viper.GetString("api_key"))
	viper.OnConfigChange(func(e fsnotify.Event) {
		r.SetAuthToken(viper.GetString("api_key"))
	})
	r.SetHeader("Content-Type", "application/json")
	r.SetHeader("User-Agent", "tunnel/client")
	s := &Server{
		resty:       r,
		Connections: map[string]*Connection{},
	}
	rpc.Register(s)
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		log.Fatal(e)
	}
	err := ioutil.WriteFile(viper.GetString("daemon_addr"), []byte(l.Addr().String()), 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	rpc.Accept(l)
}

func (s *Server) StopDaemon(req *StopDaemonRequest, rep *StopDaemonReply) (err error) {
	log.Warn("stopping daemon")
	for _, c := range s.Connections {
		go func(c *Connection) {
			c.stop()
			if err = c.delete(); err != nil {
				return
			}
		}(c)
	}
	pid := viper.GetInt("daemon_pid")
	p, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	p.Kill()
	os.Remove(viper.GetString("daemon_pid"))
	return os.Remove(viper.GetString("daemon_addr"))
}
