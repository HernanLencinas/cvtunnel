package chshare

//this file exists to maintain backwards compatibility

import (
	"github.com/HernanLencinas/cvtunnel/share/ccrypto"
	"github.com/HernanLencinas/cvtunnel/share/cio"
	"github.com/HernanLencinas/cvtunnel/share/cnet"
	"github.com/HernanLencinas/cvtunnel/share/cos"
	"github.com/HernanLencinas/cvtunnel/share/settings"
	"github.com/HernanLencinas/cvtunnel/share/tunnel"
)

const (
	DetermRandIter = ccrypto.DetermRandIter
)

type (
	Config     = settings.Config
	Remote     = settings.Remote
	Remotes    = settings.Remotes
	User       = settings.User
	Users      = settings.Users
	UserIndex  = settings.UserIndex
	HTTPServer = cnet.HTTPServer
	ConnStats  = cnet.ConnCount
	Logger     = cio.Logger
	TCPProxy   = tunnel.Proxy
)

var (
	NewDetermRand    = ccrypto.NewDetermRand
	GenerateKey      = ccrypto.GenerateKey
	FingerprintKey   = ccrypto.FingerprintKey
	Pipe             = cio.Pipe
	NewLoggerFlag    = cio.NewLoggerFlag
	NewLogger        = cio.NewLogger
	Stdio            = cio.Stdio
	DecodeConfig     = settings.DecodeConfig
	DecodeRemote     = settings.DecodeRemote
	NewUsers         = settings.NewUsers
	NewUserIndex     = settings.NewUserIndex
	UserAllowAll     = settings.UserAllowAll
	ParseAuth        = settings.ParseAuth
	NewRWCConn       = cnet.NewRWCConn
	NewWebSocketConn = cnet.NewWebSocketConn
	NewHTTPServer    = cnet.NewHTTPServer
	GoStats          = cos.GoStats
	SleepSignal      = cos.SleepSignal
	NewTCPProxy      = tunnel.NewProxy
)

// EncodeConfig old version
func EncodeConfig(c *settings.Config) ([]byte, error) {
	return settings.EncodeConfig(*c), nil
}
