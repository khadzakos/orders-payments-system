CREATE TABLE IF NOT EXISTS inbox_messages (
    id UUID PRIMARY KEY,                                    -- Уникальный ID записи Inbox
    kafka_topic VARCHAR(255) NOT NULL,                      -- Топик Kafka
    kafka_partition INT NOT NULL,                           -- Партиция Kafka
    kafka_offset BIGINT NOT NULL,                           -- Смещение Kafka
    consumer_group VARCHAR(255) NOT NULL,                   -- Потребительская группа
    payload JSONB NOT NULL,                                 -- Содержимое сообщения
    status VARCHAR(50) NOT NULL DEFAULT 'NEW',              -- Статус: NEW, PROCESSING, PROCESSED, FAILED
    received_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE                   -- Время завершения обработки

    CONSTRAINT uq_inbox_kafka_id_group UNIQUE (kafka_topic, kafka_partition, kafka_offset, consumer_group)
);

-- Индекс для быстрого поиска необработанных сообщений
CREATE INDEX IF NOT EXISTS idx_inbox_status_received_at ON inbox_messages (status, received_at);