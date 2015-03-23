package cypress

import (
	"bufio"
	"io"
	"os/exec"
)

type Run struct {
	request string
	cmd     *exec.Cmd
	stdout  io.ReadCloser
	buf     *bufio.Reader
}

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

func (r *Run) Close() error {
	r.stdout.Close()
	return r.cmd.Wait()
}
