package mysterium

import "github.com/mysteriumnetwork/go-openvpn/openvpn3"

type sessionWrapper struct {
	session *openvpn3.Session
}

func (wrapper *sessionWrapper) Start() error {

	wrapper.session.Start("", openvpn3.Credentials{})
	return nil
}

func (wrapper *sessionWrapper) Stop() {
	wrapper.session.Stop()
}

func (wrapper *sessionWrapper) Wait() error {
	return wrapper.session.Wait()
}
