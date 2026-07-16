package token

func NewToken(value string) (Token, error) {
	if value == "" {
		return Token{}, ErrInvalidTokenValue
	}
	return Token{value: value}, nil
}
