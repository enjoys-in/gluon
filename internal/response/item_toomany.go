package response

type itemTooMany struct{}

func ItemTooMany() *itemTooMany {
	return &itemTooMany{}
}

func (c *itemTooMany) String() string {
	return "TOOMANY"
}
