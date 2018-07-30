package main

import (
	"github.com/henrylee2cn/faygo"
	"wallet/router"
)

func main() {
	router.Route(faygo.New("wallet"))
	faygo.Run()
}
