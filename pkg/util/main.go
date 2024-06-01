package util

import (
	"fmt"
	"math/rand"
	"time"
)


func RandomSleep(minSecond int, maxSecond int, prefix string) {
	randSecond := rand.Intn(maxSecond-minSecond) + minSecond
	fmt.Printf("%s sleeping for %d seconds\n", prefix, randSecond)
	time.Sleep(time.Duration(randSecond) * time.Second)
}