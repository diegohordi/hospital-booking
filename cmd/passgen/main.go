package main

import (
	"flag"
	"fmt"
	"hospital-booking/internal/auth"
	"log"
)

var pass = flag.String("pass", "", "Password to encrypt")

func main() {
	flag.Parse()
	if *pass == "" {
		log.Fatal("no password was given")
	}

	passHash, err := auth.EncryptPassword(*pass)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(passHash)
}
