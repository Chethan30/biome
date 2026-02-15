package core

type FollowUpQueue struct {
	items []FollowUpItem
}

type FollowUpItem struct {
	Type    string
	Payload interface{}
}

func NewFollowUpQueue() *FollowUpQueue {
	return &FollowUpQueue{
		items: make([]FollowUpItem, 0),
	}
}

func (q *FollowUpQueue) Enqueue(item FollowUpItem) {
	q.items = append(q.items, item)
}

func (q *FollowUpQueue) Dequeue() (FollowUpItem, bool) {
	if len(q.items) == 0 {
		return FollowUpItem{}, false
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

func (q *FollowUpQueue) IsEmpty() bool {
	return len(q.items) == 0
}

func (q *FollowUpQueue) Size() int {
	return len(q.items)
}

func (q *FollowUpQueue) Clear() {
	q.items = make([]FollowUpItem, 0)
}

// Drain returns all remaining items and clears the queue (for steering interrupt / skip remaining).
func (q *FollowUpQueue) Drain() []FollowUpItem {
	rem := q.items
	q.items = make([]FollowUpItem, 0)
	return rem
}