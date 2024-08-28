package limiter

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

func New(filename string, delay time.Duration) *Limiter {
	return &Limiter{
		filename: filename,
		delay:    delay,
	}
}

type Limiter struct {
	filename string
	delay    time.Duration
	nextAt   time.Time
}

func (lim *Limiter) Load() error {
	if _, err := os.Stat(lim.filename); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("error statting file: %w", err)
	}

	bs, err := os.ReadFile(lim.filename)
	if err != nil {
		return err
	}

	lim.nextAt, err = time.Parse(time.UnixDate, string(bs))
	if err != nil {
		return err
	}

	return nil
}

func (lim *Limiter) Wait(ctx context.Context) error {
	if !lim.nextAt.IsZero() {
		now := time.Now()
		dur := lim.nextAt.Sub(now)
		if dur > time.Second {
			log.Printf("waiting %s until %s",
				lim.nextAt.Sub(now).Truncate(time.Second),
				lim.nextAt.Format(time.StampMilli))
		}

	wait:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(dur):
			break wait
		}

		if err := os.Remove(lim.filename); err != nil &&
			!errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	return nil
}

func (lim *Limiter) SetNextAt(secondsStr string) error {
	if secondsStr == "" {
		secondsStr = "60"
	}
	seconds, err := strconv.ParseInt(secondsStr, 10, 64)
	if err != nil {
		return err
	}
	var nextReqAt time.Time
	waitTime := time.Duration(seconds)*time.Second + time.Second
	lim.nextAt = time.Now().Add(waitTime)
	if err := os.WriteFile(lim.filename, []byte(nextReqAt.Format(time.UnixDate)), 0666); err != nil {
		return err
	}
	return nil
}

func (lim *Limiter) Delay() {
	lim.DelayBy(lim.delay)
}

func (lim *Limiter) DelayBy(time.Duration) {
	lim.nextAt = time.Now().Add(lim.delay)
}
