## Container
A reflection-based runtime Dependency Injection container for Go.

### Example Use
See `examples` directory for full example with types.
```go
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

	// Run and Exec also have async versions that will spawn work on goroutine
	c.ExecAsync(Start)

	// Call .Wait() to block main thread until background operations are complete
	c.Wait()
}

// Public members will be set by the container during wiring
type App struct {
	UsrSvc UserService
}

// Implements the Runnable interface, so it can be `.Run(runnable)` by the container
func (a *App) Run() {
	users := a.UsrSvc.ListAll()
	fmt.Printf("Users: %+v\n", users)
}

type UserService interface {
	ListAll() []string
	Create(u string)
}

type ConcreteUserService struct {
	store UserStore
}

// You can also define a constructor method named `New` to configure
// your deps during wiring and allow private members to be set.
func (svc *ConcreteUserService) New(store UserStore) {
	fmt.Println("Constructing ConcreteUserService")
	svc.store = store
}

// If you define a method `Init` on your dep, it will be called ~after~
// wiring is done.
func (svc *ConcreteUserService) ListAll() []string {
	return svc.store.GetAll()
}

func (svc *ConcreteUserService) Create(u string) {
	svc.store.Save(u)
}

// Arbitrary functions can also be used for `.Exec(func)` by the container. Params
// will be injected by the container.
func Start(svc UserService) {
	svc.Create("World")
	users := svc.ListAll()
	fmt.Printf("Users: %+v\n", users)
}
```