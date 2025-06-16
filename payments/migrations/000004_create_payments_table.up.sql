CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY,                               -- Уникальный ID платежа
    order_id UUID NOT NULL,                            -- ID заказа, к которому относится платеж
    user_id BIGINT NOT NULL,                           -- ID пользователя, инициировавшего платеж
    amount DECIMAL(10, 4) NOT NULL,                    -- Сумма платежа
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',     -- Статус платежа: PENDING, COMPLETED, FAILED
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),  -- Время создания платежа
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()  -- Время последнего обновления платежа
);

-- Индекс для поиска платежей по ID заказа
CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments (order_id);

-- Индекс для поиска платежей по ID пользователя
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments (user_id);

-- Индекс для поиска платежей по статусу
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments (status);