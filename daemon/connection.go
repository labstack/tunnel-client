package daemon

import (
  "bufio"
  "encoding/json"
  "fmt"
  "io"
  "net"
  "net/http"
  "net/url"
  "os"
  "time"

  "github.com/labstack/gommon/log"
  gonanoid "github.com/matoous/go-nanoid"
  "github.com/spf13/viper"

  "golang.org/x/crypto/ssh"
)

type (
  User struct {
    ID            string   `json:"id"`
    Protocol      Protocol `json:"protocol"`
    Target        string   `json:"target"`
    Configuration string   `json:"configuration"`
    Key           string   `json:"key"`
  }

  Configuration struct {
    Name     string   `json:"name"`
    Protocol Protocol `json:"protocol"`
    Prefix   string   `json:"prefix"`
    Hostname string   `json:"hostname"`
    Domain   string   `json:"domain"`
    Port     int      `json:"port"`
  }

  Connection struct {
    server        *Server
    startChan     chan error
    acceptChan    chan net.Conn
    reconnectChan chan error
    stopChan      chan bool
    started       bool
    stopped       bool
    retries       time.Duration
    user          *User
    ID            string `json:"id"`
    Name          string `json:"name"`
    Random        bool   `json:"random"`
    Hostname      string `json:"hostname"`
    Port          int    `json:"port"`
    TargetAddress string `json:"target_address"`
    RemotePort    string
    RemoteURI     string           `json:"remote_uri"`
    Status        ConnectionStatus `json:"status"`
    ConnectedAt   time.Time        `json:"connected_at"`
    Configuration *Configuration   `json:"-"`
  }

  ConnectionStatus string

  Error struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
  }
)

var (
  hostBytes = []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDoSLknvlFrFzroOlh1cqvcIFelHO+Wvj1UZ/p3J9bgsJGiKfh3DmBqEw1DOEwpHJz4zuV375TyjGuHuGZ4I4xztnwauhFplfEvriVHQkIDs6UnGwJVr15XUQX04r0i6mLbJs5KqIZTZuZ9ZGOj7ZWnaA7C07nPHGrERKV2Fm67rPvT6/qFikdWUbCt7KshbzdwwfxUohmv+NI7vw2X6vPU8pDaNEY7vS3YgwD/WlvQx+WDF2+iwLVW8OWWjFuQso6Eg1BSLygfPNhAHoiOWjDkijc8U9LYkUn7qsDCnvJxCoTTNmdECukeHfzrUjTSw72KZoM5KCRV78Wrctai1Qn6yRQz9BOSguxewLfzHtnT43/MLdwFXirJ/Ajquve2NAtYmyGCq5HcvpDAyi7lQ0nFBnrWv5zU3YxrISIpjovVyJjfPx8SCRlYZwVeUq6N2yAxCzJxbElZPtaTSoXBIFtoas2NXnCWPgenBa/2bbLQqfgbN8VQ9RaUISKNuYDIn4+eO72+RxF9THzZeV17pnhTVK88XU4asHot1gXwAt4vEhSjdUBC9KUIkfukI6F4JFxtvuO96octRahdV1Qg0vF+D0+SPy2HxqjgZWgPE2Xh/NmuIXwbE0wkymR2wrgj8Hd4C92keo2NBRh9dD7D2negnVYaYsC+3k/si5HNuCHnHQ== tunnel@labstack.com")

  ConnectionStatusStatusOnline = ConnectionStatus("online")
  ConnectionStatusReconnecting = ConnectionStatus("reconnecting")
)

func (c *Connection) Host() (host string) {
  h := viper.GetString("hostname")
  if c.Configuration.Hostname != "" {
    h = c.Configuration.Hostname
  } else if c.Hostname != "" {
    h = c.Hostname
  }
  return net.JoinHostPort(h, viper.GetString("port"))
}

func (s *Server) newConnection(req *ConnectRequest) (c *Connection, err error) {
  id, _ := gonanoid.Nanoid()
  c = &Connection{
    server:        s,
    startChan:     make(chan error),
    acceptChan:    make(chan net.Conn),
    reconnectChan: make(chan error),
    stopChan:      make(chan bool),
    ID:            id,
    TargetAddress: req.Address,
    Configuration: new(Configuration),
  }

  // Lookup config
  if req.Configuration != "" {
    e := new(Error)
    res, err := s.resty.R().
      SetResult(c.Configuration).
      SetError(e).
      Get("/configurations/" + req.Configuration)
    if err != nil {
      return nil, fmt.Errorf("failed to the find the configuration: %v", err)
    } else if res.IsError() {
      return nil, fmt.Errorf("failed to the find the configuration: %s", e.Message)
    }
    c.Name = req.Configuration
    req.Protocol = c.Configuration.Protocol
  }
  c.RemotePort = viper.GetString("remote_port")
  if req.Protocol != ProtocolHTTP {
    c.RemotePort = "0"
  }

  c.user = &User{
    ID:            id,
    Protocol:      req.Protocol,
    Target:        req.Address,
    Configuration: req.Configuration,
    Key:           viper.GetString("api_key"),
  }
  return
}

func (c *Connection) connect() {
RECONNECT:
  if c.Status == ConnectionStatusReconnecting {
    c.retries++
    if c.retries > 5 {
      log.Errorf("failed to reconnect connection: %s", c.Name)
      return
    }
    time.Sleep(c.retries * c.retries * time.Second)
    log.Warnf("reconnecting connection: name=%s, retry=%d", c.Name, c.retries)
  }
  hostKey, _, _, _, err := ssh.ParseAuthorizedKey(hostBytes)
  if err != nil {
    c.startChan <- fmt.Errorf("failed to parse host key: %v", err)
    return
  }
  user, _ := json.Marshal(c.user)
  config := &ssh.ClientConfig{
    User: string(user),
    Auth: []ssh.AuthMethod{
      ssh.Password("password"),
    },
    HostKeyCallback: ssh.FixedHostKey(hostKey),
  }

  // Connect
  sc := new(ssh.Client)
  proxy := os.Getenv("http_proxy")
  if proxy != "" {
    proxyURL, err := url.Parse(proxy)
    if err != nil {
      c.startChan <- fmt.Errorf("cannot open new session: %v", err)
      return
    }
    tcp, err := net.Dial("tcp", proxyURL.Hostname())
    if err != nil {
      c.startChan <- fmt.Errorf("cannot open new session: %v", err)
      return
    }
    connReq := &http.Request{
      Method: "CONNECT",
      URL:    &url.URL{Path: c.Host()},
      Host:   c.Host(),
      Header: make(http.Header),
    }
    if proxyURL.User != nil {
      if p, ok := proxyURL.User.Password(); ok {
        connReq.SetBasicAuth(proxyURL.User.Username(), p)
      }
    }
    connReq.Write(tcp)
    resp, err := http.ReadResponse(bufio.NewReader(tcp), connReq)
    if err != nil {
      c.startChan <- fmt.Errorf("cannot open new session: %v", err)
      return
    }
    defer resp.Body.Close()

    conn, chans, reqs, err := ssh.NewClientConn(tcp, c.Host(), config)
    if err != nil {
      c.startChan <- fmt.Errorf("cannot open new session: %v", err)
      return
    }
    sc = ssh.NewClient(conn, chans, reqs)
  } else {
    sc, err = ssh.Dial("tcp", c.Host(), config)
  }
  if err != nil {
    log.Error(err)
    c.Status = ConnectionStatusReconnecting
    goto RECONNECT
  }

  // Close
  defer func() {
    log.Infof("closing connection: %s", c.Name)
    delete(connections, c.ID)
    defer sc.Close()
  }()

  // Remote listener
  l, err := sc.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", c.RemotePort))
  if err != nil {
    c.startChan <- fmt.Errorf("failed to listen on remote host: %v", err)
    return
  }

  if err = c.server.findConnection(c); err != nil {
    c.startChan <- fmt.Errorf("failed to find connection: %v", err)
    return
  }
  // Note: Don't close the listener as it prevents closing the underlying connection
  c.retries = 0
  if !c.started {
    c.started = true
    c.startChan <- nil
  }
  connections[c.ID] = c
  log.Infof("connection %s is online", c.Name)

  // Accept connections
  go func() {
    for {
      in, err := l.Accept()
      if err != nil && !c.stopped {
        log.Error(err)
        c.reconnectChan <- err
        return
      }
      c.acceptChan <- in
    }
  }()

  // Listen events
  for {
    select {
    case <-c.stopChan:
      c.stopped = true
      return
    case in := <-c.acceptChan:
      go c.handle(in)
    case err = <-c.reconnectChan:
      c.Status = ConnectionStatusReconnecting
      goto RECONNECT
    }
  }
}

func (c *Connection) handle(in net.Conn) {
  defer in.Close()

  // Target connection
  out, err := net.Dial("tcp", c.TargetAddress)
  if err != nil {
    log.Printf("failed to connect to target: %v", err)
    return
  }
  defer out.Close()

  // Copy
  errCh := make(chan error, 2)
  cp := func(dst io.Writer, src io.Reader) {
    _, err := io.Copy(dst, src)
    errCh <- err
  }
  go cp(in, out)
  go cp(out, in)

  // Handle error
  err = <-errCh
  if err != nil && err != io.EOF {
    log.Printf("failed to copy: %v", err)
  }
}

func (c *Connection) stop() {
  log.Warnf("stopping connection: %s", c.Name)
  c.stopChan <- true
}
