package response

type itemTooFew struct{}

func ItemTooFew() *itemTooFew {
	return &itemTooFew{}
}

func (c *itemTooFew) String() string {
	return "TOOFEW"
}
