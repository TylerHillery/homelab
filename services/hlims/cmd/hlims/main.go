// The hlims command runs the Home Lab Information Management System server.
package main

import (
	"log"

	golink "github.com/TylerHillery/homelab/services/hlims"
)

func main() {
	if err := golink.Run(); err != nil {
		log.Fatal(err)
	}
}
