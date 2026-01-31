#!/usr/bin/env python3
"""
Real-time Event Sender - Un worker por planta, 1 evento/segundo via Kafka
Uso: python realtime-sender.py [--interval 1.0]
"""

import json
import time
import random
import uuid
import argparse
import threading
from datetime import datetime
from confluent_kafka import Producer

# CONFIGURACIÓN
KAFKA_BROKER = "localhost:9092"
TOPIC = "intake"

PLANTS = [
    {
        "id": "1e2d3c4b-5a6f-7e8d-9c0b-1a2b3c4d5e6f",
        "name": "Solar Plant Pasto",
        "type": "solar",
        "icon": "☀️",
        "base_gen": 150.0,
        "base_con": 10.0,
        "base_eff": 92.0,
        "base_temp": 28.0,
    },
    {
        "id": "2f3e4d5c-6b7a-8c9d-0e1f-2b3c4d5e6f7a",
        "name": "Wind Farm Cali",
        "type": "wind",
        "icon": "💨",
        "base_gen": 200.0,
        "base_con": 15.0,
        "base_eff": 88.0,
        "base_temp": 32.0,
    },
    {
        "id": "c2e78b94-76f4-49a4-b6e8-d62c8d1d23ea",
        "name": "Hydro Plant Bogotá",
        "type": "hydro",
        "icon": "💧",
        "base_gen": 300.0,
        "base_con": 20.0,
        "base_eff": 95.0,
        "base_temp": 18.0,
    },
]

stop_event = threading.Event()
counters = {plant["id"]: 0 for plant in PLANTS}
lock = threading.Lock()


def generate_event(plant: dict) -> dict:
    """Genera un evento con valores aleatorios basados en la planta."""
    variation = 0.1  # 10% variación

    power_gen = plant["base_gen"] * (1 + random.uniform(-variation, variation))
    power_con = plant["base_con"] * (1 + random.uniform(-variation, variation))
    efficiency = min(100, plant["base_eff"] * (1 + random.uniform(-variation/2, variation/2)))
    temp = plant["base_temp"] * (1 + random.uniform(-variation, variation))

    return {
        "plant_id": f"plant-{plant['type']}",
        "plant_source_id": plant["id"],
        "plant_name": plant["name"],
        "event_type": "power_reading",
        "power_generated_mw": round(power_gen, 2),
        "power_consumed_mw": round(power_con, 2),
        "efficiency_percent": round(efficiency, 2),
        "temperature_celsius": round(temp, 2),
        "status": "operational",
        "timestamp": datetime.now().strftime("%Y-%m-%dT%H:%M:%SZ"),
        "metadata": {
            "sensor_id": str(uuid.uuid4()),
            "plant_type": plant["type"],
            "realtime": True
        }
    }


def plant_worker(producer: Producer, plant: dict, interval: float):
    """Worker que envía eventos cada X segundos para una planta."""

    while not stop_event.is_set():
        try:
            event = generate_event(plant)
            key = str(uuid.uuid4())
            payload = json.dumps(event)

            producer.produce(TOPIC, key=key, value=payload)
            producer.poll(0)

            with lock:
                counters[plant["id"]] += 1
                count = counters[plant["id"]]

            print(f"{plant['icon']} [{count:>5}] {plant['name']:<25} | "
                  f"Gen: {event['power_generated_mw']:>6.1f} MW | "
                  f"Eff: {event['efficiency_percent']:>5.1f}%")

        except Exception as e:
            print(f"❌ {plant['name']} - Error: {e}")

        time.sleep(interval)


def main():
    parser = argparse.ArgumentParser(description="Real-time Event Sender")
    parser.add_argument("--interval", type=float, default=1.0, help="Segundos entre eventos (default: 1.0)")
    args = parser.parse_args()

    print(f"""
╔════════════════════════════════════════════════════════════╗
║        REAL-TIME EVENT SENDER - Kafka                      ║
╠════════════════════════════════════════════════════════════╣
║  Broker:    {KAFKA_BROKER:<44} ║
║  Topic:     {TOPIC:<44} ║
║  Plants:    {len(PLANTS):<44} ║
║  Interval:  {args.interval}s por planta                                   ║
╠════════════════════════════════════════════════════════════╣
║  Press Ctrl+C to stop                                      ║
╚════════════════════════════════════════════════════════════╝
""")

    conf = {
        'bootstrap.servers': KAFKA_BROKER,
        'client.id': 'realtime-producer',
        'queue.buffering.max.messages': 10000,
        'batch.num.messages': 100,
        'linger.ms': 10,
    }

    producer = Producer(conf)

    # Crear un thread por cada planta
    threads = []
    for plant in PLANTS:
        t = threading.Thread(
            target=plant_worker,
            args=(producer, plant, args.interval),
            daemon=True
        )
        t.start()
        threads.append(t)
        print(f"🚀 Worker started: {plant['icon']} {plant['name']}")

    print("\n" + "─" * 60)
    print("Sending events to Kafka...\n")

    try:
        while True:
            time.sleep(0.1)
    except KeyboardInterrupt:
        print("\n\n⏹️  Stopping workers...")
        stop_event.set()

        # Esperar a que terminen los threads
        for t in threads:
            t.join(timeout=2)

        # Flush final
        producer.flush()

        total = sum(counters.values())
        print(f"\n✅ Done! Total events sent: {total}")
        print("   Per plant:")
        for plant in PLANTS:
            print(f"   {plant['icon']} {plant['name']}: {counters[plant['id']]}")


if __name__ == "__main__":
    main()
