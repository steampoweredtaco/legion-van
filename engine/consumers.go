package engine

import (
	"context"
	"sync"
)

func ProcessMonkeyData(ctx context.Context, targetDir string, targetFormat string, monkeyDataChan <-chan MonkeyStats, monkeyNameChan chan<- string, monkeyDisplayChan chan<- MonkeyStats, wg *sync.WaitGroup) {
	defer wg.Done()
	outputMonkeyChan := make(chan MonkeyStats, 100)

	go func() {
		writeMonkeyWG := new(sync.WaitGroup)
		for i := 0; i < 10; i++ {
			writeMonkeyWG.Add(1)
			go func() {
				outputMonkeyData(ctx, targetDir, targetFormat, outputMonkeyChan)
				writeMonkeyWG.Done()
			}()
		}
		writeMonkeyWG.Wait()
		// All the writers are done, so chanel needs to close
		close(outputMonkeyChan)
	}()

main:
	for {
		select {
		case <-ctx.Done():
			break main
		case monkey, ok := <-monkeyDataChan:
			if !ok {
				break main
			}
			monkeyNameChan <- monkey.SillyName
			outputMonkeyChan <- monkey
			if monkeyDisplayChan != nil {
				wg.Add(1)
				go func() {
					defer wg.Done()
					select {
					case monkeyDisplayChan <- monkey:
					case <-ctx.Done():
						return
					}
				}()
			}

		}
	}
}
