package cypress

import "io"

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
