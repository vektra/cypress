package cypress

import "io"

// Read messages from gen and send them to recv
func Glue(gen Generator, recv Receiver) error {
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
