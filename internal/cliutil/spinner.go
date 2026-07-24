package cliutil

import (
	"fmt"
	"sync"
	"time"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner represents a simple CLI loader.
type Spinner struct {
	message   string
	startTime time.Time
	mu        sync.Mutex
	stop      chan struct{}
	wg        sync.WaitGroup
}

// StartSpinner starts a loader with a message.
func StartSpinner(message string) *Spinner {
	s := &Spinner{
		message:   message,
		startTime: time.Now(),
		stop:      make(chan struct{}),
	}
	
	// Print cursor hide
	fmt.Print("\033[?25l")
	
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		i := 0
		for {
			select {
			case <-s.stop:
				return
			default:
				s.mu.Lock()
				msg := s.message
				s.mu.Unlock()
				
				elapsed := time.Since(s.startTime).Seconds()
				timeStr := StyleSubtext.Render(fmt.Sprintf("(%.1fs)", elapsed))
				
				frame := StyleHighlight.Render(spinnerFrames[i%len(spinnerFrames)])
				fmt.Printf("\r%s %s %s", frame, msg, timeStr)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
	return s
}

// Stop stops the spinner and clears the line.
func (s *Spinner) Stop() {
	close(s.stop)
	s.wg.Wait()
	// Clear the line and show cursor
	fmt.Print("\r\033[K\033[?25h")
}

// UpdateMessage changes the spinner message.
func (s *Spinner) UpdateMessage(newMessage string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Clear the line so longer messages don't leave trailing chars
	fmt.Print("\r\033[K")
	s.message = newMessage
}
