package cypress

import (
	"bufio"
	"io"
	"os/exec"
)

// A type which runs a command and generates messages from the commands
// standard output.
type Run struct {
	request string
	cmd     *exec.Cmd
	stdout  io.ReadCloser
	buf     *bufio.Reader
}

// Create a new Run for the given program with arguments.
func NewRun(prog string, args ...string) (*Run, error) {
	c := exec.Command(prog, args...)

	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = c.Start()
	if err != nil {
		return nil, err
	}

	buf := bufio.NewReader(stdout)

	return &Run{request: prog, cmd: c, stdout: stdout, buf: buf}, nil
}

// Generate a Message from the programs next output line.
func (r *Run) Generate() (*Message, error) {
	str, err := r.buf.ReadString('\n')
	if err != nil {
		return nil, err
	}

	m := Log()
	m.Add("command", r.request)
	m.Add("message", str[:len(str)-1])

	return m, nil
}

// Close the output from the program and wait for it to finish.
func (r *Run) Close() error {
	r.stdout.Close()
	return r.cmd.Wait()
}
