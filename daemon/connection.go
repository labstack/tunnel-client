package daemon

import (
	"bufio"
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/spf13/viper"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

type (
	Configuration struct {
		Name     string   `json:"name"`
		Protocol Protocol `json:"protocol"`
		Prefix   string   `json:"prefix"`
		Hostname string   `json:"hostname"`
		Domain   string   `json:"domain"`
		Port     int      `json:"port"`
	}

	Connection struct {
		server            *Server
		startChan         chan bool
		acceptChan        chan net.Conn
		reconnectChan     chan error
		stopChan          chan bool
		errorChan         chan error
		retries           time.Duration
		ID                string `json:"id"`
		User              string
		Host              string
		TargetAddress     string `json:"target_address"`
		RemoteHost        string
		RemotePort        int
		RemoteURI         string           `json:"remote_uri"`
		Status            ConnectionStatus `json:"status"`
		CreatedAt         time.Time        `json:"created_at"`
		UpdatedAt         time.Time        `json:"updated_at"`
		ConfigurationName string           `json:"configuration_name"`
		Configuration     *Configuration   `json:"-"`
	}

	ConnectionStatus string

	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
)

var (
	hostBytes = []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDoSLknvlFrFzroOlh1cqvcIFelHO+Wvj1UZ/p3J9bgsJGiKfh3DmBqEw1DOEwpHJz4zuV375TyjGuHuGZ4I4xztnwauhFplfEvriVHQkIDs6UnGwJVr15XUQX04r0i6mLbJs5KqIZTZuZ9ZGOj7ZWnaA7C07nPHGrERKV2Fm67rPvT6/qFikdWUbCt7KshbzdwwfxUohmv+NI7vw2X6vPU8pDaNEY7vS3YgwD/WlvQx+WDF2+iwLVW8OWWjFuQso6Eg1BSLygfPNhAHoiOWjDkijc8U9LYkUn7qsDCnvJxCoTTNmdECukeHfzrUjTSw72KZoM5KCRV78Wrctai1Qn6yRQz9BOSguxewLfzHtnT43/MLdwFXirJ/Ajquve2NAtYmyGCq5HcvpDAyi7lQ0nFBnrWv5zU3YxrISIpjovVyJjfPx8SCRlYZwVeUq6N2yAxCzJxbElZPtaTSoXBIFtoas2NXnCWPgenBa/2bbLQqfgbN8VQ9RaUISKNuYDIn4+eO72+RxF9THzZeV17pnhTVK88XU4asHot1gXwAt4vEhSjdUBC9KUIkfukI6F4JFxtvuO96octRahdV1Qg0vF+D0+SPy2HxqjgZWgPE2Xh/NmuIXwbE0wkymR2wrgj8Hd4C92keo2NBRh9dD7D2negnVYaYsC+3k/si5HNuCHnHQ== tunnel@labstack.com")

	ConnectionStatusStatusOnline  = ConnectionStatus("online")
	ConnectionStatusStatusOffline = ConnectionStatus("offline")
	ConnectionStatusReconnecting  = ConnectionStatus("reconnecting")
)

func (s *Server) newConnection(req *ConnectRequest) (c *Connection, err error) {
	c = &Connection{
		server:        s,
		startChan:     make(chan bool),
		acceptChan:    make(chan net.Conn),
		reconnectChan: make(chan error),
		stopChan:      make(chan bool),
		errorChan:     make(chan error),
		Host:          viper.GetString("host"),
		RemoteHost:    "0.0.0.0",
		RemotePort:    80,
		Configuration: &Configuration{
			Protocol: req.Protocol,
		},
	}
	e := new(Error)

	if req.Configuration != "" {
		key := viper.GetString("api_key")
		res, err := s.resty.R().
			SetResult(c.Configuration).
			SetError(e).
			Get("/configurations/" + req.Configuration)
		if err != nil {
			return nil, fmt.Errorf("failed to the find the configuration: %v", err)
		} else if res.IsError() {
			return nil, fmt.Errorf("failed to the find the configuration: %s", e.Message)
		}
		c.User = fmt.Sprintf("key=%s,name=%s", key, req.Configuration)
		c.Host = net.JoinHostPort(c.Configuration.Hostname, "22")
	} else {
		if req.Protocol == ProtocolTLS {
			c.User = "tls=true"
		}
	}
	c.TargetAddress = req.Address
	if c.Configuration.Protocol != ProtocolHTTPS {
		c.RemotePort = 0
	}
	return
}

func (c *Connection) start() {
RECONNECT:
	if c.Status == ConnectionStatusReconnecting {
		c.retries++
		if c.retries > 5 {
			c.errorChan <- fmt.Errorf("failed to reconnect connection id=%s", c.ID)
			return
		}
		time.Sleep(c.retries * c.retries * time.Second)
		if err := c.update(); err != nil {
			c.errorChan <- err
			return
		}
		log.Warnf("reconnecting connection: id=%s, retry=%d", c.ID, c.retries)
	}
	hostKey, _, _, _, err := ssh.ParseAuthorizedKey(hostBytes)
	if err != nil {
		log.Fatalf("failed to parse host key: %v", err)
	}
	config := &ssh.ClientConfig{
		User: c.User,
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
			log.Fatalf("cannot open new session: %v", err)
		}
		tcp, err := net.Dial("tcp", proxyURL.Hostname())
		if err != nil {
			log.Fatalf("cannot open new session: %v", err)
		}
		connReq := &http.Request{
			Method: "CONNECT",
			URL:    &url.URL{Path: c.Host},
			Host:   c.Host,
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
			log.Fatalf("cannot open new session: %v", err)
		}
		defer resp.Body.Close()

		conn, chans, reqs, err := ssh.NewClientConn(tcp, c.Host, config)
		if err != nil {
			log.Fatalf("cannot open new session: %v", err)
		}
		sc = ssh.NewClient(conn, chans, reqs)
	} else {
		sc, err = ssh.Dial("tcp", c.Host, config)
	}
	if err != nil {
		log.Error(err)
		c.Status = ConnectionStatusReconnecting
		goto RECONNECT
	}

	// Session
	sess, err := sc.NewSession()
	if err != nil {
		log.Fatalf("failed to create session: %v", err)
	}
	r, err := sess.StdoutPipe()
	if err != nil {
		log.Print(err)
	}
	br := bufio.NewReader(r)
	//w, err := sess.StdinPipe()
	//if err != nil {
	//	log.Print(err)
	//}
	//bw := bufio.NewWriter(w)

	go func() {
		for {
			line, _, err := br.ReadLine()
			if err != nil {
				if err == io.EOF {
					return
				} else {
					log.Fatalf("failed to read: %v", err)
				}
			}
			// TODO: Use proper message format with type, header & body (e.g. User)
			c.RemoteURI = string(line)
			c.Status = ConnectionStatusStatusOnline
			if err := c.create(); err != nil {
				c.errorChan <- err
				return
			}
			c.retries = 0
			c.startChan <- true
			c.server.Connections[c.ID] = c
		}
	}()

	// Remote listener
	l, err := sc.Listen("tcp", fmt.Sprintf("%s:%d", c.RemoteHost, c.RemotePort))
	if err != nil {
		log.Fatalf("failed to listen on remote host: %v", err)
	}
	// Note: Don't close the listener as it prevents closing the underlying connection

	// Close
	defer func() {
		log.Infof("closing connection %s", c.ID)
		defer sess.Close()
		defer sc.Close()
		c.Status = ConnectionStatusStatusOffline
		if err := c.update(); err != nil {
			log.Error(err)
		}
	}()
	go func() {
		in, err := l.Accept()
		if err != nil {
			c.reconnectChan <- err
		} else {
			c.acceptChan <- in
		}
	}()
	select {
	case <-c.stopChan:
		return
	case in := <-c.acceptChan:
		go c.handle(in)
	case err = <-c.reconnectChan:
		log.Error(err)
		c.Status = ConnectionStatusReconnecting
		goto RECONNECT
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
	if c.Status == ConnectionStatusStatusOnline {
		log.Warnf("stopping connection %s", c.ID)
		c.stopChan <- true
	}
}

func (c *Connection) create() error {
	if c.ID != "" {
		return c.update()
	}
	e := new(Error)
	res, err := c.server.resty.R().
		SetBody(c).
		SetResult(c).
		SetError(e).
		Post("/connections")
	if err != nil {
		return err
	} else if res.IsError() {
		return fmt.Errorf("failed to create a connection: %s", e.Message)
	}
	return nil
}

func (c *Connection) update() (err error) {
	if c.ID == "" {
		return
	}
	e := new(Error)
	res, err := c.server.resty.R().
		SetBody(c).
		SetResult(c).
		SetError(e).
		Put("/connections/" + c.ID)
	if err != nil {
		return
	} else if res.IsError() {
		return fmt.Errorf("failed to update the connection: id=%s, error=%s", c.ID, e.Message)
	}
	return
}

func (c *Connection) delete() error {
	log.Warnf("removing connection %s", c.ID)
	e := new(Error)
	res, err := c.server.resty.R().
		SetError(e).
		Delete("/connections/" + c.ID)
	if err != nil {
		return err
	} else if res.IsError() {
		return fmt.Errorf("failed to delete the connection: %s", e.Message)
	}
	delete(c.server.Connections, c.ID)
	return nil
}
