# Configuración de Notificaciones de Telegram

Este documento explica cómo configurar las notificaciones de Telegram para recibir alertas de errores de validación de datos en el servicio de monitoreo de energía.

## Datos de Configuración

Basado en el archivo `teleevents.py`, ya tienes un bot de Telegram configurado con los siguientes datos:

```bash
TELEGRAM_ENABLED=true
TELEGRAM_BOT_TOKEN=8372010556:AAHMS-2wvOkbomXvQroM__6zRb3RyMnLyP0
TELEGRAM_CHAT_ID=8535154955
```

## Pasos para Configurar

### 1. Actualizar el archivo .env

Agrega o actualiza las siguientes variables en tu archivo `.env`:

```bash
# Telegram Notifications
TELEGRAM_ENABLED=true
TELEGRAM_BOT_TOKEN=8372010556:AAHMS-2wvOkbomXvQroM__6zRb3RyMnLyP0
TELEGRAM_CHAT_ID=8535154955
```

### 2. Verificar que el bot esté activo

1. Abre Telegram
2. Busca tu bot usando el nombre que le diste
3. Envía el comando `/start` para activar el bot
4. El bot debería responder confirmando que está activo

### 3. Iniciar el servicio

Una vez configuradas las variables de entorno, inicia el servicio:

```bash
make dev
```

o

```bash
make run
```

## Tipos de Notificaciones

El servicio enviará notificaciones automáticamente cuando ocurran los siguientes errores:

### 1. UUID Inválido

Cuando el campo `plant_source_id` no tiene un formato de UUID válido:

```
🚨 Error de Validación

⏰ Hora: 2026-01-11 15:30:45
🔴 Tipo: UUID Inválido
📝 Mensaje: Error al parsear UUID: invalid UUID format
📋 Contexto: Campo: plant_source_id, UUID inválido: abc-123-invalid

🏢 Servicio: Monitoring Energy Service
```

### 2. Campo Faltante

Cuando el campo `plant_source_id` no está presente en el mensaje:

```
🚨 Error de Validación

⏰ Hora: 2026-01-11 15:31:20
🔴 Tipo: Validación de Datos
📝 Mensaje: El campo plant_source_id no está presente en el mensaje
📋 Contexto: Campo: plant_source_id, Valor: campo ausente

🏢 Servicio: Monitoring Energy Service
```

### 3. Planta No Existe

Cuando se intenta registrar un evento para una planta que no existe en la base de datos:

```
🚨 Error de Validación

⏰ Hora: 2026-01-11 15:32:10
🔴 Tipo: Validación de Datos
📝 Mensaje: La planta con ID 1e2d3c4b-5a6f-7e8d-9c0b-000000000000 no existe en la base de datos
📋 Contexto: Campo: plant_source_id, Valor: 1e2d3c4b-5a6f-7e8d-9c0b-000000000000

🏢 Servicio: Monitoring Energy Service
```

## Desactivar Notificaciones

Si deseas desactivar las notificaciones de Telegram sin eliminar la configuración:

```bash
TELEGRAM_ENABLED=false
```

## Pruebas

Para probar que las notificaciones funcionan correctamente, puedes enviar un evento con datos inválidos al topic de Kafka:

```bash
# Evento con UUID inválido
echo '{
  "event_type": "power_reading",
  "plant_name": "Test Plant",
  "plant_source_id": "invalid-uuid",
  "data": {
    "power": 100
  }
}' | kafka-console-producer --broker-list localhost:9092 --topic events.default
```

Deberías recibir una notificación en Telegram inmediatamente después de que el servicio procese el mensaje.

## Solución de Problemas

### No recibo notificaciones

1. **Verifica que el servicio esté ejecutándose:**
   ```bash
   ps aux | grep monitoring-energy
   ```

2. **Verifica los logs del servicio:**
   ```bash
   # Busca mensajes como "Telegram notification sent successfully"
   # o "Failed to send Telegram notification"
   ```

3. **Verifica que las variables de entorno estén cargadas:**
   - El servicio debería mostrar las variables al inicio (el token aparecerá enmascarado por seguridad)
   - Ejemplo: `TELEGRAM_BOT_TOKEN=83**************************P0`

4. **Verifica la conectividad:**
   ```bash
   curl -X GET "https://api.telegram.org/bot8372010556:AAHMS-2wvOkbomXvQroM__6zRb3RyMnLyP0/getMe"
   ```

### El bot no responde

1. Verifica que hayas enviado `/start` al bot
2. Verifica que el `TELEGRAM_CHAT_ID` sea correcto
3. Verifica que el token del bot sea válido

## Seguridad

⚠️ **IMPORTANTE:**

- **NUNCA** compartas el token del bot públicamente
- **NUNCA** commits el token en repositorios públicos
- Usa variables de entorno en producción
- Considera rotar el token periódicamente mediante BotFather

Para regenerar el token si fue comprometido:
1. Habla con [@BotFather](https://t.me/botfather)
2. Envía `/token`
3. Selecciona tu bot
4. Actualiza la variable `TELEGRAM_BOT_TOKEN` con el nuevo token
