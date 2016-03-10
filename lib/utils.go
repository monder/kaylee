package lib

import (
	"log"
)

func Assert(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
