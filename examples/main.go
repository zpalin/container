package main

import (
	"fmt"
	"github.com/zpalin/container"
)

type App struct {
	UsrSvc UserService
}

func (a *App) Run() {
	users := a.UsrSvc.ListAll()
	fmt.Printf("Users: %+v\n", users)
}

func Start(svc UserService) {
	svc.Create("World")
	users := svc.ListAll()
	fmt.Printf("Users: %+v\n", users)
}

func main() {
	c := container.New()
	// Register any number of types to be wired up
	c.Register(ConcreteUserService{})

	// Register specific reference, will not be wired but will be used as dep
	c.Register(&InMemoryUserStore{
		users: []string{"Hello"},
	})

	// Can hand container arbitrary function and it will inject the types of the signature
	c.Exec(Start)

	// Can also hand a reference to a Runnable object, will be wired up as usual
	// and then it will have .Run() called on it.
	c.Run(&App{})
}