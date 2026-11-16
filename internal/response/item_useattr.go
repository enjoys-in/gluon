package response

type itemUseAttr struct{}

func ItemUseAttr() *itemUseAttr {
	return &itemUseAttr{}
}

func (c *itemUseAttr) String() string {
	return "USEATTR"
}
