package token

type Token struct {
	value string
}

func (t Token) Value() string {
	return t.value
}
