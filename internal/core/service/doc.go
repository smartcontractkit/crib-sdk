// Package service provides glue code for the core service layer. Services defined here
// should implement the business logic of the application and connect adapters, other services, and repositories.
// Typically, a service should explicitly implement an interface defined in the core/port package and it is
// recommended to explicitly assert against the interface in the service constructor.
//
// Services should generally be stateless and should not hold any state that is not passed in as a parameter.
// This allows services to be easily tested and reused in different contexts. Under DI, this is "lazily" injected.
package service
