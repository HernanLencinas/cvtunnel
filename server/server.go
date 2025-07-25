package chserver

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"time"

	chshare "github.com/HernanLencinas/cvtunnel/share"
	"github.com/HernanLencinas/cvtunnel/share/ccrypto"
	"github.com/HernanLencinas/cvtunnel/share/cio"
	"github.com/HernanLencinas/cvtunnel/share/cnet"
	"github.com/HernanLencinas/cvtunnel/share/settings"
	"github.com/gorilla/websocket"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	KeySeed   string
	KeyFile   string
	AuthFile  string
	Auth      string
	Proxy     string
	Socks5    bool
	Reverse   bool
	KeepAlive time.Duration
	TLS       TLSConfig
}

type Server struct {
	*cio.Logger
	config       *Config
	fingerprint  string
	httpServer   *cnet.HTTPServer
	reverseProxy *httputil.ReverseProxy
	sessCount    int32
	sessions     *settings.Users
	sshConfig    *ssh.ServerConfig
	users        *settings.UserIndex
}

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  settings.EnvInt("WS_BUFF_SIZE", 0),
	WriteBufferSize: settings.EnvInt("WS_BUFF_SIZE", 0),
}

func NewServer(c *Config) (*Server, error) {
	server := &Server{
		config:     c,
		httpServer: cnet.NewHTTPServer(),
		Logger:     cio.NewLogger("server"),
		sessions:   settings.NewUsers(),
	}
	server.Info = true
	server.users = settings.NewUserIndex(server.Logger)
	if c.AuthFile != "" {
		if err := server.users.LoadUsers(c.AuthFile); err != nil {
			return nil, err
		}
	}
	if c.Auth != "" {
		u := &settings.User{Addrs: []*regexp.Regexp{settings.UserAllowAll}}
		u.Name, u.Pass = settings.ParseAuth(c.Auth)
		if u.Name != "" {
			server.users.AddUser(u)
		}
	}

	var pemBytes []byte
	var err error
	if c.KeyFile != "" {
		var key []byte

		if ccrypto.IsCvtKey([]byte(c.KeyFile)) {
			key = []byte(c.KeyFile)
		} else {
			key, err = os.ReadFile(c.KeyFile)
			if err != nil {
				log.Fatalf("Failed to read key file %s", c.KeyFile)
			}
		}

		pemBytes = key
		if ccrypto.IsCvtKey(key) {
			pemBytes, err = ccrypto.CvtKey2PEM(key)
			if err != nil {
				log.Fatalf("Invalid key %s", string(key))
			}
		}
	} else {
		//generate private key (optionally using seed)
		pemBytes, err = ccrypto.Seed2PEM(c.KeySeed)
		if err != nil {
			log.Fatal("Failed to generate key")
		}
	}

	//convert into ssh.PrivateKey
	private, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		log.Fatal("Failed to parse key")
	}
	//fingerprint this key
	server.fingerprint = ccrypto.FingerprintKey(private.PublicKey())
	//create ssh config
	server.sshConfig = &ssh.ServerConfig{
		ServerVersion:    "SSH-" + chshare.ProtocolVersion + "-server",
		PasswordCallback: server.authUser,
	}
	server.sshConfig.AddHostKey(private)
	//setup reverse proxy
	if c.Proxy != "" {
		u, err := url.Parse(c.Proxy)
		if err != nil {
			return nil, err
		}
		if u.Host == "" {
			return nil, server.Errorf("Missing protocol (%s)", u)
		}
		server.reverseProxy = httputil.NewSingleHostReverseProxy(u)
		//always use proxy host
		server.reverseProxy.Director = func(r *http.Request) {
			//enforce origin, keep path
			r.URL.Scheme = u.Scheme
			r.URL.Host = u.Host
			r.Host = u.Host
		}
	}
	//print when reverse tunnelling is enabled
	if c.Reverse {
		server.Infof("Tunel reverso activado")
	}
	return server, nil
}

func (s *Server) Run(host, port string) error {
	if err := s.Start(host, port); err != nil {
		return err
	}
	return s.Wait()
}

func (s *Server) Start(host, port string) error {
	return s.StartContext(context.Background(), host, port)
}

func (s *Server) StartContext(ctx context.Context, host, port string) error {
	//s.Infof("Fingerprint %s", s.fingerprint)
	if s.users.Len() > 0 {
		s.Infof("Autenticacion activada")
	}
	if s.reverseProxy != nil {
		s.Infof("Proxy reverso activado")
	}
	l, err := s.listener(host, port)
	if err != nil {
		return err
	}
	h := http.Handler(http.HandlerFunc(s.handleClientHandler))
	/* 	if s.Debug {
		o := requestlog.DefaultOptions
		o.TrustProxy = true
		h = requestlog.WrapWith(h, o)
	} */
	return s.httpServer.GoServe(ctx, l, h)
}

// Wait waits for the http server to close
func (s *Server) Wait() error {
	return s.httpServer.Wait()
}

// Close forcibly closes the http server
func (s *Server) Close() error {
	return s.httpServer.Close()
}

// GetFingerprint is used to access the server fingerprint
func (s *Server) GetFingerprint() string {
	return s.fingerprint
}

// authUser is responsible for validating the ssh user / password combination
func (s *Server) authUser(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	// check if user authentication is enabled and if not, allow all
	if s.users.Len() == 0 {
		return nil, nil
	}
	// check the user exists and has matching password
	n := c.User()
	user, found := s.users.Get(n)
	if !found || user.Pass != string(password) {
		s.Debugf("Login failed for user: %s", n)
		return nil, errors.New("invalid authentication for username: %s")
	}
	s.sessions.Set(string(c.SessionID()), user)
	return nil, nil
}

func (s *Server) AddUser(user, pass string, addrs ...string) error {
	authorizedAddrs := []*regexp.Regexp{}
	for _, addr := range addrs {
		authorizedAddr, err := regexp.Compile(addr)
		if err != nil {
			return err
		}
		authorizedAddrs = append(authorizedAddrs, authorizedAddr)
	}
	s.users.AddUser(&settings.User{
		Name:  user,
		Pass:  pass,
		Addrs: authorizedAddrs,
	})
	return nil
}

func (s *Server) DeleteUser(user string) {
	s.users.Del(user)
}

func (s *Server) ResetUsers(users []*settings.User) {
	s.users.Reset(users)
}
