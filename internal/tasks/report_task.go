package tasks

import (
	"fmt"
	"log"
	"time"
)

// ReportTask define la estructura para tu nueva automatización
type ReportTask struct {
	// Aquí puedes añadir dependencias, ej: DB *sql.DB
}

// Name define el identificador que usarás en el flag de la terminal
func (t *ReportTask) Name() string {
	return "report-task"
}

// Execute contiene la lógica principal del reporte
func (t *ReportTask) Execute() error {
	log.Printf("Iniciando generación de reporte a las %s...", time.Now().Format("15:04:05"))

	// Simulación de lógica de negocio
	fmt.Println("-> Extrayendo datos de la base de datos...")
	fmt.Println("-> Generando archivo Excel...")
	fmt.Println("-> Enviando correo de notificación...")

	log.Println("Reporte finalizado con éxito.")
	return nil
}
