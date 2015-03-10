package cypress

import "io"

type Probe struct {
	r   io.Reader
	buf []byte
	hdr StreamHeader

	Stream io.Reader
}

func NewProbe(r io.Reader) *Probe {
	return &Probe{
		r:      r,
		buf:    make([]byte, 128),
		Stream: r,
	}
}

func (p *Probe) Probe() error {
	_, err := p.r.Read(p.buf[:1])
	if err != nil {
		return err
	}

	if p.buf[0] != '-' {
		p.Stream = &bytePeekReader{b: p.buf[0], r: p.Stream}
		return nil
	}

	return p.hdr.UnmarshalFrom(p.r)
}

func (p *Probe) Compression() StreamHeader_Compression {
	return p.hdr.GetCompression()
}

func (p *Probe) Reader() io.Reader {
	return ReadCompressed(p.Stream, p.Compression())
}

func (p *Probe) Writer(w io.Writer) io.Writer {
	return WriteCompressed(w, p.Compression())
}
