package events

type Event struct {
	Type string
	Data map[string]string
}

type Subscriber func(Event)

type Broker struct {
	subs map[string][]Subscriber
}

func NewBroker() *Broker {
	return &Broker{subs: make(map[string][]Subscriber)}
}

func (b *Broker) Subscribe(eventType string, fn Subscriber) {
	b.subs[eventType] = append(b.subs[eventType], fn)
}

func (b *Broker) Publish(event Event) {
	if handlers, ok := b.subs[event.Type]; ok {
		for _, handler := range handlers {
			go handler(event) // async
		}
	}
}
