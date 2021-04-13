package main

func (m *Mapler) Maple(input string) error {
	m.Emit("1", input)
	return nil
}
