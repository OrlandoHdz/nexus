package tasks

import (
	"fmt"
	"log"
)

// Task es la interfaz que todos los procesos deben implementar
type Task interface {
	Execute() error
	Name() string
}

// Ejemplo de una tarea específica
type HelloTask struct{}

func (t *HelloTask) Name() string {
	return "hello-task"
}

func (t *HelloTask) Execute() error {
	log.Println("Ejecutando Nexus: Proceso de saludo...")
	fmt.Println("¡Hola desde Nexus!")
	return nil
}
