package main

import (
	"fmt"
	"reflect"
	"sync"
)

const (
	eatReq = iota
	finishNotice

	numPhilos = 5
	rounds    = 3
)

type Chopstick struct {
	sync.Mutex
}

type Philosopher struct {
	number int
	start  chan int  // for eatReq
	finish chan int  // for finishNotice
	answer chan bool // permit to eat
	left   *Chopstick
	right  *Chopstick
}

func (p Philosopher) eat(wg *sync.WaitGroup) {
	for i := 0; i < rounds; i++ {
		for {
			p.start <- eatReq
			// will block until host answer yes or no
			ok := <-p.answer
			if !ok {
				// continue to raise eat request until success
				continue
			}

			// the order of acquiring locks should not matter
			p.right.Lock()
			p.left.Lock()
			fmt.Println("start to eat", p.number)

			fmt.Println("finishing eating", p.number)
			p.left.Unlock()
			p.right.Unlock()

			// this will block until host receives the notice
			p.finish <- finishNotice
			break
		}
	}
	wg.Done()
}

func main() {
	wg := sync.WaitGroup{}

	philos := setUpTable()
	for i := 0; i < numPhilos; i++ {
		wg.Add(1)
		go philos[i].eat(&wg)
	}

	// Fire up a host goroutine to mediate this dinner
	go host(philos)

	wg.Wait()
}

func host(philos []*Philosopher) {
	// the host has a book which keeps the eating status of each philosopher
	var book [numPhilos]bool

	startCases := make([]reflect.SelectCase, numPhilos)
	finishCases := make([]reflect.SelectCase, numPhilos)
	for i, p := range philos {
		startCases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(p.start),
		}

		finishCases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(p.finish),
		}
	}

	// always put default cases because we don't want the select to become fully blocked.
	startCases = append(startCases, reflect.SelectCase{Dir: reflect.SelectDefault})
	finishCases = append(finishCases, reflect.SelectCase{Dir: reflect.SelectDefault})

	for {
		// process messages from finish channels has higher priority because
		// we need to free up the occupied book slots asap.
		chosen, _, ok := reflect.Select(finishCases)
		if ok {
			book[chosen] = false
			continue
		}

		chosen, _, ok = reflect.Select(startCases)
		if ok {
			if allowToEat(book) {
				book[chosen] = true
				philos[chosen].answer <- true
				continue
			} else {
				philos[chosen].answer <- false
				continue
			}
		}
	}
}

func allowToEat(book [numPhilos]bool) bool {
	count := 0
	for _, b := range book {
		if b {
			count++
		}
	}
	return count < 2
}

///*
//        c4 p0
//     p4       c0
//  c3             p1
//     p3       c1
//        c2 p2
//*/
func setUpTable() []*Philosopher {
	chops := make([]*Chopstick, numPhilos)
	philos := make([]*Philosopher, numPhilos)

	for i := range chops {
		chops[i] = new(Chopstick)
	}

	for i := range philos {
		philos[i] = &Philosopher{
			number: i + 1,
			start:  make(chan int),
			finish: make(chan int),
			answer: make(chan bool),
			left:   chops[i],
			right:  chops[(i+4)%numPhilos],
		}
	}
	return philos
}
