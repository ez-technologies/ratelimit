// Command httpserver is a small net/http server that shows two limits from one ratelimit.Limiter:
//
//   - POST /login: per client IP, a burst of 3, then one more every 2 seconds.
//   - every route: one global limit shared by all clients, a burst of 5, then one more every 500 ms.
//
// Run it with `go run ./examples/httpserver` and send requests with curl, or run ./examples/httpserver/demo.sh.
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/ez-technologies/ratelimit"
)

const (
	loginEvery  = 2 * time.Second
	loginBurst  = 3
	globalEvery = 500 * time.Millisecond
	globalBurst = 5
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "address to listen on")
	flag.Parse()

	// One Limiter can hold any number of limits: the key decides which bucket is used.
	limits := ratelimit.New(10 * time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		if !limits.Allow("login:"+clientIP(r), loginEvery, loginBurst) {
			tooMany(w, "login", loginEvery)
			return
		}
		fmt.Fprintln(w, "login ok")
	})
	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello")
	})

	log.Printf("listening on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, globalLimit(limits, mux)))
}

// globalLimit refuses any request once the whole server has used its shared budget.
func globalLimit(limits *ratelimit.Limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limits.Allow("global", globalEvery, globalBurst) {
			tooMany(w, "global", globalEvery)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP is the address the connection comes from. Behind a reverse proxy, use the address the proxy reports
// instead, and only when the proxy is trusted: a client can put anything in X-Forwarded-For.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// tooMany answers 429 with a Retry-After of one refill interval, rounded up to whole seconds.
// The body names the limit that was hit, which makes the demo easier to follow.
func tooMany(w http.ResponseWriter, limit string, every time.Duration) {
	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(every.Seconds()))))
	http.Error(w, "too many requests ("+limit+" limit)", http.StatusTooManyRequests)
}
