package daemon

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/labstack/gommon/log"
	"github.com/labstack/gommon/random"
	"github.com/spf13/viper"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type (
	Tunnel struct {
		startChan     chan bool
		acceptChan    chan net.Conn
		reconnectChan chan error
		stopChan      chan bool
		reconnectWait time.Duration
		Name          string   `json:"name"`
		Protocol      Protocol `json:"protocol"`
		Subdomain     string   `json:"subdomain"`
		Domain        string   `json:"domain"`
		Port          int      `json:"port"`
		Host          string   `json:"host"`
		User          string
		RemoteHost    string
		RemotePort    int
		RemoteURI     string
		TargetAddress string
		TargetHost    string
		TargetPort    int
		Status        string
		CreatedAt     time.Time
		HideBanner    bool
	}

	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
)

var (
	tunnels   = map[string]*Tunnel{}
	hostBytes = []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDoSLknvlFrFzroOlh1cqvcIFelHO+Wvj1UZ/p3J9bgsJGiKfh3DmBqEw1DOEwpHJz4zuV375TyjGuHuGZ4I4xztnwauhFplfEvriVHQkIDs6UnGwJVr15XUQX04r0i6mLbJs5KqIZTZuZ9ZGOj7ZWnaA7C07nPHGrERKV2Fm67rPvT6/qFikdWUbCt7KshbzdwwfxUohmv+NI7vw2X6vPU8pDaNEY7vS3YgwD/WlvQx+WDF2+iwLVW8OWWjFuQso6Eg1BSLygfPNhAHoiOWjDkijc8U9LYkUn7qsDCnvJxCoTTNmdECukeHfzrUjTSw72KZoM5KCRV78Wrctai1Qn6yRQz9BOSguxewLfzHtnT43/MLdwFXirJ/Ajquve2NAtYmyGCq5HcvpDAyi7lQ0nFBnrWv5zU3YxrISIpjovVyJjfPx8SCRlYZwVeUq6N2yAxCzJxbElZPtaTSoXBIFtoas2NXnCWPgenBa/2bbLQqfgbN8VQ9RaUISKNuYDIn4+eO72+RxF9THzZeV17pnhTVK88XU4asHot1gXwAt4vEhSjdUBC9KUIkfukI6F4JFxtvuO96octRahdV1Qg0vF+D0+SPy2HxqjgZWgPE2Xh/NmuIXwbE0wkymR2wrgj8Hd4C92keo2NBRh9dD7D2negnVYaYsC+3k/si5HNuCHnHQ== tunnel@labstack.com")
)

func splitHostPort(addr string) (host string, port int, err error) {
	parts := strings.Split(addr, ":")
	if len(parts) == 1 {
		port, err = strconv.Atoi(parts[0])
		if err != nil {
			return
		}
	} else if len(parts) == 2 {
		host = parts[0]
		port, err = strconv.Atoi(parts[1])
		if err != nil {
			return
		}
	}
	return
}

func newTunnel(req *StartRequest) (t *Tunnel, err error) {
	t = &Tunnel{
		startChan:     make(chan bool),
		acceptChan:    make(chan net.Conn),
		reconnectChan: make(chan error),
		stopChan:      make(chan bool),
		Host:          "labstack.me:22",
		Protocol:      req.Protocol,
		RemoteHost:    "0.0.0.0",
		RemotePort:    80,
	}
	e := new(Error)

	if req.Name != "" {
		key := viper.GetString("api_key")
		if key == "" {
			return nil, errors.New("failed to find api key in the config")
		}
		res, err := resty.New().R().
			SetAuthToken(key).
			SetHeader("Content-Type", "application/json").
			SetResult(t).
			SetError(e).
			SetHeader("User-Agent", "labstack/tunnel").
			Get(fmt.Sprintf("https://tunnel.labstack.com/api/v1/configurations/%s", req.Name))
		if err != nil {
			return nil, fmt.Errorf("failed to the find tunnel: %v", err)
		} else if res.StatusCode() != http.StatusOK {
			return nil, fmt.Errorf("failed to the find tunnel: %s", e.Message)
		}
		t.User = fmt.Sprintf("key=%s,name=%s", key, req.Name)
		t.Host += ":22"
	} else {
		t.Name = random.String(3, random.Lowercase)
		if req.Protocol == ProtocolTLS {
			t.User = "tls=true"
		}
	}
	t.TargetAddress = req.Address
	t.TargetHost, t.TargetPort, err = splitHostPort(req.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target address: %v", err)
	}
	if t.Protocol != ProtocolHTTP {
		t.RemotePort = 0
	}
	tunnels[t.Name] = t
	return
}

func (t *Tunnel) start(req *StartRequest, rep *StartReply) {
RECONNECT:
	if t.Status == StatusReconnecting {
		t.CreatedAt = time.Time{}
		t.reconnectWait++
		if t.reconnectWait > 5 {
			t.reconnectWait = 1
		}
		time.Sleep(t.reconnectWait * time.Second)
		log.Warnf("reconnecting tunnel: name=%s", t.Name)
	}
	hostKey, _, _, _, err := ssh.ParseAuthorizedKey(hostBytes)
	if err != nil {
		log.Fatalf("failed to parse host key: %v", err)
	}
	config := &ssh.ClientConfig{
		User: t.User,
		Auth: []ssh.AuthMethod{
			ssh.Password("password"),
		},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
		BannerCallback: func(message string) error {
			if !t.HideBanner {
				fmt.Print(message)
			}
			return nil
		},
	}
	c := new(ssh.Client)

	// Connect
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
			URL:    &url.URL{Path: t.Host},
			Host:   t.Host,
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

		conn, chans, reqs, err := ssh.NewClientConn(tcp, t.Host, config)
		if err != nil {
			log.Fatalf("cannot open new session: %v", err)
		}
		c = ssh.NewClient(conn, chans, reqs)
	} else {
		c, err = ssh.Dial("tcp", t.Host, config)
	}
	if err != nil {
		t.Status = StatusReconnecting
		goto RECONNECT
	}
	defer c.Close()

	// Session
	sess, err := c.NewSession()
	if err != nil {
		log.Fatalf("failed to create session: %v", err)
	}
	defer sess.Close()
	r, err := sess.StdoutPipe()
	if err != nil {
		log.Print(err)
	}
	br := bufio.NewReader(r)

	go func() {
		for {
			line, _, err := br.ReadLine()
			if err != nil {
				if err == io.EOF {
					// ch <- EventReconnect
					return
				} else {
					log.Fatalf("failed to read: %v", err)
				}
			}
			l := string(line)
			if strings.Contains(l, "traffic from") {
				i := strings.LastIndex(l, " ")
				t.RemoteURI = l[i+1:]
			}
			fmt.Printf("%s\n", l)
			t.reconnectWait = 0
			t.Status = StatusOnline
			t.CreatedAt = time.Now()
			t.startChan <- true
		}
	}()

	// Remote listener
	l, err := c.Listen("tcp", fmt.Sprintf("%s:%d", t.RemoteHost, t.RemotePort))
	if err != nil {
		log.Fatalf("failed to listen on remote host: %v", err)
	}
	defer l.Close()

	for {
		go func() {
			in, err := l.Accept()
			if err != nil {
				t.reconnectChan <- err
				return
			} else {
				t.acceptChan <- in
			}
		}()
		select {
		case <-t.stopChan:
			log.Warnf("stopping tunnel: name=%s", t.Name)
			return
		case in := <-t.acceptChan:
			go t.handleConnection(in)
		case <-t.reconnectChan:
			t.Status = StatusReconnecting
			goto RECONNECT
		}
	}
}

func (t *Tunnel) handleConnection(in net.Conn) {
	defer in.Close()

	// Target connection
	out, err := net.Dial("tcp", fmt.Sprintf("%s:%d", t.TargetHost, t.TargetPort))
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
