package codec

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int
	CodeType Type
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodeType:GobType,
}