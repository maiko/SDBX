package tui

import (
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Spinner provides an animated spinner for long-running operations
type Spinner struct {
	message   string
	frames    []string
	interval  time.Duration
	current   int
	running   bool
	done      chan struct{}
	mu        sync.Mutex
	style     lipgloss.Style
	doneStyle lipgloss.Style
}

// NewSpinner creates a new spinner with a message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message:   message,
		frames:    SpinnerFrames,
		interval:  80 * time.Millisecond,
		style:     lipgloss.NewStyle().Foreground(ColorPrimary),
		doneStyle: SuccessStyle,
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.done = make(chan struct{})
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				s.mu.Lock()
				s.current = (s.current + 1) % len(s.frames)
				frame := s.frames[s.current]
				s.mu.Unlock()

				// Clear line and print spinner
				fmt.Printf("\r%s %s", s.style.Render(frame), s.message)
			}
		}
	}()
}

// Stop stops the spinner and shows completion
func (s *Spinner) Stop(success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.done)

	// Clear line
	fmt.Print("\r\033[K")

	if success {
		fmt.Printf("%s %s\n", s.doneStyle.Render(IconSuccess), s.message)
	} else {
		fmt.Printf("%s %s\n", ErrorStyle.Render(IconError), s.message)
	}
}

// StopWithMessage stops the spinner with a custom message
func (s *Spinner) StopWithMessage(success bool, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.done)

	// Clear line
	fmt.Print("\r\033[K")

	if success {
		fmt.Printf("%s %s\n", s.doneStyle.Render(IconSuccess), message)
	} else {
		fmt.Printf("%s %s\n", ErrorStyle.Render(IconError), message)
	}
}

// UpdateMessage updates the spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// Task represents a task with spinner
type Task struct {
	Name    string
	Action  func() error
}

// RunTasks runs multiple tasks with spinners sequentially
func RunTasks(tasks []Task) error {
	for _, task := range tasks {
		spinner := NewSpinner(task.Name)
		spinner.Start()

		err := task.Action()

		if err != nil {
			spinner.StopWithMessage(false, fmt.Sprintf("%s: %v", task.Name, err))
			return err
		}
		spinner.Stop(true)
	}
	return nil
}

// RunWithSpinner runs a function with a spinner
func RunWithSpinner(message string, fn func() error) error {
	spinner := NewSpinner(message)
	spinner.Start()
	err := fn()
	spinner.Stop(err == nil)
	return err
}
