package stream_test

import (
	"errors"
	"testing"
	"time"

	"github.com/biome/agent-core/packages/stream"
)

func TestEventStreamBasicFlow(t *testing.T) {
	s := stream.NewEventStream[string, int]()

	go func() {
		s.Push("event1")
		s.Push("event2")
		s.Push("event3")
		s.End(42)
	}()

	events := []string{}
	for event := range s.Events() {
		events = append(events, event)
	}

	if len(events) != 3 {
		t.Errorf("Expected 3 events, got %d", len(events))
	}

	expectedEvents := []string{"event1", "event2", "event3"}
	for i, e := range events {
		if e != expectedEvents[i] {
			t.Errorf("Event %d: expected %s, got %s", i, expectedEvents[i], e)
		}
	}

	result, err := s.Result()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("Expected result 42, got %d", result)
	}
}

func TestEventStreamMultipleResultCalls(t *testing.T) {
	s := stream.NewEventStream[string, string]()

	go func() {
		s.Push("hello")
		s.End("final result")
	}()

	for range s.Events() {
	}

	result1, err1 := s.Result()
	result2, err2 := s.Result()
	result3, err3 := s.Result()

	if err1 != nil || err2 != nil || err3 != nil {
		t.Error("Expected no errors on multiple Result() calls")
	}

	if result1 != "final result" || result2 != "final result" || result3 != "final result" {
		t.Errorf("Results don't match: %s, %s, %s", result1, result2, result3)
	}
}

func TestEventStreamError(t *testing.T) {
	s := stream.NewEventStream[int, string]()

	go func() {
		s.Push(1)
		s.Push(2)
		s.EndWithError(errors.New("test error"))
	}()

	count := 0
	for range s.Events() {
		count++
	}

	if count != 2 {
		t.Errorf("Expected 2 events before error, got %d", count)
	}

	_, err := s.Result()
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != "test error" {
		t.Errorf("Expected 'test error', got '%s'", err.Error())
	}
}

func TestEventStreamPushAfterEnd(t *testing.T) {
	s := stream.NewEventStream[string, int]()

	s.Push("event1")
	s.End(100)
	s.Push("event2") // Should be ignored

	events := []string{}
	for e := range s.Events() {
		events = append(events, e)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event (second should be ignored), got %d", len(events))
	}
	if events[0] != "event1" {
		t.Errorf("Expected 'event1', got '%s'", events[0])
	}
}

func TestEventStreamConcurrentProducers(t *testing.T) {
	s := stream.NewEventStream[int, string]()

	for i := 0; i < 5; i++ {
		go func(id int) {
			s.Push(id)
		}(i)
	}

	time.Sleep(100 * time.Millisecond)
	s.End("done")

	count := 0
	for range s.Events() {
		count++
	}

	if count != 5 {
		t.Errorf("Expected 5 events from concurrent producers, got %d", count)
	}

	result, _ := s.Result()
	if result != "done" {
		t.Errorf("Expected result 'done', got '%s'", result)
	}
}

func TestEventStreamResultBlocksUntilEnd(t *testing.T) {
	s := stream.NewEventStream[string, int]()

	resultReceived := false
	done := make(chan bool)

	go func() {
		result, err := s.Result()
		if err != nil || result != 99 {
			t.Errorf("Expected result 99, got %d (err: %v)", result, err)
		}
		resultReceived = true
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)

	if resultReceived {
		t.Error("Result() returned before End() was called!")
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.End(99)
	}()

	select {
	case <-done:
		// Success!
	case <-time.After(1 * time.Second):
		t.Error("Result() didn't unblock after End()")
	}

	if !resultReceived {
		t.Error("Result() never received the value")
	}
}

func TestEventStreamEmpty(t *testing.T) {
	s := stream.NewEventStream[string, bool]()

	go func() {
		s.End(true)
	}()

	count := 0
	for range s.Events() {
		count++
	}

	if count != 0 {
		t.Errorf("Expected 0 events, got %d", count)
	}

	result, err := s.Result()
	if err != nil || result != true {
		t.Errorf("Expected result true, got %v (err: %v)", result, err)
	}
}

type CustomEvent struct {
	ID      int
	Message string
}

type CustomResult struct {
	Count  int
	Status string
}

func TestEventStreamWithCustomTypes(t *testing.T) {
	s := stream.NewEventStream[CustomEvent, CustomResult]()

	go func() {
		s.Push(CustomEvent{ID: 1, Message: "hello"})
		s.Push(CustomEvent{ID: 2, Message: "world"})
		s.End(CustomResult{Count: 2, Status: "complete"})
	}()

	events := []CustomEvent{}
	for e := range s.Events() {
		events = append(events, e)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	result, _ := s.Result()
	if result.Count != 2 || result.Status != "complete" {
		t.Errorf("Unexpected result: %+v", result)
	}
}
