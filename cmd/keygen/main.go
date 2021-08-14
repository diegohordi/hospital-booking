package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/gob"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
)

var dir = flag.String("dir", "", "Directory where the keys will be stored")

func writeFile(filename string, key interface{}) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalln(err)
	}
	encoder := gob.NewEncoder(file)
	if err = encoder.Encode(key); err != nil {
		log.Fatalln(err)
		return
	}
	if err = file.Close(); err != nil {
		log.Fatalln(err)
		return
	}
}

func main() {
	flag.Parse()
	if *dir == "" {
		log.Fatal("no directory was given")
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalln(err)
	}

	publicKey := &privateKey.PublicKey

	writeFile(fmt.Sprintf("%s/%s", *dir, "private.key"), privateKey)
	writeFile(fmt.Sprintf("%s/%s", *dir, "public.key"), publicKey)

	pemKey := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	pemFile, err := os.Create(fmt.Sprintf("%s/%s", *dir, "private.pem"))
	if err != nil {
		log.Fatalln(err)
	}
	if err = pem.Encode(pemFile, pemKey); err != nil {
		log.Fatalln(err)
	}
	if err = pemFile.Close(); err != nil {
		log.Fatalln(err)
	}
}
