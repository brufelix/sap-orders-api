INSERT INTO orders (id, order_number, status, created_by, created_at, updated_at)
VALUES
    (
        'a1111111-1111-4111-8111-111111111101',
        'PO-2026-001',
        'OPEN',
        'dev@brunodias.dev',
        NOW() - INTERVAL '7 days',
        NOW() - INTERVAL '1 day'
    ),
    (
        'a1111111-1111-4111-8111-111111111102',
        'PO-2026-002',
        'IN_PROGRESS',
        'dev@brunodias.dev',
        NOW() - INTERVAL '3 days',
        NOW()
    ),
    (
        'a1111111-1111-4111-8111-111111111103',
        'PO-2026-003',
        'CLOSED',
        'dev@brunodias.dev',
        NOW() - INTERVAL '30 days',
        NOW() - INTERVAL '2 days'
    );

INSERT INTO order_items (id, order_id, demand_code, description, delivery_date, status, created_at, updated_at)
VALUES
    (
        'b2222222-2222-4222-8222-222222222201',
        'a1111111-1111-4111-8111-111111111101',
        'DEM-1001',
        'Fornecimento de materiais de escritório',
        CURRENT_DATE + INTERVAL '15 days',
        'PENDING',
        NOW() - INTERVAL '7 days',
        NOW() - INTERVAL '7 days'
    ),
    (
        'b2222222-2222-4222-8222-222222222202',
        'a1111111-1111-4111-8111-111111111101',
        'DEM-1002',
        'Licenças de software corporativo',
        CURRENT_DATE + INTERVAL '30 days',
        'UPDATED',
        NOW() - INTERVAL '6 days',
        NOW() - INTERVAL '1 day'
    ),
    (
        'b2222222-2222-4222-8222-222222222203',
        'a1111111-1111-4111-8111-111111111102',
        'DEM-2001',
        'Manutenção preventiva de equipamentos',
        CURRENT_DATE + INTERVAL '10 days',
        'SYNCED',
        NOW() - INTERVAL '3 days',
        NOW() - INTERVAL '1 day'
    ),
    (
        'b2222222-2222-4222-8222-222222222204',
        'a1111111-1111-4111-8111-111111111102',
        'DEM-2002',
        'Treinamento técnico da equipe',
        CURRENT_DATE + INTERVAL '45 days',
        'PENDING',
        NOW() - INTERVAL '2 days',
        NOW() - INTERVAL '2 days'
    ),
    (
        'b2222222-2222-4222-8222-222222222205',
        'a1111111-1111-4111-8111-111111111103',
        'DEM-3001',
        'Consultoria de integração SAP',
        CURRENT_DATE - INTERVAL '5 days',
        'SYNCED',
        NOW() - INTERVAL '30 days',
        NOW() - INTERVAL '2 days'
    );
