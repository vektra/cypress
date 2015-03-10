package cypress

import "io"

type Probe struct {
	r   io.Reader
	hdr StreamHeader

	Stream io.Reader
}

func NewProbe(r io.Reader) *Probe {
	return &Probe{
		r:      r,
		Stream: r,
	}
}

func (p *Probe) Probe() error {
	buf := pbBufPool.Get().([]byte)

	_, err := p.r.Read(buf[:1])
	if err != nil {
		pbBufPool.Put(buf)
		return err
	}

	b := buf[0]
	pbBufPool.Put(buf)

	if b != '-' {
		p.Stream = &bytePeekReader{b: b, r: p.Stream}
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
