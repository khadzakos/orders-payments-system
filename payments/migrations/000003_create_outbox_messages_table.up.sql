CREATE TABLE IF NOT EXISTS outbox_messages (
    id UUID PRIMARY KEY,                               -- Уникальный ID записи Outbox
    aggregate_id UUID NOT NULL,                        -- ID бизнес-сущности (например, OrderID, PaymentID)
    aggregate_type VARCHAR(255) NOT NULL,              -- Тип бизнес-сущности (например, 'Payment', 'Order')
    message_type VARCHAR(255) NOT NULL,                -- Тип события (например, 'PaymentProcessedEvent')
    topic VARCHAR(255) NOT NULL,                       -- Целевой топик Kafka
    key_value VARCHAR(255) NOT NULL,                   -- Ключ сообщения Kafka
    payload JSONB NOT NULL,                            -- Содержимое сообщения
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',     -- Статус: PENDING, SENT, FAILED
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMP WITH TIME ZONE                   -- Время отправки
);

-- Индекс для быстрого поиска необработанных сообщений
CREATE INDEX IF NOT EXISTS idx_outbox_status_created_at ON outbox_messages (status, created_at);

-- Индекс для поиска сообщений по aggregate_id (для отладки)
CREATE INDEX IF NOT EXISTS idx_outbox_aggregate_id ON outbox_messages (aggregate_id);