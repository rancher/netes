package proxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/v3"
)

var (
	wsDialer = &websocket.Dialer{}
)

const (
	cattleURL    = "http://localhost:8081/v3/dial"
	callbackHost = "localhost:8080"
)

func NewDialer(cluster *client.Cluster, accessKey, secretKey string) func(network, addr string) (net.Conn, error) {
	d := &dialer{
		clusterID: cluster.Id,
		accessKey: accessKey,
		secretKey: secretKey,
	}
	return d.Dial
}

type dialer struct {
	clusterID string
	accessKey string
	secretKey string
}

func (p *dialer) Dial(network, addr string) (net.Conn, error) {
	conn, err := p.openConnection(network, addr)
	return &wsConn{
		Conn: conn,
		conn: &WebSocketIO{conn},
	}, err
}

func (p *dialer) openConnection(network, addr string) (*websocket.Conn, error) {
	data := map[string]interface{}{
		"clusterId": p.clusterID,
		"protocol":  network,
		"address":   addr,
	}
	content, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", cattleURL, bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(p.accessKey, p.secretKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("Invalid response: %d", resp.StatusCode)
	}

	hostAccess := client.HostAccess{}
	err = json.NewDecoder(resp.Body).Decode(&hostAccess)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshaling response")
	}

	u := fmt.Sprintf("%s?token=%s", hostAccess.Url, hostAccess.Token)
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	parsed.Host = callbackHost
	conn, _, err := p.websocket(parsed.String())
	return conn, err
}

func (p *dialer) websocket(url string) (*websocket.Conn, *http.Response, error) {
	httpHeaders := http.Header{}
	s := p.accessKey + ":" + p.secretKey
	httpHeaders.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(s)))

	return wsDialer.Dial(url, http.Header(httpHeaders))
}

type wsConn struct {
	*websocket.Conn
	conn *WebSocketIO
	temp []byte
}

func (w *wsConn) Read(buf []byte) (int, error) {
	var err error
	if len(w.temp) > 0 {
		return w.readFromTemp(buf), nil
	}

	w.temp, err = w.conn.Read()
	return w.readFromTemp(buf), err
}

func (w *wsConn) Write(p []byte) (n int, err error) {
	return w.conn.Write(p)
}

func (w *wsConn) SetDeadline(t time.Time) error {
	if err := w.SetReadDeadline(t); err != nil {
		return err
	}
	return w.SetWriteDeadline(t)
}

func (w *wsConn) readFromTemp(buf []byte) int {
	n := copy(buf, w.temp)
	w.temp = w.temp[n:]
	return n
}

type WebSocketIO struct {
	Conn *websocket.Conn
}

func (w *WebSocketIO) Read() ([]byte, error) {
	_, b, err := w.Conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	b, err = base64.StdEncoding.DecodeString(string(b))
	logrus.Debugf("Websocket Read: %d: %s", len(b), string(b))
	return b, err
}

func (w *WebSocketIO) Write(buf []byte) (int, error) {
	logrus.Debugf("Websocket Writer: %d: %s", len(buf), string(buf))
	str := base64.StdEncoding.EncodeToString(buf)
	err := w.Conn.WriteMessage(websocket.TextMessage, []byte(str))
	return len(buf), err
}
