// Copyright 2022 MaoLongLong. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s <from> <to>\n", os.Args[0])
	os.Exit(2)
}

func handle(from *net.TCPConn, taddr *net.TCPAddr) {
	defer from.Close()

	to, err := net.DialTCP("tcp", nil, taddr)
	if err != nil {
		log.Println(err)
		return
	}
	defer to.Close()

	go io.Copy(to, from)
	io.Copy(from, to)
}

func main() {
	if len(os.Args) != 3 {
		usage()
	}

	faddr, err := net.ResolveTCPAddr("tcp", os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	taddr, err := net.ResolveTCPAddr("tcp", os.Args[2])
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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	<-quit
	ln.Close()
	wg.Wait()
}
