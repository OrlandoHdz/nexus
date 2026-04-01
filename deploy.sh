#!/bin/bash

# Configuración del servidor destino
SERVER="administrator@172.16.100.221"
DEST_DIR="/home/administrator/app/Nexus"

# Archivos a transferir
FILES=".env run_nexus.sh nexus-linux-amd64"

echo "🚀 Iniciando proceso de copiar archivos al servidor..."
echo "Destino: $SERVER:$DEST_DIR"

# Ejecutamos scp para transferir los archivos de forma segura. 
# La bandera -p mantiene los permisos originales de los archivos (ej. de ejecución)
scp -p $FILES $SERVER:$DEST_DIR/

if [ $? -eq 0 ]; then
    echo "✅ ¡Archivos subidos exitosamente!"
else
    echo "❌ Ocurrió un error al intentar subir los archivos. Revisa tu conexión."
    exit 1
fi
