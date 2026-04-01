package main

import (
	"flag"
	"log"
	"os"

	"github.com/OrlandoHdz/nexus/internal/app"
)

func main() {
	// Definimos un flag llamado "task"
	taskName := flag.String("task", "", "Nombre de la tarea a ejecutar")
	flag.Parse()

	if *taskName == "" {
		log.Fatal("Error: Debes especificar una tarea con el flag -task")
		os.Exit(1)
	}

	err := app.RunTask(*taskName)
	if err != nil {
		log.Fatalf("Error ejecutando [%s]: %v", *taskName, err)
	}
}
