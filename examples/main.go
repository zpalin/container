package main

import (
	"fmt"
	"github.com/zpalin/container"
)

type App struct {
	UsrSvc UserService
}

// Implements the Runnable interface, so it can be run by the container
func (a *App) Run() {
	users := a.UsrSvc.ListAll()
	fmt.Printf("Users: %+v\n", users)
}

// Arbitrary functions can also be used for `.Exec(fn)` by the container. Params
// will be injected by the container.
func SeedUsers(svc UserService, c container.Container) {
	svc.Create("Bob")
	svc.Create("Carl")

	fmt.Printf("Container: %+v\n", c) // Reference to the container
}

func main() {
	c := container.New()
	// Register any number of types to be wired up
	c.Register(ConcreteUserService{})

	// Register specific reference, will not be wired but will be used as dep
	c.Register(&InMemoryUserStore{
		users: []string{"Alice"},
	})

	// Can hand container arbitrary function and it will inject the types of the signature
	c.Exec(SeedUsers)

	// Can also hand a reference to a Runnable object, will be wired up as usual
	// and then it will have .Run() called on it.
	c.Run(&App{}) // prints [Alice Bob Carl]

	// Run and Exec also have async versions that will spawn work on goroutine
	c.ExecAsync(SeedUsers)

	// Call .Wait() to block main thread until background operations are complete
	c.Wait()
}