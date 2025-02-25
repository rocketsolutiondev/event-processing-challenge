CREATE TABLE IF NOT EXISTS exchange_rates (
    currency TEXT PRIMARY KEY,
    rate_to_eur DECIMAL NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Initial rates
INSERT INTO exchange_rates (currency, rate_to_eur) VALUES
    ('EUR', 1.0),
    ('USD', 0.85),
    ('GBP', 1.15),
    ('BTC', 35000),
    ('ETH', 2000),
    ('NZD', 0.57),
    ('AUD', 0.61),
    ('CAD', 0.69),
    ('CHF', 1.05),
    ('CNY', 0.13),
    ('JPY', 0.0075)
ON CONFLICT (currency) DO NOTHING; 