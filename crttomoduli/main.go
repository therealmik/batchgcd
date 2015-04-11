package main

import (
	"bufio"
	"crypto/rsa"
	"github.com/therealmik/x509"
	"encoding/base64"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strings"
)

var zmapFile = flag.Bool("zmap", false, "Data is in zmap format")

func main() {
	log.SetOutput(os.Stderr)
	flag.Parse()
	if len(flag.Args()) == 0 {
		log.Fatal("No files specified")
	}

	ch := make(chan []byte)

	go printModuli(ch)
	for _, filename := range flag.Args() {
		log.Print("Loading moduli from ", filename)
		if *zmapFile {
			readZmap(filename, ch)
		} else {
			readPem(filename, ch)
		}
	}
}

func readPem(filename string, ch chan<- []byte) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	for len(data) > 0 {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}

		ch <- block.Bytes
	}
}

func readZmap(filename string, ch chan<- []byte) {
	fd, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(fd)
	var lineNumber int
	for scanner.Scan() {
		lineNumber += 1
		fields := strings.Split(scanner.Text(), ",")
		if len(fields) != 2 {
			log.Fatalf("Malformed line in %s:%d (should be exactly 1 comma per line), got %v", filename, lineNumber, fields)
		}
		data, err := base64.StdEncoding.DecodeString(fields[1])
		if err != nil {
			log.Fatalf("Malformed base64 in %s:%d: %v", filename, lineNumber, err)
		}
		ch <- data
	}
}

func printModuli(ch <-chan []byte) {
	smallest := big.NewInt(65537)
	for blob := range ch {
		cert, err := x509.ParseCertificate(blob)
		if err != nil {
			log.Printf("Error in cert %v: %s", err, base64.StdEncoding.EncodeToString(blob))
			continue
		}
		if cert.PublicKeyAlgorithm != x509.RSA {
			log.Printf("Skipping non-RSA certificate")
			continue
		}
		pk := cert.PublicKey.(*rsa.PublicKey)
		if pk.N.Cmp(smallest) < 1 {
			log.Printf("Skipping small/negative modulus")
			continue
		}
		fmt.Printf("%x,%s\n", pk.N, base64.StdEncoding.EncodeToString(cert.Raw))
	}
}
