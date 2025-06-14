CREATE TABLE IF NOT EXISTS outbox_messages (
    id VARCHAR(36) PRIMARY KEY,                               -- ID сообщения (UUID в виде строки)
    topic VARCHAR(255) NOT NULL,                              -- Топик Kafka для отправки
    payload JSONB NOT NULL,                                   -- Содержимое сообщения в JSONB
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',            -- Статус сообщения: 'PENDING', 'SENT'
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMP WITH TIME ZONE                          -- Время отправки (NULL, если не отправлено)
);

-- Индекс для быстрого поиска необработанных сообщений
CREATE INDEX IF NOT EXISTS idx_outbox_messages_status ON outbox_messages (status, created_at);