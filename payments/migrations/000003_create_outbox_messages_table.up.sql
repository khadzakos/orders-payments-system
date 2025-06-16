CREATE TABLE IF NOT EXISTS outbox_messages (
    id UUID PRIMARY KEY,                                 -- Уникальный ID записи Outbox
    order_id UUID NOT NULL,                              -- ID заказа, к которому относится сообщение
    order_status VARCHAR(50) NOT NULL,                   -- Статус заказа
    payload JSONB NOT NULL,                              -- Содержимое сообщения в формате JSON
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',       -- Статус: PENDING, PROCESSED, FAILED
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- Время создания записи
    sent_at TIMESTAMP WITH TIME ZONE                     -- Время отправки сообщения
);

-- Индекс для быстрого поиска необработанных сообщений
CREATE INDEX IF NOT EXISTS idx_outbox_status_created_at ON outbox_messages (status, created_at);