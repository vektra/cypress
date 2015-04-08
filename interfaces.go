package cypress

// A core interface, represending a type that can take a Message
type Receiver interface {
	Receive(msg *Message) error
}

// A core interface, representing a type that can create a Message
type Generator interface {
	Generate() (*Message, error)
	Close() error
}

// Use to allow types to handle new Generators as they're created
type GeneratorHandler interface {
	HandleGenerator(g Generator)
}

// A GeneratorHandler that just calls itself as a function
type GeneratorHandlerFunc func(g Generator)

func (f GeneratorHandlerFunc) HandleGenerator(g Generator) {
	f(g)
}

// Used by Send to allow a sender to interact with the Message
// transmit and ack lifecycle
type SendRequest interface {
	Ack(*Message)
	Nack(*Message)
}
