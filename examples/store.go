package main

type UserStore interface {
	GetAll() []string
	Save(u string)
}

type InMemoryUserStore struct {
	users []string
}

func (s *InMemoryUserStore) GetAll() []string {
	return s.users
}

func (s *InMemoryUserStore) Save(u string) {
	s.users = append(s.users, u)
}