package cypress

import "io"

// A type which can look at a reader and detect the format
type Probe struct {
	r io.Reader

	Header *StreamHeader
	Stream io.Reader
}

// Create a new Probe from the data in r
func NewProbe(r io.Reader) *Probe {
	return &Probe{
		r:      r,
		Stream: r,
		Header: new(StreamHeader),
	}
}

// Inspect the stream and populate Header with the data
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

	return p.Header.UnmarshalFrom(p.r)
}

// Indicate the compression in use
func (p *Probe) Compression() StreamHeader_Compression {
	return p.Header.GetCompression()
}

// Create an io.Reader for the remainder of the stream
func (p *Probe) Reader() io.Reader {
	return ReadCompressed(p.Stream, p.Compression())
}

// Create an io.Writer that will match the parameters of the probed
// stream.
func (p *Probe) Writer(w io.Writer) io.Writer {
	return WriteCompressed(w, p.Compression())
}
