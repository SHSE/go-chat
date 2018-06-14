package main

import (
	"fmt"
	"os"
	"testing"
)

func noopReceiver() chan<- string {
	events := make(chan string)

	go func() {
		for range events {
		}
	}()

	return events
}

func newTestSession() (*Session, error) {
	return NewSession(os.Getenv("SERVER_URL"), noopReceiver())
}

func BenchmarkSendCommand(bench *testing.B) {
	session, err := newTestSession()

	if err != nil {
		panic(err)
	}

	defer session.Close()

	_, err = session.SendCommand("join", []string{fmt.Sprintf("%d", 1)})

	if err != nil {
		panic(err)
	}

	bench.ResetTimer()

	for i := 0; i < bench.N; i++ {
		_, err = session.SendCommand("say", []string{"hi"})
	}
}

func BenchmarkConcurrent(bench *testing.B) {
	bench.SetParallelism(20)

	bench.RunParallel(func(pb *testing.PB) {
		session, err := newTestSession()

		if err != nil {
			panic(err)
		}

		_, err = session.SendCommand("join", []string{fmt.Sprintf("%d", 1)})

		if err != nil {
			panic(err)
		}

		for pb.Next() {
			_, err = session.SendCommand("say", []string{"hi"})

			if err != nil {
				panic(err)
			}
		}

		session.Close()
	})
}

func BenchmarkConcurrentSingleClient(bench *testing.B) {
	bench.SetParallelism(1000)

	session, err := newTestSession()

	if err != nil {
		panic(err)
	}

	_, err = session.SendCommand("join", []string{fmt.Sprintf("%d", 1)})

	if err != nil {
		panic(err)
	}

	bench.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err = session.SendCommand("say", []string{"hi"})

			if err != nil {
				panic(err)
			}
		}
	})

	session.Close()
}
