package tunnel

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"

	"golang.org/x/crypto/ssh"
)

type (
	Tunnel struct {
		Protocol   string
		RemoteHost string
		RemotePort int
		TargetHost string
		TargetPort int
	}
)

var (
	hostBytes = []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDoSLknvlFrFzroOlh1cqvcIFelHO+Wvj1UZ/p3J9bgsJGiKfh3DmBqEw1DOEwpHJz4zuV375TyjGuHuGZ4I4xztnwauhFplfEvriVHQkIDs6UnGwJVr15XUQX04r0i6mLbJs5KqIZTZuZ9ZGOj7ZWnaA7C07nPHGrERKV2Fm67rPvT6/qFikdWUbCt7KshbzdwwfxUohmv+NI7vw2X6vPU8pDaNEY7vS3YgwD/WlvQx+WDF2+iwLVW8OWWjFuQso6Eg1BSLygfPNhAHoiOWjDkijc8U9LYkUn7qsDCnvJxCoTTNmdECukeHfzrUjTSw72KZoM5KCRV78Wrctai1Qn6yRQz9BOSguxewLfzHtnT43/MLdwFXirJ/Ajquve2NAtYmyGCq5HcvpDAyi7lQ0nFBnrWv5zU3YxrISIpjovVyJjfPx8SCRlYZwVeUq6N2yAxCzJxbElZPtaTSoXBIFtoas2NXnCWPgenBa/2bbLQqfgbN8VQ9RaUISKNuYDIn4+eO72+RxF9THzZeV17pnhTVK88XU4asHot1gXwAt4vEhSjdUBC9KUIkfukI6F4JFxtvuO96octRahdV1Qg0vF+D0+SPy2HxqjgZWgPE2Xh/NmuIXwbE0wkymR2wrgj8Hd4C92keo2NBRh9dD7D2negnVYaYsC+3k/si5HNuCHnHQ== tunnel@labstack.com")
)

func (t *Tunnel) Create() {
	hostKey, _, _, _, err := ssh.ParseAuthorizedKey(hostBytes)
	if err != nil {
		log.Fatalf("Failed to parse host key %v\n", err)
	}
	config := &ssh.ClientConfig{
		User: t.Protocol,
		Auth: []ssh.AuthMethod{
			ssh.Password("password"),
		},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
		BannerCallback: func(message string) error {
			fmt.Print(message)
			return nil
		},
	}

	// Connect
	client, err := ssh.Dial("tcp", "labstack.me:22", config)
	if err != nil {
		log.Fatalf("Failed to connect %v\n", err)
	}
	defer client.Close()

	// Session
	sess, err := client.NewSession()
	if err != nil {
		log.Fatalf("Failed to create session %v\n", err)
	}
	defer sess.Close()
	r, err := sess.StdoutPipe()
	if err != nil {
		log.Print(err)
	}
	br := bufio.NewReader(r)
	go func() {
		for {
			line, _, _ := br.ReadLine()
			fmt.Printf("%s\n", line)
		}
	}()

	// Remote listener
	ln, err := client.Listen("tcp", fmt.Sprintf("%s:%d", t.RemoteHost, t.RemotePort))
	if err != nil {
		log.Fatalf("Failed to listen on remote host %v\n", err)
	}
	defer ln.Close()

	for {
		// Handle inbound connection
		in, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to accept connection %v\n", err)
			break
		}

		go func(in net.Conn) {
			defer in.Close()

			// Target connection
			out, err := net.Dial("tcp", fmt.Sprintf("%s:%d", t.TargetHost, t.TargetPort))
			if err != nil {
				log.Printf("%v\n", err)
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
				log.Printf("Failed to copy %v\n", err)
			}
		}(in)
	}
}
