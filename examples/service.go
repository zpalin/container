package main

import "fmt"

type UserService interface {
	ListAll() []string
	Create(u string)
}

type ConcreteUserService struct {
	store UserStore
}

func (svc *ConcreteUserService) ListAll() []string {
	return svc.store.GetAll()
}

func (svc *ConcreteUserService) Create(u string) {
	svc.store.Save(u)
}

func (svc *ConcreteUserService) New(store UserStore) {
	fmt.Println("Constructing ConcreteUserService")
	svc.store = store
}

func (svc *ConcreteUserService) Init() {
	fmt.Println("Init ConcreteUserService")
}