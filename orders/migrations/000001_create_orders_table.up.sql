-- ENUM для статусов заказа
CREATE TYPE order_status_enum AS ENUM ('NEW', 'PENDING_PAYMENT', 'FINISHED', 'CANCELLED');

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- Уникальный идентификатор заказа
    user_id BIGINT NOT NULL,                       -- Идентификатор пользователя
    amount NUMERIC(19, 4) NOT NULL,                -- Общая сумма заказа (19 знаков, 4 после запятой для точности)
    description TEXT,                              -- Описание заказа
    status order_status_enum NOT NULL DEFAULT 'NEW',            -- Текущий статус заказа (NEW, PENDING_PAYMENT, FINISHED, CANCELLED)
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(), -- Время создания, с учетом часового пояса
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()  -- Время обновления, с учетом часового пояса
);

-- Индекс для быстрого поиска заказов по пользователю
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders (user_id);

-- Индекс для быстрого поиска заказов по статусу
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);