package tasks

import (
	"archive/zip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/godror/godror"
	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
	"gopkg.in/gomail.v2"
)

type ReportForecastTask struct {
}

func (t *ReportForecastTask) Name() string {
	return "report-forecast-task"
}

func (t *ReportForecastTask) Execute() error {
	log.Printf("Iniciando generación de reporte a las %s...", time.Now().Format("15:04:05"))

	// Cargar variables de entorno
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Advertencia: No se pudo cargar el archivo .env o no existe: %v", err)
	}

	host := os.Getenv("ascp_host")
	user := os.Getenv("ascp_user")
	password := os.Getenv("ascp_password")
	sid := os.Getenv("ascp_sid")
	portStr := os.Getenv("ascp_port")

	// Configurar la cadena de conexión
	dsn := fmt.Sprintf(`user="%s" password="%s" connectString="%s:%s/%s"`, user, password, host, portStr, sid)

	fmt.Println("-> Conectando a la base de datos Oracle (ASCP)...")
	db, err := sql.Open("godror", dsn)
	if err != nil {
		return fmt.Errorf("error al conectar con oracle: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("error al hacer ping a oracle: %w", err)
	}
	fmt.Println("-> Conexión exitosa a ASCP.")

	// Parámetros de procedimiento obtenidos de las variables de entorno
	orgStr := os.Getenv("org")
	porg, err := strconv.Atoi(orgStr)
	if err != nil {
		return fmt.Errorf("error: la variable de entorno 'org' (%s) no es un número válido: %w", orgStr, err)
	}

	pplan := os.Getenv("plan_name")
	if pplan == "" {
		return fmt.Errorf("error: la variable de entorno 'plan_name' está vacía o no fue definida")
	}

	if err := executeProgramSalesSP(db, porg, pplan); err != nil {
		return err
	}

	fmt.Println("-> Extrayendo datos y generando archivo Excel...")

	now := time.Now()
	// Mes y año actual
	currentMonth := int(now.Month())
	currentYear := now.Year()

	// Para calcular el mes siguiente de manera segura y evitar saltos extra por meses de 31 días
	// (ej: si hoy es 31 de marzo, sumar 1 mes resulta en el 1 de mayo), usamos siempre el día 1.
	firstDayOfCurrentMonth := time.Date(currentYear, now.Month(), 1, 0, 0, 0, 0, time.UTC)
	nextMonthTime := firstDayOfCurrentMonth.AddDate(0, 1, 0)
	nextMonth := int(nextMonthTime.Month())
	nextYear := nextMonthTime.Year()

	querySelect := `
		SELECT * FROM smx_program_sales 
		WHERE ((month = :1 AND year = :2) OR (month = :3 AND year = :4))
		  AND quantity_rate > 0 
		ORDER BY new_due_date ASC`

	// Contexto independiente para la lectura y extracción de los datos (timeout de 10 min por si acaso)
	queryCtx, queryCancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer queryCancel()

	rows, err := db.QueryContext(queryCtx, querySelect, currentMonth, currentYear, nextMonth, nextYear)
	if err != nil {
		return fmt.Errorf("error al consultar smx_program_sales: %w", err)
	}
	defer rows.Close()

	// Obtener los nombres de las columnas para los encabezados
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("error al obtener columnas de la tabla: %w", err)
	}

	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("advertencia cerrando archivo Excel: %v\n", err)
		}
	}()
	sheet := "Sheet1"

	// Escribir los encabezados en la primera fila (row=1)
	for i, col := range columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, col)
	}

	// Iterar e inyectar valores en el Excel dinámicamente
	rowIndex := 2
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("error leyendo fila de smx_program_sales: %w", err)
		}

		for i, val := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, rowIndex)
			if val == nil {
				f.SetCellValue(sheet, cell, "")
				continue
			}

			switch v := val.(type) {
			case []byte:
				f.SetCellValue(sheet, cell, string(v))
			default:
				f.SetCellValue(sheet, cell, v)
			}
		}
		rowIndex++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterando resultados de db: %w", err)
	}

	fileName := fmt.Sprintf("Reporte_Ventas_%d_%02d.xlsx", currentYear, currentMonth)
	if err := f.SaveAs(fileName); err != nil {
		return fmt.Errorf("error al guardar el archivo Excel: %w", err)
	}

	fmt.Printf("-> Archivo '%s' finalizado exitosamente con %d registros de datos.\n", fileName, rowIndex-2)

	fmt.Println("-> Comprimiendo archivo Excel a ZIP...")
	zipFileName := fmt.Sprintf("Reporte_Ventas_%d_%02d.zip", currentYear, currentMonth)
	if err := zipExcelFile(fileName, zipFileName); err != nil {
		return fmt.Errorf("error al comprimir archivo excel en zip: %w", err)
	}
	fmt.Printf("-> Archivo '%s' guardado exitosamente.\n", zipFileName)

	fmt.Println("-> Enviando correo de notificación...")
	if err := sendEmailWithAttachment(zipFileName, currentMonth, currentYear); err != nil {
		return fmt.Errorf("error al enviar el correo: %w", err)
	}

	log.Println("Reporte finalizado y notificado con éxito.")
	return nil
}

// zipExcelFile toma el archivo generado y lo empaqueta en un ZIP homónimo
func zipExcelFile(sourceFile, zipName string) error {
	zipOut, err := os.Create(zipName)
	if err != nil {
		return err
	}
	defer zipOut.Close()

	zipWriter := zip.NewWriter(zipOut)
	defer zipWriter.Close()

	fileToZip, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = sourceFile
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	if _, err := io.Copy(writer, fileToZip); err != nil {
		return err
	}
	return nil
}

// sendEmailWithAttachment configura un cliente SMTP enviado el archivo ZIP adjunto
func sendEmailWithAttachment(attachmentPath string, month, year int) error {
	smtpHost := os.Getenv("smtp_host")
	smtpPortStr := os.Getenv("smtp_port")
	smtpUser := os.Getenv("smtp_user")
	smtpPass := os.Getenv("smtp_pass")
	emailFrom := os.Getenv("email_from")
	emailTo := os.Getenv("email_to")

	// Prevenir intentar enviarlo si faltan datos cardinales
	if smtpHost == "" || smtpPortStr == "" || emailFrom == "" || emailTo == "" {
		return fmt.Errorf("faltan variables de entorno SMTP requeridas (smtp_host, smtp_port, email_from, email_to)")
	}

	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		return fmt.Errorf("el puerto SMTP configurado no es un número válido: %w", err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", emailFrom)

	// Soportar múltiples destinatarios separados por coma
	recipients := strings.Split(emailTo, ",")
	for i := range recipients {
		recipients[i] = strings.TrimSpace(recipients[i])
	}
	m.SetHeader("To", recipients...)
	m.SetHeader("Subject", fmt.Sprintf("Reporte de Forecast ASCP - Mes %02d de %d", month, year))
	m.SetBody("text/plain", "Hola,\n\nAdjunto a este correo encontrará el reporte de forecast del mes actual y el mes siguiente comprimido en un archivo ZIP.\n\nSaludos.")
	m.Attach(attachmentPath)

	d := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPass)
	return d.DialAndSend(m)
}

// executeProgramSalesSP aísla la lógica y el timeout de 30 minutos necesarios para invocar el procedimiento almacenado principal
func executeProgramSalesSP(db *sql.DB, porg int, pplan string) error {
	fmt.Printf("-> Ejecutando procedimiento p_all_program_sales (porg=%d, pplan='%s')...\n", porg, pplan)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	query := `BEGIN xxsmx.xxsmx_sales_program.p_all_program_sales(:1, :2); END;`
	_, err := db.ExecContext(ctx, query, porg, pplan)
	if err != nil {
		return fmt.Errorf("error ejecutando procedimiento almacenado: %w", err)
	}

	fmt.Println("-> Procedimiento ejecutado correctamente.")
	return nil
}
