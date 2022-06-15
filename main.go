// Copyright 2022 MaoLongLong. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/oklog/run"
	log "github.com/sirupsen/logrus"
)

var cnt uint64

var (
	debug = flag.Bool("debug", false, "Print debug info")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: relay [options] <from> <to>\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "\trelay 127.0.0.1:8081 127.0.0.1:8080\n")
	fmt.Fprintf(os.Stderr, "\trelay -debug 127.0.0.1:8081 127.0.0.1:8080\n")
	os.Exit(2)
}

func handle(cli *net.TCPConn, taddr *net.TCPAddr) {
	id := atomic.AddUint64(&cnt, 1)
	log.Infof("[%v] Accepted from: %v", id, cli.RemoteAddr())

	srv, err := net.DialTCP("tcp", nil, taddr)
	if err != nil {
		log.WithError(err).Errorf("[%v] Failed to dial %v", id, taddr)
		return
	}
	log.Infof("[%v] Connected to server: %v", id, srv.RemoteAddr())

	closeSrv := func(_ error) {
		srv.Close()
		log.Infof("[%v] Server connection closed", id)
	}
	closeCli := func(_ error) {
		cli.Close()
		log.Infof("[%v] Client connection closed", id)
	}

	var g run.Group
	if *debug {
		g.Add(func() error { return dump(id, "server", cli, srv) }, closeSrv)
		g.Add(func() error { return dump(id, "client", srv, cli) }, closeCli)
	} else {
		g.Add(func() error { _, err := io.Copy(cli, srv); return err }, closeSrv)
		g.Add(func() error { _, err := io.Copy(srv, cli); return err }, closeCli)
	}

	if err := g.Run(); err != nil && err != io.EOF {
		log.WithError(err).Errorf("[%v] Failed to transfer data", id)
	}
}

func dump(id uint64, source string, dst io.Writer, src io.Reader) error {
	buf := make([]byte, 32*1024)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			log.Debugf("[%v] Read from %v:\n%v\n", id, source, hex.Dump(buf[:nr]))

			nw, ew := dst.Write(buf[:nr])
			if nw < 0 || nr < nw {
				return errors.New("invalid write result")
			}
			if ew != nil {
				return ew
			}
			if nr != nw {
				return io.ErrShortWrite
			}
		}
		if er != nil {
			return er
		}
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 2 {
		usage()
	}

	log.SetFormatter(&log.TextFormatter{
		ForceColors:      true,
		DisableTimestamp: true,
	})
	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	faddr, err := net.ResolveTCPAddr("tcp", flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	taddr, err := net.ResolveTCPAddr("tcp", flag.Arg(1))
	if err != nil {
		log.Fatal(err)
	}

	ln, err := net.ListenTCP("tcp", faddr)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			conn, err := ln.AcceptTCP()
			if err != nil {
				if !errors.Is(err, net.ErrClosed) {
					log.Fatal(err)
				}
				return
			}
			go handle(conn, taddr)
		}
	}()
	log.Infof("Relay server listening on %v", faddr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	<-quit
	log.Info("Shutdown...")
	ln.Close()
	wg.Wait()
}
