package main

import (
	"log"

	"github.com/aksiksi/ncdmv/pkg/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
