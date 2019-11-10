package daemon

import (
  "errors"
  "fmt"
  "github.com/fsnotify/fsnotify"
  "github.com/labstack/gommon/log"
  "github.com/spf13/viper"
  "golang.org/x/sync/errgroup"
  "gopkg.in/resty.v1"
  "io/ioutil"
  "net"
  "net/rpc"
  "os"
  "os/signal"
  "syscall"
)

type (
  Server struct {
    resty       *resty.Client
    Connections map[string]*Connection
  }

  Protocol string

  ConnectRequest struct {
    Name     string
    Address  string
    Protocol Protocol
  }

  ConnectReply struct {
  }

  PSRequest struct {
  }

  PSReply struct {
    Connections []*Connection
  }

  RMRequest struct {
    Name  string
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
  go c.connect()
  <-c.startChan
  return
}

func (s *Server) PS(req *PSRequest, rep *PSReply) (err error) {
  for _, c := range s.Connections {
    rep.Connections = append(rep.Connections, c)
  }
  return nil
}

func (s *Server) RM(req *RMRequest, rep *RMReply) error {
  if c, ok := s.Connections[req.Name]; ok {
    c.stop()
    return c.delete()
  }
  return nil
}

func (s *Server) stopConnections() error {
  g := new(errgroup.Group)
  for _, c := range s.Connections {
    c := c
    g.Go(func() error {
      c.stop()
      return c.delete()
    })
  }
  return g.Wait()
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

  // Shutdown hook
  c := make(chan os.Signal)
  signal.Notify(c, os.Interrupt, syscall.SIGTERM)
  go func() {
    <-c
    if err := s.stopConnections(); err != nil {
      log.Error("failed stopping connections: %v", err)
    }
  }()

  // Cleanup
  if err := s.deleteAll(); err != nil {
    log.Error(err)
  }

  // Listen
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
  return s.stopConnections()
}

func (s *Server) findConnection(c *Connection) (err error) {
  e := new(Error)
  res, err := s.resty.R().
    SetResult(c).
    SetError(e).
    Get("/connections/" + c.ID)
  if err != nil {
    return
  }
  if res.IsError() {
    return errors.New(e.Message)
  }
  s.Connections[c.Name] = c
  return
}

func (s *Server) deleteAll() (err error) {
  log.Warnf("removing all connections")
  e := new(Error)
  res, err := s.resty.R().
    SetError(e).
    Delete("/connections")
  if err != nil {
    return
  }
  if res.IsError() {
    return fmt.Errorf("failed to delete all connections")
  }
  return
}
