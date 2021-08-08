package main

import (
	"context"
	"testing"
	"time"

	"github.com/golang/glog"
)

func TestStack(t *testing.T) {
	ctx := context.Background()
	//cancelCtx := context.WithCancel(ctx)
	deadlineCtx, cancel := context.WithTimeout(ctx, time.Minute)
	finishedChan := make(chan struct{})
	targetDir := t.TempDir()
	go func(done chan<- struct{}) {
		_ = cancel

		monkeyNameChan := make(chan string, 100)
		for i := uint(0); i < 10; i++ {
			go generateFlamingMonkeys(deadlineCtx, 10000, targetDir, "png", monkeyNameChan)
		}
		var monkeyHeadCount uint64
	main:
		for {
			select {
			case <-deadlineCtx.Done():
				glog.Info("Ended Monkeygedon")
				glog.Infof("Found a total of %d monKeys", monkeyHeadCount)
				break main
			case monkeyName, ok := <-monkeyNameChan:
				if !ok {
					break main
				}
				monkeyHeadCount++
				glog.Infof("Say hi to %s", monkeyName)
			}
		}
		done <- struct{}{}
	}(finishedChan)

	<-finishedChan
}
