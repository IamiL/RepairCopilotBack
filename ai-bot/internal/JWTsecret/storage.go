package JWTsecret

type JWTSecret struct {
	secret []byte
}

func NewJWTSecret(secret []byte) *JWTSecret {
	return &JWTSecret{secret: secret}
}

func (s *JWTSecret) Secret() []byte {
	return s.secret
}
