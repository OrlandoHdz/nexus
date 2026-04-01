#!/bin/bash

# Forzamos detener la ejecución si ocurre algún error
set -e

# Exportamos las variables de entorno donde se encuentra el Oracle Instant Client
export ORACLE_HOME=/home/administrator/instantclient_21_1
export DYLD_LIBRARY_PATH=/home/administrator/instantclient_21_1
export LD_LIBRARY_PATH=/home/administrator/instantclient_21_1
export VERSION=11.2.0.4.0

cd /home/administrator/app/Nexus

# Ejecutar el binario
./nexus-linux-amd64 -task=report-forecast-task
