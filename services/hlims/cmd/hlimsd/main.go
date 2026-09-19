// The hlimsd command runs the Home Lab Information Management System server.
package main

import (
	"log"

	"github.com/TylerHillery/homelab/services/hlims"
)

func main() {
	if err := hlims.Run(); err != nil {
		log.Fatal(err)
	}
}
