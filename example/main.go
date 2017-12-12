package main

import (
	"log"

	"github.com/prizem-io/routerstore"
)

// Service is the representation of an API or microservice.
type Service struct {
	Name    string
	Version string
}

func main() {
	helloService := Service{
		Name:    "Hello",
		Version: "V1",
	}
	router := routerstore.New()
	router.GET("/people", &helloService)
	router.GET("/people/:id", &helloService)
	router.GET("/people/:id/tasks", &helloService)
	router.GET("/people/:last/:first", &helloService)

	var result routerstore.Result
	var err error
	var service *Service

	log.Printf("GET /people/1234")
	err = router.Match(routerstore.GET, "/people/1234", &result)
	service = checkResult(&result, err)
	log.Printf("\tService %s, Version %s", service.Name, service.Version)
	log.Printf("\tPerson ID: %s", result.Param("id"))

	log.Printf("GET /people/doe/john")
	err = router.Match(routerstore.GET, "/people/doe/john", &result)
	service = checkResult(&result, err)
	log.Printf("\tService %s, Version %s", service.Name, service.Version)
	log.Printf("\tPerson Name: %s %s", result.Param("first"), result.Param("last"))
}

func checkResult(result *routerstore.Result, err error) *Service {
	if err != nil {
		log.Fatalf("could not match: %v", err)
	}
	service, ok := result.Data.(*Service)
	if !ok {
		log.Fatal("Unexpected result data")
	}
	return service
}
