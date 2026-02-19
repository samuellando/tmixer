package tmux

import (
	"fmt"
	"time"
)

func timeout(done func() (bool, error)) error {
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			d, err := done()
			if err != nil {
				return err
			} else if d {
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("timeout")
		}
	}
}
