package stream

type EventStream[T, R any] struct {
	events		chan T
	resultChan 	chan R
	doneChan	chan struct{}
	resultValue R
	err			error
	closed		bool
}

func NewEventStream[T, R any]() *EventStream[T, R] {
	return &EventStream[T, R]{
		events:		make(chan T, 10),
		resultChan: make(chan R, 1),
		doneChan: 	make(chan struct{}),
	}
}

func (es *EventStream[T, R]) Push(event T) {
	select {
	case es.events <- event:
	case <-es.doneChan:
	}
}

func (es *EventStream[T, R]) End(result R) {
	if es.closed {
		return
	}
	es.closed = true
	es.resultValue = result

	close(es.events)
	es.resultChan <- result
	close(es.doneChan)
}

func (es *EventStream[T, R]) EndWithError(err error) {
	if es.closed {
		return
	}
	es.closed = true
	es.err = err

	close(es.events)
	close(es.doneChan)
}

func (es *EventStream[T, R]) Events() <-chan T {
	return es.events
}

func (es *EventStream[T, R]) Result() (R, error) {
	<-es.doneChan

	if es.err != nil {
		var zero R
		return zero, es.err
	}
	return es.resultValue, nil
}