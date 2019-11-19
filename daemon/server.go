package daemon

import (
  "errors"
  "github.com/fsnotify/fsnotify"
  "github.com/labstack/gommon/log"
  "github.com/spf13/viper"
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
    resty *resty.Client
  }

  Protocol string

  ConnectRequest struct {
    Configuration string
    Address       string
    Protocol      Protocol
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
)

const (
  ProtocolHTTP = Protocol("http")
  ProtocolTCP  = Protocol("tcp")
  ProtocolTLS  = Protocol("tls")
)

var (
  connections = map[string]*Connection{}
)

func (s *Server) Connect(req *ConnectRequest, rep *ConnectReply) (err error) {
  c, err := s.newConnection(req)
  if err != nil {
    return
  }
  go c.connect()
  return <-c.startChan
}

func (s *Server) PS(req *PSRequest, rep *PSReply) (err error) {
  for _, c := range connections {
    rep.Connections = append(rep.Connections, c)
  }
  return nil
}

func (s *Server) RM(req *RMRequest, rep *RMReply) error {
  for _, c := range connections {
    if c.Name == req.Name {
      c.stop()
    }
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
  s := &Server{resty: r}
  rpc.Register(s)

  // Shutdown hook
  c := make(chan os.Signal)
  signal.Notify(c, os.Interrupt, syscall.SIGTERM)
  go func() {
    <-c
    log.Warn("stopping daemon")
    os.Exit(0)
  }()

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
  return
}
