package commands

type Command struct {
	Name, Short, Long string
	Cmd               interface{}
}

var Commands = map[string]*Command{}

func Add(name, short, long string, cmd interface{}) {
	Commands[name] = &Command{name, short, long, cmd}
}
