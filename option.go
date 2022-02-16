package dayu

import "time"

type Option func(p *Pool) error

func Expire(expire int) Option {
	return func(p *Pool) error {
		if expire < 0 {
			return ErrInvalidPoolExpire
		}
		p.expire = time.Duration(expire) * time.Second
		return nil
	}
}

func Size(size int) Option {
	return func(p *Pool) error {
		if size < 0 {
			return ErrInvalidPoolSize
		}
		p.capacity = size
		return nil
	}
}
