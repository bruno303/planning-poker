package main

import "planning-poker/internal/planningpoker"

type Container struct {
	Hub *planningpoker.Hub
}

func NewContainer() *Container {
	return &Container{
		Hub: planningpoker.NewHub(),
	}
}
