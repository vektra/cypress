package cypress

import (
	"io"
	"os"
)

// Read messages from gen and send them to recv
func Glue(gen Generator, recv Receiver) error {
	defer recv.Close()
	defer gen.Close()

	for {
		m, err := gen.Generate()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}

		err = recv.Receive(m)
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}
	}

	return nil
}

// Read messages from gen and send them to recv
func GlueFiltered(gen Generator, filt Filterer, recv Receiver) error {
	defer recv.Close()
	defer gen.Close()

	for {
		m, err := gen.Generate()
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}

		m, err = filt.Filter(m)
		if err != nil {
			return err
		}

		if m == nil {
			continue
		}

		err = recv.Receive(m)
		if err != nil {
			if err == io.EOF {
				return nil
			}

			return err
		}
	}

	return nil
}

func StandardStreamFilter(f Filterer) error {
	dec, err := NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	enc := NewStreamEncoder(os.Stdout)

	return GlueFiltered(dec, f, enc)
}
