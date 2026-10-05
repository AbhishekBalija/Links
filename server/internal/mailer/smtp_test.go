package mailer

import (
	"bufio"
	"net"
	"strings"
	"sync"
	"testing"
)

// fakeSMTP is a tiny mail server that accepts every message and keeps it,
// like a testing inbox such as Mailtrap.
type fakeSMTP struct {
	mu       sync.Mutex
	messages []string
	rcpts    []string
	authed   bool
}

func startFakeSMTP(t *testing.T) (*fakeSMTP, string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	server := &fakeSMTP{}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go server.serve(conn)
		}
	}()
	return server, listener.Addr().String()
}

func (s *fakeSMTP) serve(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	reply := func(line string) { conn.Write([]byte(line + "\r\n")) }
	reply("220 fake ready")
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(command, "EHLO"):
			reply("250-fake")
			reply("250 AUTH PLAIN")
		case strings.HasPrefix(command, "AUTH PLAIN"):
			s.mu.Lock()
			s.authed = true
			s.mu.Unlock()
			reply("235 ok")
		case strings.HasPrefix(command, "MAIL FROM"):
			reply("250 ok")
		case strings.HasPrefix(command, "RCPT TO"):
			s.mu.Lock()
			s.rcpts = append(s.rcpts, strings.TrimSpace(line[len("RCPT TO:"):]))
			s.mu.Unlock()
			reply("250 ok")
		case command == "DATA":
			reply("354 go on")
			var body strings.Builder
			for {
				part, err := reader.ReadString('\n')
				if err != nil || part == ".\r\n" {
					break
				}
				body.WriteString(part)
			}
			s.mu.Lock()
			s.messages = append(s.messages, body.String())
			s.mu.Unlock()
			reply("250 queued")
		case command == "QUIT":
			reply("221 bye")
			return
		default:
			reply("250 ok")
		}
	}
}

func TestSMTPMailerDeliversToATestingInbox(t *testing.T) {
	server, address := startFakeSMTP(t)
	host, port, _ := net.SplitHostPort(address)
	m := NewSMTPMailer(SMTPSettings{Host: host, Port: port, Username: "user", Password: "secret"}, "links@example.com", "https://links.example.com")

	if err := m.SendSignInCode("asha@gmail.com", "482913"); err != nil {
		t.Fatalf("send code: %v", err)
	}
	to := []Recipient{{Email: "a@gmail.com", FullName: "Asha Rao"}, {Email: "b@gmail.com", FullName: "Bala K"}}
	if err := m.SendEventNotice(to, EventNotice{Title: "Robotics meetup", Cancelled: true, When: "Fri 9 Oct, 11 am", Where: "CS Lab 2"}); err != nil {
		t.Fatalf("send notice: %v", err)
	}

	server.mu.Lock()
	defer server.mu.Unlock()
	if !server.authed {
		t.Error("the mailer didn't sign in to the mail server")
	}
	if len(server.messages) != 3 {
		t.Fatalf("messages = %d, want 3 (one code, two notices)", len(server.messages))
	}
	code := server.messages[0]
	if !strings.Contains(code, "Subject: Your LINKS sign-in code: 482913") || !strings.Contains(code, "Content-Type: text/html") || !strings.Contains(code, "482913") {
		t.Errorf("code message = %s", code)
	}
	if !strings.Contains(server.messages[1], "Hello Asha,") || !strings.Contains(server.messages[2], "Hello Bala,") {
		t.Error("each notice should greet its own recipient")
	}
}
