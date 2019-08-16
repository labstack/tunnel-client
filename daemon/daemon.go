package daemon

import (
	"github.com/labstack/gommon/log"
	"github.com/spf13/viper"
	"io/ioutil"
	"net"
	"net/rpc"
)

type (
	Daemon struct {
	}

	Protocol string

	Status string

	StartRequest struct {
		Name     string
		Address  string
		Protocol Protocol
	}

	StartReply struct {
	}

	StopRequest struct {
		Name string
	}

	StopReply struct {
	}

	PSRequest struct {
	}

	PSReply struct {
		Tunnels []*Tunnel
	}

	RMRequest struct {
		Name string
	}

	RMReply struct {
	}
)

const (
	ProtocolHTTP = "http"
	ProtocolTCP  = "tcp"
	ProtocolTLS  = "tls"

	StatusReconnecting = "reconnecting"
	StatusOnline       = "online"
	StatusOffline      = "offline"
)

func (d *Daemon) Start(req *StartRequest, rep *StartReply) error {
	t, err := newTunnel(req)
	if err != nil {
		return err
	}
	go t.start(req, rep)
	<-t.startChan
	return nil
}

func (d *Daemon) Stop(req *StopRequest, rep *StopReply) error {
	t := tunnels[req.Name]
	t.stopChan <- true
	t.Status = StatusOffline
	return nil
}

func (d *Daemon) PS(req *PSRequest, rep *PSReply) error {
	for _, t := range tunnels {
		rep.Tunnels = append(rep.Tunnels, t)
	}
	return nil
}

func (d *Daemon) RM(req *RMRequest, rep *RMReply) error {
	delete(tunnels, req.Name)
	return nil
}

func Start() {
	d := new(Daemon)
	rpc.Register(d)
	// syscall.Unlink("/tmp/rpc.sock")
	// l, e := net.Listen("unix", "/tmp/rpc.sock")
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
