package main

import (
	"fmt"

	"github.com/kamioair/qf"
)

func main() {
	ctx, _ := qf.NewContext(map[string]any{"name": "111"}, nil, nil)

	s := ctx.Get[A]("")
	fmt.Println(s)
}

type A struct {
	Name string
}
