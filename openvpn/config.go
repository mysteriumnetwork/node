package openvpn

import (
	"net"
	"strconv"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/stamp/go-openssl"
)

type Config struct {
	remote string
	flags  map[string]bool
	values map[string]string
	params []string
}

func NewConfig() *Config {
	return &Config{
		flags:  make(map[string]bool),
		values: make(map[string]string),
		params: make([]string, 0),
	}
}

func (c *Config) Set(key, val string) {
	a := strings.Split("--"+key+" "+val, " ")
	for _, ar := range a {
		c.params = append(c.params, ar)
	}
}

func (c *Config) Flag(key string) {
	//c.params = append(c.params, "--"+key)
	a := strings.Split("--"+key, " ")
	for _, ar := range a {
		c.params = append(c.params, ar)
	}
}

func (c *Config) Validate() (config []string, err error) {
	return c.params, nil
}

func (c *Config) ServerMode(port int, ca *openssl.CA, cert *openssl.Cert, dh *openssl.DH, ta *openssl.TA) {
	c.Set("mode", "server")
	c.Set("port", strconv.Itoa(port))
	c.Flag("tls-server")

	c.Set("ca", ca.GetFilePath())
	c.Set("crl-verify", ca.GetCRLPath())
	c.Set("cert", cert.GetFilePath())
	c.Set("key", cert.GetKeyPath())
	c.Set("dh", dh.GetFilePath())
	c.Set("tls-auth", ta.GetFilePath())
}
func (c *Config) ClientMode(ca *openssl.CA, cert *openssl.Cert, dh *openssl.DH, ta *openssl.TA) {
	c.Flag("client")
	c.Flag("tls-client")

	c.Set("ca", ca.GetFilePath())
	c.Set("cert", cert.GetFilePath())
	c.Set("key", cert.GetKeyPath())
	c.Set("dh", dh.GetFilePath())
	c.Set("tls-auth", ta.GetFilePath())
}

func (c *Config) Remote(r string, port int) {
	c.Set("port", strconv.Itoa(port))
	c.Set("remote", r)
	c.remote = r
}
func (c *Config) Protocol(p string) {
	c.Set("proto", p)
}
func (c *Config) Device(t string) {
	c.Set("dev", t)
}
func (c *Config) IpConfig(local, remote string) {
}
func (c *Config) IpPool(pool string) {

	ip, net, err := net.ParseCIDR(pool)
	if err != nil {
		log.Error(err)
		return
	}

	c.Set("server", ip.String()+" "+strconv.Itoa(int(net.Mask[0]))+"."+strconv.Itoa(int(net.Mask[1]))+"."+strconv.Itoa(int(net.Mask[2]))+"."+strconv.Itoa(int(net.Mask[3])))
}

func (c *Config) Secret(key string) {
	c.Set("secret", key)
}

func (c *Config) KeepAlive(interval, timeout int) {
	c.Set("keepalive", strconv.Itoa(interval)+" "+strconv.Itoa(timeout))
}
func (c *Config) PingTimerRemote() {
	c.Flag("ping-timer-rem")
}
func (c *Config) PersistTun() {
	c.Flag("persist-tun")
}
func (c *Config) PersistKey() {
	c.Flag("persist-key")
}

func (c *Config) Compression() {
	//comp-lzo
}
func (c *Config) ClientToClient() {
	c.Flag("client-to-client")
}

func (c *Config) setManagementPath(path string) {
	c.Set("management", path+" unix")
	c.Flag("management-client")
	c.Flag("management-hold")
	c.Flag("management-signal")
	c.Flag("management-up-down")

	log.Info("Current config:", c)
}
