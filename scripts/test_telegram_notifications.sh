#!/bin/bash

# Script para probar las notificaciones de Telegram enviando eventos con datos erróneos

KAFKA_BROKER="${KAFKA_BROKER:-localhost:9092}"
TOPIC="${TOPIC:-events.default}"

echo "=========================================="
echo "Test de Notificaciones de Telegram"
echo "=========================================="
echo "Kafka Broker: $KAFKA_BROKER"
echo "Topic: $TOPIC"
echo ""

# Función para enviar un evento a Kafka
send_event() {
    local event_json="$1"
    local description="$2"

    echo "📤 Enviando: $description"
    echo "$event_json" | docker exec -i monitoring-energy-kafka kafka-console-producer \
        --broker-list "$KAFKA_BROKER" \
        --topic "$TOPIC" 2>/dev/null

    if [ $? -eq 0 ]; then
        echo "✅ Evento enviado exitosamente"
    else
        echo "❌ Error al enviar evento"
        echo "   Intentando con comando alternativo..."
        echo "$event_json" | kafka-console-producer.sh \
            --broker-list "$KAFKA_BROKER" \
            --topic "$TOPIC" 2>/dev/null
    fi

    echo "⏳ Esperando 3 segundos..."
    sleep 3
    echo ""
}

echo "Iniciando pruebas..."
echo ""

# Test 1: UUID Inválido
echo "Test 1: UUID Inválido"
echo "----------------------"
send_event '{
  "event_type": "power_reading",
  "plant_name": "Test Plant",
  "plant_source_id": "invalid-uuid-format",
  "power_generated_mw": 150.5,
  "power_consumed_mw": 10.2,
  "efficiency_percent": 85.3,
  "temperature_celsius": 32.1,
  "status": "operational"
}' "Evento con UUID inválido"

# Test 2: Campo plant_source_id faltante
echo "Test 2: Campo plant_source_id Faltante"
echo "---------------------------------------"
send_event '{
  "event_type": "status_update",
  "plant_name": "Another Test Plant",
  "power_generated_mw": 200.0,
  "status": "maintenance"
}' "Evento sin campo plant_source_id"

# Test 3: Planta inexistente (UUID válido pero no existe en DB)
echo "Test 3: Planta Inexistente"
echo "---------------------------"
send_event '{
  "event_type": "alert",
  "plant_name": "Non-Existent Plant",
  "plant_source_id": "00000000-0000-0000-0000-000000000000",
  "alert_type": "temperature_high",
  "temperature_celsius": 95.0
}' "Evento con planta que no existe en la base de datos"

# Test 4: Evento válido (no debería generar notificación)
echo "Test 4: Evento Válido (Sin Notificación)"
echo "----------------------------------------"
send_event '{
  "event_type": "power_reading",
  "plant_name": "Solar Plant Alpha",
  "plant_source_id": "1e2d3c4b-5a6f-7e8d-9c0b-1a2b3c4d5e6f",
  "power_generated_mw": 250.0,
  "power_consumed_mw": 15.0,
  "efficiency_percent": 90.5,
  "temperature_celsius": 28.5,
  "status": "operational"
}' "Evento válido (NO debería enviar notificación)"

echo "=========================================="
echo "✅ Pruebas completadas"
echo "=========================================="
echo ""
echo "Revisa tu bot de Telegram para ver las notificaciones."
echo "Deberías haber recibido 3 notificaciones:"
echo "  1. UUID Inválido"
echo "  2. Campo plant_source_id Faltante"
echo "  3. Planta Inexistente"
echo ""
echo "El último evento es válido y NO debería generar notificación."
echo ""
echo "Si no recibes las notificaciones, verifica:"
echo "  - Que el servicio esté ejecutándose"
echo "  - Que TELEGRAM_ENABLED=true en .env"
echo "  - Los logs del servicio para errores"
echo ""
