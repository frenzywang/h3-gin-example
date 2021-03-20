package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	_ "net/http/pprof"

	"github.com/frenzywang/h3-gin-example/ssl"
	"github.com/gin-gonic/gin"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/lucas-clemente/quic-go/logging"
	"github.com/lucas-clemente/quic-go/qlog"
)

func setupHandler() http.Handler {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	return r
}

func main() {
	// defer profile.Start().Stop()
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	// runtime.SetBlockProfileRate(1)

	// can fallback to http1 and http2(according to your client) when enable tcp
	tcp := flag.Bool("tcp", true, "also listen on TCP")
	enableQlog := flag.Bool("qlog", false, "output a qlog (in the same directory)")
	flag.Parse()

	logger := log.Default()

	handler := setupHandler()
	quicConf := &quic.Config{}
	if *enableQlog {
		quicConf.Tracer = qlog.NewTracer(func(_ logging.Perspective, connID []byte) io.WriteCloser {
			filename := fmt.Sprintf("server_%x.qlog", connID)
			f, err := os.Create(filename)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Creating qlog file %s.\n", filename)
			return f
		})
	}

	locahost := "localhost:6121"
	var err error
	if *tcp {
		certFile, keyFile := ssl.GetCertificatePaths()
		err = http3.ListenAndServe(locahost, certFile, keyFile, handler)
	} else {
		server := http3.Server{
			Server:     &http.Server{Handler: handler, Addr: locahost},
			QuicConfig: quicConf,
		}
		err = server.ListenAndServeTLS(ssl.GetCertificatePaths())
	}
	if err != nil {
		logger.Fatal(err)
	}
}
