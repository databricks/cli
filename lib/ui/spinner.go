package ui

import (
	"context"
	"math/rand"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// https://github.com/leaanthony/spinner alternative

func Spinner(label string, fn func(*spinner.Spinner) error, final string) error {
	rand.Seed(time.Now().UnixMilli())
	s := spinner.New(spinner.CharSets[rand.Intn(11)], 200*time.Millisecond)
	_ = s.Color("green")
	s.Start()
	s.Prefix = " "
	s.Suffix = " " + label

	err := fn(s)
	if err == nil {
		s.FinalMSG = color.GreenString(" ✓ %s", final) // or ✓
	} else {
		s.FinalMSG = color.RedString(" ✗ %s", err) // or
	}
	s.Stop()
	println("")
	return err
}

type Stage struct {
	InProgress string
	Callback   func(context.Context, func(string)) error
	Complete   string
}

func SpinStages(ctx context.Context, stages []Stage) error {
	for _, v := range stages {
		err := Spinner(v.InProgress, func(s *spinner.Spinner) error {
			updateMsg := func(msg string) {
				// TODO: print to stdout for non-tty
				s.Suffix = " " + msg
			}
			return v.Callback(ctx, updateMsg)
		}, v.Complete)
		if err != nil {
			return err
		}
	}
	return nil
}
