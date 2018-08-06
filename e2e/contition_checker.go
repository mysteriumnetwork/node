package e2e

import (
	"errors"
	"time"
)

type conditionChecker func() (bool, error)

func waitForCondition(checkFunc conditionChecker) error {
	for i := 0; i < 10; i++ {
		state, err := checkFunc()
		switch {
		case err != nil:
			return err
		case state:
			return nil
		case !state:
			time.Sleep(1 * time.Second)
		}
	}
	return errors.New("state was still false")
}
