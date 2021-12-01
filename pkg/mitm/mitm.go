package mitm

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"scf-proxy/pkg/scf"
	"strings"
	"sync"
)

const (
	CONNECT = "CONNECT"
)

// HandlerWrapper wraps an http.Handler with MITM'ing functionality
type HandlerWrapper struct {
	cryptoConf      *CryptoConfig
	wrapped         http.Handler
	pk              *PrivateKey
	pkPem           []byte
	issuingCert     *Certificate
	issuingCertPem  []byte
	serverTLSConfig *tls.Config
	dynamicCerts    *Cache
	certMutex       sync.Mutex
}

func Wrap(handler http.Handler, cryptoConf *CryptoConfig) (*HandlerWrapper, error) {
	wrapper := &HandlerWrapper{
		cryptoConf:   cryptoConf,
		wrapped:      handler,
		dynamicCerts: NewCache(),
	}
	err := wrapper.initCrypto()
	if err != nil {
		return nil, err
	}
	return wrapper, nil
}

// ServeHTTP implements ServeHTTP from http.Handler
func (wrapper *HandlerWrapper) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	auth := req.Header.Get("Proxy-Authorization")
	if auth =="" || len(auth) < 6{
		resp.Write([]byte("hello world"))
		return
	}
	auth = auth[6:]
	decode_auth,err := base64.StdEncoding.DecodeString(auth)
	if err!=nil{
		resp.Write([]byte("hello world"))
		return
	}
	if string(decode_auth) != "zabbix:0xfafu"{
		resp.Write([]byte("hello world"))
		return
	}
	if req.Method == CONNECT {
		wrapper.intercept(resp, req)
	} else {
		// wrapper.wrapped.ServeHTTP(resp, req)
		reqdump, _ := httputil.DumpRequest(req, true)
		fmt.Println("dump req:", string(reqdump))
		scf.HandlerHttp(resp, req)
	}
}

func (wrapper *HandlerWrapper) intercept(resp http.ResponseWriter, req *http.Request) {
	// Find out which host to MITM
	addr := hostIncludingPort(req)
	host := strings.Split(addr, ":")[0]

	cert, err := wrapper.mitmCertForName(host)
	if err != nil {
		msg := fmt.Sprintf("Could not get mitm cert for name: %s\nerror: %s", host, err)
		respBadGateway(resp, msg)
		return
	}

	connIn, _, err := resp.(http.Hijacker).Hijack()
	if err != nil {
		msg := fmt.Sprintf("Unable to access underlying connection from client: %s", err)
		respBadGateway(resp, msg)
		return
	}
	tlsConfig := makeConfig(wrapper.cryptoConf.ServerTLSConfig)

	tlsConfig.Certificates = []tls.Certificate{*cert}
	tlsConnIn := tls.Server(connIn, tlsConfig)

	listener := &mitmListener{tlsConnIn}

	handler := http.HandlerFunc(func(resp2 http.ResponseWriter, req2 *http.Request) {
		req2.URL.Scheme = "https"
		req2.URL.Host = req2.Host
		req2.RequestURI = req2.URL.String()
		// wrapper.wrapped.ServeHTTP(resp2, req2)

		scf.HandlerHttp(resp2, req2)
	})

	go func() {
		err = http.Serve(listener, handler)
		if err != nil && err != io.EOF {
			log.Printf("Error serving mitm'ed connection: %s", err)
		}
	}()

	connIn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
}

func makeConfig(template *tls.Config) *tls.Config {
	tlsConfig := &tls.Config{}
	if template != nil {
		// Copy the provided tlsConfig
		*tlsConfig = *template
	}
	return tlsConfig
}

func hostIncludingPort(req *http.Request) (host string) {
	host = req.Host
	if !strings.Contains(host, ":") {
		host = host + ":443"
	}
	return
}

func respBadGateway(resp http.ResponseWriter, msg string) {
	log.Println(msg)
	resp.WriteHeader(502)
	resp.Write([]byte(msg))
}
