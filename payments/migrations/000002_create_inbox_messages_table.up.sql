CREATE TABLE IF NOT EXISTS inbox_messages (
    id UUID PRIMARY KEY,                                    -- Уникальный ID записи Inbox
    order_id UUID NOT NULL,                                 -- ID бизнес-сущности (например, OrderID, PaymentID)
    payload JSONB NOT NULL,                                 -- Содержимое сообщения
    status VARCHAR(50) NOT NULL DEFAULT 'NEW',              -- Статус: NEW, PROCESSING, PROCESSED, FAILED
    received_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE                   -- Время завершения обработки
);

-- Индекс для быстрого поиска необработанных сообщений
CREATE INDEX IF NOT EXISTS idx_inbox_status_received_at ON inbox_messages (status, received_at);