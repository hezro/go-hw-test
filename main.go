package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"go-hw-test/mynewpackage"

	"github.com/miekg/dns"
	"github.com/spf13/cobra"
	"golang.org/x/net/html"
	"golang.org/x/net/http2"
)

var NS = "ns7.dns.tds.net"

const url = "https://localhost:8000"

func main() {
	cmd := &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hello, World!!")
			mynewpackage.PrintHello()
		},
	}
	fmt.Println("Calling cmd.Execute()!")
	cmd.Execute()

	parseHTML()

	//testsrvpackage.runSrv()
	test := http2.NextProtoTLS
	fmt.Println(test)
	testHTTP2()

	conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		log.Fatal("error making client from default file", err)
	}

	m := &dns.Msg{
		Question: make([]dns.Question, 1),
	}

	c := new(dns.Client)
	addr := addresses(conf, c, NS)
	if len(addr) == 0 {
		log.Fatalf("No address found for %s\n", NS)
	}

	for _, a := range addr {
		m.Question[0] = dns.Question{"version.bind.", dns.TypeTXT, dns.ClassCHAOS}
		in, rtt, err := c.Exchange(m, a)
		if err != nil {
			fmt.Println(err)
		}
		if in != nil && len(in.Answer) > 0 {
			log.Printf("(time %.3d µs) %v\n", rtt/1e3, in.Answer[0])
		}
		m.Question[0] = dns.Question{"hostname.bind.", dns.TypeTXT, dns.ClassCHAOS}
		in, rtt, err = c.Exchange(m, a)
		if err != nil {
			fmt.Println(err)
		}
		if in != nil && len(in.Answer) > 0 {
			log.Printf("(time %.3d µs) %v\n", rtt/1e3, in.Answer[0])
		}
	}

}

func testHTTP2() {

	// Create a pool with the server certificate since it is not signed
	// by a known CA
	caCert, err := ioutil.ReadFile("server.crt")
	if err != nil {
		log.Fatalf("Reading server certificate: %s", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Create TLS configuration with the certificate of the server
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}
	client := &http.Client{}
	client.Transport = &http2.Transport{
		TLSClientConfig: tlsConfig,
	}

	// Perform the request
	resp, err := client.Get(url)
	if err != nil {
		log.Fatalf("Failed get: %s", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed reading response body: %s", err)
	}
	fmt.Printf(
		"Got response %d: %s %s\n",
		resp.StatusCode, resp.Proto, string(body))
}

func do(t chan *dns.Msg, wg *sync.WaitGroup, c *dns.Client, m *dns.Msg, addr string) {
	defer wg.Done()
	r, _, err := c.Exchange(m, addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	t <- r
}

func addresses(conf *dns.ClientConfig, c *dns.Client, name string) (ips []string) {
	m4 := new(dns.Msg)
	m4.SetQuestion(dns.Fqdn(NS), dns.TypeA)
	m6 := new(dns.Msg)
	m6.SetQuestion(dns.Fqdn(NS), dns.TypeAAAA)
	t := make(chan *dns.Msg, 2)

	wg := new(sync.WaitGroup)
	wg.Add(2)
	go do(t, wg, c, m4, net.JoinHostPort(conf.Servers[0], conf.Port))
	go do(t, wg, c, m6, net.JoinHostPort(conf.Servers[0], conf.Port))
	wg.Wait()
	close(t)

	for d := range t {
		if d.Rcode == dns.RcodeSuccess {
			for _, a := range d.Answer {
				switch t := a.(type) {
				case *dns.A:
					ips = append(ips, net.JoinHostPort(t.A.String(), "53"))
				case *dns.AAAA:
					ips = append(ips, net.JoinHostPort(t.AAAA.String(), "53"))

				}
			}
		}
	}

	return ips
}

// using net/html

func parseHTML() {
	webPage := "http://webcode.me/countries.html"
	data, err := getHtmlPage(webPage)

	if err != nil {
		log.Fatal(err)
	}

	parseAndShow(data)
}

func getHtmlPage(webPage string) (string, error) {

	resp, err := http.Get(webPage)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {

		return "", err
	}

	return string(body), nil
}

func parseAndShow(text string) {

	tkn := html.NewTokenizer(strings.NewReader(text))

	var isTd bool
	var n int

	//tnk, err := html.ParseFragment(strings.NewReader(text), nil)
	tnk, err := html.Parse(strings.NewReader(text))
	println(tnk)
	if err != nil {
		panic("error generating context")
	}

	for {

		tt := tkn.Next()

		switch {

		case tt == html.ErrorToken:
			return

		case tt == html.StartTagToken:

			t := tkn.Token()
			isTd = t.Data == "td"

		case tt == html.TextToken:

			t := tkn.Token()

			if isTd {

				fmt.Printf("%s ", t.Data)
				n++
			}

			if isTd && n%3 == 0 {

				fmt.Println()
			}

			isTd = false
		}
	}
}
