package chclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	chshare "github.com/HernanLencinas/cvtunnel/share"
	"github.com/HernanLencinas/cvtunnel/share/backoff"
	"github.com/HernanLencinas/cvtunnel/share/cnet"
	"github.com/HernanLencinas/cvtunnel/share/cos"
	"github.com/HernanLencinas/cvtunnel/share/settings"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

func (c *Client) connectionLoop(ctx context.Context) error {
	//connection loop!
	b := &backoff.Backoff{Max: c.config.MaxRetryInterval}
	for {
		connected, err := c.connectionOnce(ctx)
		//reset backoff after successful connections
		if connected {
			b.Reset()
		}
		//connection error
		attempt := int(b.Attempt())
		maxAttempt := c.config.MaxRetryCount
		//dont print closed-connection errors
		if strings.HasSuffix(err.Error(), "use of closed network connection") {
			err = io.EOF
		}
		//show error message and attempt counts (excluding disconnects)
		if err != nil && err != io.EOF {
			msg := fmt.Sprintf("Error de conexión: %s", err)
			if attempt > 0 {
				maxAttemptVal := fmt.Sprint(maxAttempt)
				if maxAttempt < 0 {
					maxAttemptVal = "unlimited"
				}
				msg += fmt.Sprintf(" (Intento: %d/%s)", attempt, maxAttemptVal)
			}
			c.Infof(msg)
		}
		//give up?
		if maxAttempt >= 0 && attempt >= maxAttempt {
			c.Infof("Me rindo...")
			break
		}
		d := b.Duration()
		c.Infof("Reintentando en %s...", d)
		select {
		case <-cos.AfterSignal(d):
			continue //retry now
		case <-ctx.Done():
			c.Infof("Cancelado")
			return nil
		}
	}
	c.Close()
	return nil
}

func (c *Client) connectionOnce(ctx context.Context) (connected bool, err error) {
	//already closed?
	select {
	case <-ctx.Done():
		return false, errors.New("Cancelado")
	default:
		//still open
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	//prepare dialer
	d := websocket.Dialer{
		HandshakeTimeout: settings.EnvDuration("WS_TIMEOUT", 45*time.Second),
		Subprotocols:     []string{chshare.ProtocolVersion},
		TLSClientConfig:  c.tlsConfig,
		ReadBufferSize:   settings.EnvInt("WS_BUFF_SIZE", 0),
		WriteBufferSize:  settings.EnvInt("WS_BUFF_SIZE", 0),
		NetDialContext:   c.config.DialContext,
	}
	//optional proxy
	if p := c.proxyURL; p != nil {
		if err := c.setProxy(p, &d); err != nil {
			return false, err
		}
	}
	wsConn, _, err := d.DialContext(ctx, c.server, c.config.Headers)
	if err != nil {
		return false, err
	}
	conn := cnet.NewWebSocketConn(wsConn)
	// perform SSH handshake on net.Conn
	c.Debugf("Handshaking...")
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, "", c.sshConfig)
	if err != nil {
		e := err.Error()
		if strings.Contains(e, "imposible autenticarse") {
			c.Infof("Autenticacion fallida")
			c.Debugf(e)
		} else {
			c.Infof(e)
		}
		return false, err
	}
	defer sshConn.Close()
	// send configuration
	c.Debugf("Sending config")
	t0 := time.Now()
	_, configerr, err := sshConn.SendRequest(
		"config",
		true,
		settings.EncodeConfig(c.computed),
	)
	if err != nil {
		c.Infof("Config verification failed")
		return false, err
	}
	if len(configerr) > 0 {
		return false, errors.New(string(configerr))
	}
	c.Infof("Conectado (Latencia %s)", time.Since(t0))
	//connected, handover ssh connection for tunnel to use, and block
	err = c.tunnel.BindSSH(ctx, sshConn, reqs, chans)
	c.Infof("Deconectado")
	connected = time.Since(t0) > 5*time.Second
	return connected, err
}
