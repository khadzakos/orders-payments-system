CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- Уникальный идентификатор счета
    user_id BIGINT NOT NULL UNIQUE,                -- Идентификатор пользователя (не более одного счета на пользователя)
    balance NUMERIC(19, 4) NOT NULL DEFAULT 0.0000, -- Текущий баланс счета (19 знаков, 4 после запятой)
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Индекс для быстрого поиска счета по user_id
CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts (user_id);