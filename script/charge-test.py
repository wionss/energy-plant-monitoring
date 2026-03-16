import json
import time
import random
import uuid
import argparse
from datetime import datetime, timedelta
from concurrent.futures import ThreadPoolExecutor
from confluent_kafka import Producer
from faker import Faker

# CONFIGURACIÓN
KAFKA_BROKER = "localhost:9092"
TOPIC = "intake"

VALID_PLANT_IDS = [
    {"id": "1e2d3c4b-5a6f-7e8d-9c0b-1a2b3c4d5e6f", "name": "Solar Plant Alpha"},
    {"id": "2f3e4d5c-6b7a-8c9d-0e1f-2b3c4d5e6f7a", "name": "Wind Farm Beta"},
    {"id": "c2e78b94-76f4-49a4-b6e8-d62c8d1d23ea", "name": "Hydro Plant Gamma"},
]

EVENT_TYPES = ["power_reading", "status_update", "efficiency_report", "alert"]
STATUSES = ["operational", "maintenance", "standby", "peak_load"]

fake = Faker()

def get_random_timestamp(days_back=90):
    """Genera un timestamp aleatorio en los últimos X días"""
    end = datetime.now()
    start = end - timedelta(days=days_back)
    random_date = start + (end - start) * random.random()
    return random_date.strftime("%Y-%m-%dT%H:%M:%SZ")

def generate_backfill_event():
    plant = random.choice(VALID_PLANT_IDS)
    
    # 10% de probabilidad de ser una alerta
    is_alert = random.random() < 0.1
    evt_type = "alert" if is_alert else random.choice(EVENT_TYPES)

    event = {
        "plant_id": f"plant-{plant['name'].split()[-1].lower()}",
        "plant_source_id": plant['id'],
        "plant_name": plant['name'],
        "event_type": evt_type,
        "power_generated_mw": round(random.uniform(0, 1000), 2),
        "power_consumed_mw": round(random.uniform(0, 50), 2),
        "efficiency_percent": round(random.uniform(75, 99), 2),
        "temperature_celsius": round(random.uniform(20, 50), 2),
        "status": "warning" if is_alert else random.choice(STATUSES),
        "timestamp": get_random_timestamp(days_back=90), # key: use a past timestamp
        "metadata": {
            "sensor_id": fake.uuid4(),
            "backfill": True
        }
    }
    return event

def produce_load(producer, num_events):
    for _ in range(num_events):
        event = generate_backfill_event()
        # Use plant_source_id as the partition key to guarantee ordering per plant
        # This ensures all events for the same plant go to the same partition
        key = str(event['plant_source_id'])
        payload = json.dumps(event)
        
        try:
            producer.produce(TOPIC, key=key, value=payload)
            producer.poll(0)
        except BufferError:
            producer.poll(1)

    producer.flush()

def main():
    parser = argparse.ArgumentParser(description="Backfill Generator")
    parser.add_argument("--events", type=int, default=500000, help="Total number of events")
    parser.add_argument("--workers", type=int, default=10, help="Number of parallel worker threads")
    args = parser.parse_args()

    conf = {
        'bootstrap.servers': KAFKA_BROKER,
        'client.id': 'backfill-producer',
        'queue.buffering.max.messages': 500000,
        'batch.num.messages': 10000, # Batch grande para velocidad
        'linger.ms': 100,
        'compression.type': 'lz4'
    }

    producer = Producer(conf)
    
    print(f"🚀 Generando {args.events} eventos históricos (últimos 90 días)...")
    start_time = time.time()

    events_per_worker = args.events // args.workers
    with ThreadPoolExecutor(max_workers=args.workers) as executor:
        futures = [executor.submit(produce_load, producer, events_per_worker) for _ in range(args.workers)]
        for f in futures: f.result()

    duration = time.time() - start_time
    print(f"✅ Terminado en {duration:.2f}s. Tasa: {args.events/duration:.0f} ev/s")

if __name__ == "__main__":
    main()