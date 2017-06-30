package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"

	"github.com/julienschmidt/httprouter"
)

var argFrom string
var argTo string
var argPort int
var argHTTPS bool
var argProtocol string

func copyHeaders(dst *http.Header, src *http.Header) {
	for k, vals := range *src {
		for _, v := range vals {
			log.Printf("Copying header %s: %s", k, v)
			dst.Set(k, v)
		}
	}
}

func executeProxyCall(w http.ResponseWriter, r *http.Request, path string) {
	method := r.Method
	url := fmt.Sprintf("%s://%s%s", argProtocol, argTo, path)

	log.Printf("%s -> %s\n", method, url)

	client := &http.Client{}

	req, err := http.NewRequest(method, url, r.Body)

	if err != nil {
		fmt.Printf("Erorr while creating request %s\n", err)
		return
	}

	sourceHeaders := r.Header
	destinationHeaders := req.Header
	copyHeaders(&destinationHeaders, &sourceHeaders)

	proxyResp, err := client.Do(req)

	if err != nil {
		fmt.Printf("Erorr while executing request %s\n", err)
		return
	}

	sourceHeaders = proxyResp.Header
	destinationHeaders = w.Header()
	copyHeaders(&destinationHeaders, &sourceHeaders)

	io.Copy(w, proxyResp.Body)
}

func proxyHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	hostMatch := regexp.MustCompile(fmt.Sprintf("^%s.*$", argFrom))
	host := r.Host

	if hostMatch.MatchString(host) {
		path := ps.ByName("path")
		executeProxyCall(w, r, path)
	} else {
		fmt.Fprintf(w, "Could not process url with host %s", host)
	}

}

func init() {
	flag.StringVar(&argFrom, "from", "", "source domain that proxy will work with")
	flag.StringVar(&argTo, "to", "", "target domain that proxy will work with")
	flag.IntVar(&argPort, "port", 8080, "port that proxy should use")
	flag.BoolVar(&argHTTPS, "https", false, "is target https enabled")
	flag.Parse()

	if argFrom == "" || argTo == "" {
		log.Fatalf("from and to cant be empty")
	}

	if argHTTPS {
		argProtocol = "https"
	} else {
		argProtocol = "http"
	}

}

func startServer() {
	router := httprouter.New()
	router.GET("/*path", proxyHandler)
	router.POST("/*path", proxyHandler)
	router.PUT("/*path", proxyHandler)
	router.DELETE("/*path", proxyHandler)

	log.Printf("Starting server on port %d", argPort)
	log.Printf("Server will proxy %s -> %s://%s", argFrom, argProtocol, argTo)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", argPort), router))
}

func main() {
	startServer()
}
