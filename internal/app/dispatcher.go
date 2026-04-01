package app

import (
	"fmt"

	"github.com/OrlandoHdz/nexus/internal/tasks"
)

func GetTasks() map[string]tasks.Task {
	// Instanciamos las tareas
	hello := &tasks.HelloTask{}
	report := &tasks.ReportTask{}                 // <--- Nueva instancia
	reportForecast := &tasks.ReportForecastTask{} // <--- Nueva instancia

	return map[string]tasks.Task{
		hello.Name():          hello,
		report.Name():         report,         // <--- Registro en el mapa
		reportForecast.Name(): reportForecast, // <--- Registro en el mapa
	}
}

func RunTask(taskName string) error {
	taskList := GetTasks()

	task, exists := taskList[taskName]
	if !exists {
		return fmt.Errorf("la tarea '%s' no existe en Nexus", taskName)
	}

	return task.Execute()
}
