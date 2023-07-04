package main

import (
	"fmt"
	"math/rand"
	"time"
)

var (
	all []string
)

func main() {
	all := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
	rand.Seed(time.Now().Unix())
	alls := len(all)
	fmt.Println(alls)
	rnd := rand.Intn(len(all))
	fmt.Println(rnd)
	if rnd >= (alls / 2) {
		uploop(rnd, alls)
	} else {
		downloop(rnd, alls)
	}
}

func uploop(rnd, alls int) {
	cnt := alls
	for i := rnd; i <= len(all); i++ {
		fmt.Println(all[i])
		cnt = cnt - 1
	}

	for i := 0; i <= cnt; i++ {
		fmt.Println(all[i])
	}
}

func downloop(rnd, alls int) {
	i := rnd
	for {
		fmt.Println(all[i])
		i = i - 1
		if i == 0 {
			break
		}
	}

	count := alls - rnd
	for i := 0; i <= count; i++ {
		fmt.Println(all[alls-i])
	}
}
