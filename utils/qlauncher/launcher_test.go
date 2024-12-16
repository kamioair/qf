package qlauncher

import (
	"fmt"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	go func() {
		time.Sleep(time.Second * 5)
		Exit()
	}()
	Run(start, stop)
	fmt.Println("finish")
}

func start() {
	fmt.Println("start1")
	time.Sleep(time.Second)
	fmt.Println("start2")
	time.Sleep(time.Second)
}

func stop() {
	fmt.Println("stop")
	time.Sleep(time.Second)
}
